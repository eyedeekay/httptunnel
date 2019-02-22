package i2phttpproxy

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

import (
	"github.com/eyedeekay/goSam"
	"github.com/eyedeekay/sam3"
)

type SAMHTTPProxy struct {
	gosam              *goSam.Client
	Client             *http.Client
	samcon             *sam3.SAM
	keys               sam3.I2PKeys
	stream             *sam3.StreamSession
	SamHost            string
	SamPort            string
	TunName            string
	inLength           uint
	outLength          uint
	inVariance         int
	outVariance        int
	inQuantity         uint
	outQuantity        uint
	inBackups          uint
	outBackups         uint
	dontPublishLease   bool
	encryptLease       bool
	reduceIdle         bool
	reduceIdleTime     uint
	reduceIdleQuantity uint
	compression        bool

	debug bool
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

var hopHeaders = []string{
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Proxy-Connection",
	"X-Forwarded-For",
	"Accept-Language",
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
	if header.Get("User-Agent") != "MYOB/6.66 (AN/ON)" {
		header.Set("User-Agent", "MYOB/6.66 (AN/ON)")
	}
}

func (p *SAMHTTPProxy) freshClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial:                  p.gosam.Dial,
			MaxIdleConns:          0,
			MaxIdleConnsPerHost:   3,
			DisableKeepAlives:     false,
			ResponseHeaderTimeout: time.Second * 600,
			ExpectContinueTimeout: time.Second * 600,
			IdleConnTimeout:       time.Second * 600,
			TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: nil,
		Timeout:       time.Second * 600,
	}

}

func (p *SAMHTTPProxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, " ", req.Method, " ", req.URL)

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" && !strings.HasSuffix(req.URL.Host, ".i2p") {
		msg := "unsupported protocal scheme " + req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	if req.Method == http.MethodConnect {
		log.Println("Connecting tunnel")
		p.connect(wr, req)
	} else {
		req.RequestURI = ""
		delHopHeaders(req.Header)
		p.get(wr, req)
	}

}

func (p *SAMHTTPProxy) connect(wr http.ResponseWriter, req *http.Request) {
	dest_conn, err := p.stream.Dial("tcp", req.Host)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusServiceUnavailable)
		return
	}
	wr.WriteHeader(http.StatusOK)
	hijacker, ok := wr.(http.Hijacker)
	if !ok {
		http.Error(wr, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(wr, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func (p *SAMHTTPProxy) get(wr http.ResponseWriter, req *http.Request) {
	Client := p.freshClient()
	resp, err := Client.Do(req)
	if err != nil {
		log.Println("ServeHTTP:", err)
		return
	}
	defer resp.Body.Close()

	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func (p *SAMHTTPProxy) sam3Args() []string {
	return []string{
		p.inlength(),
		p.outlength(),
		p.invariance(),
		p.outvariance(),
		p.inquantity(),
		p.outquantity(),
		p.inbackups(),
		p.outbackups(),
		p.dontpublishlease(),
		p.encryptlease(),
		p.reduceonidle(),
		p.reduceidletime(),
		p.reduceidlecount(),
		"i2cp.gzip=true",
	}
}

func NewHttpProxy(opts ...func(*SAMHTTPProxy) error) (*SAMHTTPProxy, error) {
	var handler SAMHTTPProxy
	handler.SamHost = "127.0.0.1"
	handler.SamPort = "7656"
	handler.TunName = "sam-http-proxy"
	handler.inLength = 2
	handler.outLength = 2
	handler.inVariance = 0
	handler.outVariance = 0
	handler.inQuantity = 1
	handler.outQuantity = 1
	handler.inBackups = 1
	handler.outBackups = 1
	handler.dontPublishLease = true
	handler.encryptLease = false
	handler.reduceIdle = false
	handler.reduceIdleTime = 2000000
	handler.reduceIdleQuantity = 1
	for _, o := range opts {
		if err := o(&handler); err != nil {
			return nil, err
		}
	}
	var err error
	handler.gosam, err = goSam.NewClientFromOptions(
		goSam.SetHost(handler.SamHost),
		goSam.SetPort(handler.SamPort),
		goSam.SetUnpublished(handler.dontPublishLease),
		goSam.SetInLength(handler.inLength),
		goSam.SetOutLength(handler.outLength),
		goSam.SetInQuantity(handler.inQuantity),
		goSam.SetOutQuantity(handler.outQuantity),
		goSam.SetInBackups(handler.inBackups),
		goSam.SetOutBackups(handler.outBackups),
		goSam.SetReduceIdle(handler.reduceIdle),
		goSam.SetReduceIdleTime(handler.reduceIdleTime),
		goSam.SetReduceIdleQuantity(handler.reduceIdleQuantity),
	)
	if err != nil {
		return nil, err
	}
	handler.samcon, err = sam3.NewSAM(handler.SamHost + ":" + handler.SamPort)
	if err != nil {
		return nil, err
	}
	handler.keys, err = handler.samcon.NewKeys()
	if err != nil {
		return nil, err
	}
	handler.stream, err = handler.samcon.NewStreamSession("sam-http-connector", handler.keys, handler.sam3Args())
	if err != nil {
		return nil, err
	}
	handler.Client = &http.Client{
		Transport: &http.Transport{
			Dial:                  handler.gosam.Dial,
			MaxIdleConns:          0,
			MaxIdleConnsPerHost:   3,
			DisableKeepAlives:     false,
			ResponseHeaderTimeout: time.Second * 600,
			ExpectContinueTimeout: time.Second * 600,
			IdleConnTimeout:       time.Second * 600,
			TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: nil,
		Timeout:       time.Second * 600,
	}
	return &handler, nil
}
