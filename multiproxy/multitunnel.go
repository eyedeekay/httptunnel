package i2pbrowserproxy

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

import (
	"github.com/eyedeekay/goSam"
	"github.com/eyedeekay/goSam/compat"
	"github.com/eyedeekay/httptunnel/common"
	"github.com/eyedeekay/sam-forwarder/i2pkeys"
	"github.com/eyedeekay/sam-forwarder/interface"
	"github.com/eyedeekay/sam3/i2pkeys"
)

type samClient struct {
	goSam       *goSam.Client
	client      *http.Client
	transport   *http.Transport
	rateLimiter *rate.Limiter
}

type SAMMultiProxy struct {
	clients            map[string]*samClient
	tunName            string
	sigType            string
	proxyHost          string
	proxyPort          string
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
	up     bool
}

var Quiet bool

func (f *SAMMultiProxy) findClient() *samClient {
	return f.clients["general"]
}

func plog(in ...interface{}) {
	if !Quiet {
		log.Println(in...)
	}
}

func (f *SAMMultiProxy) print() []string {
	return strings.Split(f.Print(), " ")
}

func (f *SAMMultiProxy) GetType() string {
	return "httpclient"
}

func (f *SAMMultiProxy) ID() string {
	return f.tunName
}

func (f *SAMMultiProxy) Keys() i2pkeys.I2PKeys {
	k, _ := samkeys.DestToKeys(f.findClient().goSam.Destination())
	return k
}

func (f *SAMMultiProxy) Props() map[string]string {
	r := make(map[string]string)
	for _, prop := range f.print() {
		k, v := sfi2pkeys.Prop(prop)
		r[k] = v
	}
	return r
}

func (p *SAMMultiProxy) Cleanup() {
	p.Close()
}

func (p *SAMMultiProxy) Print() string {
	return p.findClient().goSam.Print()
}

func (p *SAMMultiProxy) Search(search string) string {
	terms := strings.Split(search, ",")
	if search == "" {
		return p.Print()
	}
	for _, value := range terms {
		if !strings.Contains(p.Print(), value) {
			return ""
		}
	}
	return p.Print()
}

func (p *SAMMultiProxy) Target() string {
	return p.proxyHost + ":" + p.proxyPort
}

func (p *SAMMultiProxy) Base32() string {
	return p.findClient().goSam.Base32()
}

func (p *SAMMultiProxy) Base64() string {
	return p.findClient().goSam.Base64()
}

func (p *SAMMultiProxy) Serve() error {
	ln, err := net.Listen("tcp", p.proxyHost+":"+p.proxyPort)
	if err != nil {
		return err
	}
	srv := &http.Server{
		ReadTimeout:  600 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         ln.Addr().String(),
	}
	srv.Handler = p
	if err != nil {
		return err
	}
	log.Println("Starting proxy server on", ln.Addr())
	if err := srv.Serve(ln); err != nil {
		if err == http.ErrServerClosed {
			return err
		}
	}
	log.Println("Stopping proxy server on", ln.Addr())
	return nil
}

func (p *SAMMultiProxy) Close() error {
	p.up = false
	return p.findClient().goSam.Close()
}

func (p *SAMMultiProxy) freshTransport() *http.Transport {
	t := http.Transport{
		DialContext:           p.findClient().goSam.DialContext,
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

func (p *SAMMultiProxy) freshClient() *http.Client {
	return &http.Client{
		Transport:     p.findClient().transport,
		Timeout:       time.Second * 300,
		CheckRedirect: nil,
	}
}

func (p *SAMMultiProxy) freshSAMClient() (*goSam.Client, error) {
	return p.findClient().goSam.NewClient()
}

//return the combined host:port of the SAM bridge
func (p *SAMMultiProxy) samaddr() string {
	return fmt.Sprintf("%s:%s", p.SamHost, p.SamPort)
}

func (p *SAMMultiProxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	plog(req.RemoteAddr, " ", req.Method, " ", req.URL)
	p.Save()
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		if !(req.Method == http.MethodConnect) {
			msg := "Unsupported protocol scheme " + req.URL.Scheme
			http.Error(wr, msg, http.StatusBadRequest)
			plog(msg)
			return
		}
	}

	if !strings.HasSuffix(req.URL.Host, ".i2p") {
		if req.URL.Host == p.controlHost+":"+p.controlPort {
			p.reset(wr, req)
			return
		}
		msg := "Unsupported host " + req.URL.Host
		if !Quiet {
			http.Error(wr, msg, http.StatusBadRequest)
		}
		plog(msg)
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

func (p *SAMMultiProxy) reset(wr http.ResponseWriter, req *http.Request) {
	plog("Validating control access from", req.RemoteAddr, p.controlHost+":"+p.controlPort)
	if strings.SplitN(req.RemoteAddr, ":", 2)[0] == p.controlHost {
		plog("Validated control access from", req.RemoteAddr, p.controlHost+":"+p.controlPort)
		resp, err := http.Get("http://" + p.controlHost + ":" + p.controlPort)
		if err == nil {
			wr.Header().Set("Content-Type", "text/html; charset=utf-8")
			wr.Header().Set("Access-Control-Allow-Origin", "*")
			wr.WriteHeader(resp.StatusCode)
			io.Copy(wr, resp.Body)
			return
		}
	}
}

func (p *SAMMultiProxy) get(wr http.ResponseWriter, req *http.Request) {
	req.RequestURI = ""
	proxycommon.DelHopHeaders(req.Header)
	p.findClient().client = p.freshClient()
	resp, err := p.findClient().client.Do(req)
	if err != nil {
		msg := "Proxy Error " + err.Error()
		if !Quiet {
			http.Error(wr, msg, http.StatusBadRequest)
		}
		plog(msg)
		return
	}
	defer resp.Body.Close()

	proxycommon.CopyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func (p *SAMMultiProxy) connect(wr http.ResponseWriter, req *http.Request) {
	plog("CONNECT via i2p to", req.URL.Host)
	dest_conn, err := p.findClient().goSam.Dial("tcp", req.URL.Host)
	if err != nil {
		if !Quiet {
			http.Error(wr, err.Error(), http.StatusServiceUnavailable)
		}
		return
	}
	wr.WriteHeader(http.StatusOK)
	hijacker, ok := wr.(http.Hijacker)
	if !ok {
		if !Quiet {
			http.Error(wr, "Hijacking not supported", http.StatusInternalServerError)
		}
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		if !Quiet {
			http.Error(wr, err.Error(), http.StatusServiceUnavailable)
		}
		return
	}
	go proxycommon.Transfer(dest_conn, client_conn)
	go proxycommon.Transfer(client_conn, dest_conn)
}

func (f *SAMMultiProxy) Up() bool {
	return f.up
}

func (p *SAMMultiProxy) Save() string {
	if p.keyspath != "invalid.tunkey" {
		if _, err := os.Stat(p.keyspath); os.IsNotExist(err) {
			if p.findClient().goSam != nil {
				if p.findClient().goSam.Destination() != "" {
					ioutil.WriteFile(p.keyspath, []byte(p.findClient().goSam.Destination()), 0644)
					p.destination = p.findClient().goSam.Destination()
					return p.findClient().goSam.Destination()
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

func (handler *SAMMultiProxy) Load() (samtunnel.SAMTunnel, error) {
	var err error
	handler.destination = handler.Save()
	handler.findClient().goSam, err = goSam.NewClientFromOptions(
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
	handler.findClient().transport = handler.freshTransport()
	handler.findClient().client = handler.freshClient()
	handler.up = true
	return handler, nil
}

func NewHttpProxy(opts ...func(*SAMMultiProxy) error) (*SAMMultiProxy, error) {
	var handler SAMMultiProxy
	handler.SamHost = "127.0.0.1"
	handler.SamPort = "7656"
	handler.controlHost = "127.0.0.1"
	handler.controlPort = "7951"
	handler.proxyHost = "127.0.0.1"
	handler.proxyPort = "7950"
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
	handler.tunName = "0"
	handler.keyspath = "invalid.tunkey"
	handler.destination = ""
	for _, o := range opts {
		if err := o(&handler); err != nil {
			return nil, err
		}
	}
	l, e := handler.Load()
	if e != nil {
		return nil, e
	}
	return l.(*SAMMultiProxy), nil
}
