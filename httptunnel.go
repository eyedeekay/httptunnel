package i2phttpproxy

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

import (
	"github.com/eyedeekay/goSam"
	"github.com/eyedeekay/httptunnel/common"
)

type SAMHTTPProxy struct {
	goSam              *goSam.Client
	client             *http.Client
	transport          *http.Transport
	rateLimiter        *rate.Limiter
	id                 int32
	SamHost            string
	SamPort            string
	controlHost        string
	controlPort        string
	destination        string
	keyspath           string
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

	dialed bool
	debug  bool
}

func (p *SAMHTTPProxy) freshTransport() *http.Transport {
	t := http.Transport{
		DialContext:           p.goSam.DialContext,
		MaxConnsPerHost:       1,
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   1,
		DisableKeepAlives:     false,
		ResponseHeaderTimeout: time.Second * 600,
		IdleConnTimeout:       time.Second * 300,
		TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	return &t
}

func (p *SAMHTTPProxy) freshClient() *http.Client {
	return &http.Client{
		Transport:     p.transport,
		Timeout:       time.Second * 300,
		CheckRedirect: nil,
	}
}

func (p *SAMHTTPProxy) freshSAMClient() (*goSam.Client, error) {
	return p.goSam.NewClient()
}

//return the combined host:port of the SAM bridge
func (p *SAMHTTPProxy) samaddr() string {
	return fmt.Sprintf("%s:%s", p.SamHost, p.SamPort)
}

func (p *SAMHTTPProxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, " ", req.Method, " ", req.URL)
	p.Save()
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		if !(req.Method == http.MethodConnect) {
			msg := "Unsupported protocol scheme " + req.URL.Scheme
			http.Error(wr, msg, http.StatusBadRequest)
			log.Println(msg)
			return
		}
	}

	if !strings.HasSuffix(req.URL.Host, ".i2p") {
		if req.URL.Host == p.controlHost+":"+p.controlPort {
			p.reset(wr, req)
			return
		}
		msg := "Unsupported host " + req.URL.Host
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	if req.Method != http.MethodConnect {
		p.get(wr, req)
		return
	} else {
		p.connect(wr, req)
		return
	}

}

func (p *SAMHTTPProxy) reset(wr http.ResponseWriter, req *http.Request) {
	log.Println("Validating control access from", req.RemoteAddr, p.controlHost+":"+p.controlPort)
	if strings.SplitN(req.RemoteAddr, ":", 2)[0] == p.controlHost {
		log.Println("Validated control access from", req.RemoteAddr, p.controlHost+":"+p.controlPort)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil {
			wr.Header().Set("Content-Type", "text/html; charset=utf-8")
			wr.Header().Set("Access-Control-Allow-Origin", "*")
			wr.WriteHeader(resp.StatusCode)
			io.Copy(wr, resp.Body)
			return
		}
	}
}

func (p *SAMHTTPProxy) get(wr http.ResponseWriter, req *http.Request) {
	req.RequestURI = ""
	proxycommon.DelHopHeaders(req.Header)
	p.client = p.freshClient()
	resp, err := p.client.Do(req)
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
	dest_conn, err := p.goSam.Dial("tcp", req.URL.Host)
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

func (p *SAMHTTPProxy) Close() error {
	return p.goSam.Close()
}

func (p *SAMHTTPProxy) Save() string {
	if p.keyspath != "invalid.tunkey" {
		if _, err := os.Stat(p.keyspath); os.IsNotExist(err) {
			if p.goSam != nil {
				if p.goSam.Destination() != "" {
					ioutil.WriteFile(p.keyspath, []byte(p.goSam.Destination()), 0644)
					p.destination = p.goSam.Destination()
					return p.goSam.Destination()
				}
			}
		} else {
			if keys, err := ioutil.ReadFile(p.keyspath); err == nil {
				p.destination = string(keys)
				return string(keys)
			}
		}
	}
	return ""
}

func NewHttpProxy(opts ...func(*SAMHTTPProxy) error) (*SAMHTTPProxy, error) {
	var handler SAMHTTPProxy
	handler.SamHost = "127.0.0.1"
	handler.SamPort = "7656"
	handler.controlHost = "127.0.0.1"
	handler.controlPort = "7951"
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
	handler.closeIdleTime = 3000000
	handler.reduceIdleQuantity = 1
	handler.useOutProxy = false
	handler.compression = true
	handler.id = 0
	handler.keyspath = "invalid.tunkey"
	handler.destination = ""
	for _, o := range opts {
		if err := o(&handler); err != nil {
			return nil, err
		}
	}
	var err error
	handler.destination = handler.Save()
	handler.goSam, err = goSam.NewClientFromOptions(
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
		goSam.SetLocalDestination(handler.destination),
	)
	if err != nil {
		return nil, err
	}
	handler.transport = handler.freshTransport()
	handler.client = handler.freshClient()
	return &handler, nil
}
