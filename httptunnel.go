package i2phttpproxy

import (
	//"crypto/tls"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

import (
	"github.com/eyedeekay/goSam"
    "github.com/eyedeekay/httptunnel/common"
)

type SAMHTTPProxy struct {
	gosam              *goSam.Client
	Client             *http.Client
	SamHost            string
	SamPort            string
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
	closeIdle          bool
	closeIdleTime      uint
	compression        bool

	useOutProxy bool

	debug bool
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
			//TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			//TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
		Timeout:       time.Second * 600,
		CheckRedirect: nil,
	}
}

func (p *SAMHTTPProxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, " ", req.Method, " ", req.URL)

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		if !(req.Method == http.MethodConnect) {
			msg := "Unsupported protocol scheme " + req.URL.Scheme
			http.Error(wr, msg, http.StatusBadRequest)
			log.Println(msg)
			return
		}
	}

	log.Println(req.URL.Host)

	if !strings.HasSuffix(req.URL.Host, ".i2p") {
		msg := "Unsupported host " + req.URL.Host
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	if req.Method == http.MethodConnect {
		p.connect(wr, req)
		return
	} else {
		p.get(wr, req)
		return
	}

}

func (p *SAMHTTPProxy) get(wr http.ResponseWriter, req *http.Request) {
	req.RequestURI = ""
	proxycommon.DelHopHeaders(req.Header)
	resp, err := p.Client.Do(req)
	if err != nil {
		msg := "Proxy Error " + err.Error()
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}
	defer resp.Body.Close()

	proxycommon.CopyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func (p *SAMHTTPProxy) connect(wr http.ResponseWriter, req *http.Request) {
	log.Println("CONNECT via i2p to", req.URL.Host)
	dest_conn, err := p.gosam.Dial("tcp", req.URL.Host)
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
		return
	}
	go proxycommon.Transfer(dest_conn, client_conn)
	go proxycommon.Transfer(client_conn, dest_conn)
}

func NewHttpProxy(opts ...func(*SAMHTTPProxy) error) (*SAMHTTPProxy, error) {
	var handler SAMHTTPProxy
	handler.SamHost = "127.0.0.1"
	handler.SamPort = "7656"
	handler.inLength = 2
	handler.outLength = 2
	handler.inVariance = 0
	handler.outVariance = 0
	handler.inQuantity = 2
	handler.outQuantity = 2
	handler.inBackups = 1
	handler.outBackups = 1
	handler.dontPublishLease = true
	handler.encryptLease = false
	handler.reduceIdle = false
	handler.reduceIdleTime = 2000000
	handler.closeIdleTime = 3000000
	handler.reduceIdleQuantity = 1
	handler.useOutProxy = false
	handler.compression = true
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
		goSam.SetCloseIdle(handler.closeIdle),
		goSam.SetCloseIdleTime(handler.closeIdleTime),
		goSam.SetCompression(handler.compression),
		goSam.SetDebug(handler.debug),
	)
	if err != nil {
		return nil, err
	}
	handler.Client = handler.freshClient()
	return &handler, nil
}
