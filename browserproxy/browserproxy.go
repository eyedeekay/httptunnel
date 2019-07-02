package i2pbrowserproxy

import (
	//	"crypto/tls"
	"fmt"
	//	"golang.org/x/time/rate"
	//  "io"
	//	"io/ioutil"
	"log"
	"net"
	"net/http"
	//  "os"
	"strings"
	"time"
)

import (
	//  "github.com/eyedeekay/goSam"
	"github.com/eyedeekay/goSam/compat"
	"github.com/eyedeekay/httptunnel"
	//	"github.com/eyedeekay/httptunnel/common"
	"github.com/eyedeekay/sam-forwarder/i2pkeys"
	"github.com/eyedeekay/sam-forwarder/interface"
	"github.com/eyedeekay/sam3/i2pkeys"
)

type SAMBrowserHTTPProxy struct {
	proxyList          map[string]*i2phttpproxy.SAMHTTPProxy
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

func plog(in ...interface{}) {
	if !Quiet {
		log.Println(in...)
	}
}

func (f *SAMBrowserHTTPProxy) print() []string {
	return strings.Split(f.Print(), " ")
}

func (f *SAMBrowserHTTPProxy) GetType() string {
	return "httpclient"
}

func (f *SAMBrowserHTTPProxy) ID() string {
	return f.tunName
}

func (f *SAMBrowserHTTPProxy) ControlAddr() string {
	return f.controlHost + ":" + f.controlPort
}

func (f *SAMBrowserHTTPProxy) Keys() i2pkeys.I2PKeys {
	k, _ := samkeys.DestToKeys(null)
	return k
}

func (f *SAMBrowserHTTPProxy) Props() map[string]string {
	r := make(map[string]string)
	for _, prop := range f.print() {
		k, v := sfi2pkeys.Prop(prop)
		r[k] = v
	}
	return r
}

func (p *SAMBrowserHTTPProxy) Cleanup() {
	p.Close()
}

func (p *SAMBrowserHTTPProxy) Print() string {
	return p.proxyList["main"].Print()
}

func (p *SAMBrowserHTTPProxy) Search(search string) string {
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

func (p *SAMBrowserHTTPProxy) Target() string {
	return p.proxyHost + ":" + p.proxyPort
}

func (p *SAMBrowserHTTPProxy) Base32() string {
	return p.Keys().Addr().Base32()
}

func (p *SAMBrowserHTTPProxy) Base64() string {
	return p.Keys().Addr().Base64()
}

func (p *SAMBrowserHTTPProxy) Serve() error {
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

func (p *SAMBrowserHTTPProxy) Close() error {
	p.up = false
	for _, v := range p.proxyList {
		err := v.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *SAMBrowserHTTPProxy) find(name string) bool {
	for _, v := range p.proxyList {
		if v.ID() == name {
			return true
		}
	}
	return false
}

func (p *SAMBrowserHTTPProxy) Find(name string) (*i2phttpproxy.SAMHTTPProxy, error) {
	for _, v := range p.proxyList {
		if v.ID() == name {
			return v, nil
		}
	}
	var err error
	p.proxyList[name], err = p.New(name)
	if err != nil {
		return p.proxyList[name], nil
	}
	return nil, err
}

//return the combined host:port of the SAM bridge
func (p *SAMBrowserHTTPProxy) samaddr() string {
	return fmt.Sprintf("%s:%s", p.SamHost, p.SamPort)
}

func (p *SAMBrowserHTTPProxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	plog(req.RemoteAddr, " ", req.Method, " ", req.URL)
	var err error
	if p.proxyList[req.Host], err = p.Find(req.Host); err != nil {
		plog(err)
		return
	}
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

func (p *SAMBrowserHTTPProxy) reset(wr http.ResponseWriter, req *http.Request) {
	plog("Validating control access from", req.RemoteAddr, p.controlHost+":"+p.controlPort)
	if strings.SplitN(req.RemoteAddr, ":", 2)[0] == p.controlHost {
		for _, v := range p.proxyList {
			v.Reset(wr, req)
		}
	}
}

func (p *SAMBrowserHTTPProxy) get(wr http.ResponseWriter, req *http.Request) {
	proxy, err := p.Find(req.Host)
	if err != nil {
		proxy.Get(wr, req)
	}
}

func (p *SAMBrowserHTTPProxy) connect(wr http.ResponseWriter, req *http.Request) {
	proxy, err := p.Find(req.Host)
	if err != nil {
		proxy.Connect(wr, req)
	}
}

func (f *SAMBrowserHTTPProxy) Up() bool {
	return f.up
}

func (f *SAMBrowserHTTPProxy) New(tunName string) (*i2phttpproxy.SAMHTTPProxy, error) {
	handler, err := i2phttpproxy.NewHttpProxy(
		i2phttpproxy.SetHost(f.SamHost),
		i2phttpproxy.SetPort(f.SamPort),
		i2phttpproxy.SetProxyAddr(f.Target()),
		i2phttpproxy.SetControlAddr(f.ControlAddr()),
		i2phttpproxy.SetDebug(f.debug),
		i2phttpproxy.SetInLength(uint(f.inLength)),
		i2phttpproxy.SetOutLength(uint(f.outLength)),
		i2phttpproxy.SetInQuantity(uint(f.inQuantity)),
		i2phttpproxy.SetOutQuantity(uint(f.outQuantity)),
		i2phttpproxy.SetInBackups(uint(f.inBackups)),
		i2phttpproxy.SetOutBackups(uint(f.outBackups)),
		i2phttpproxy.SetInVariance(f.inVariance),
		i2phttpproxy.SetOutVariance(f.outVariance),
		i2phttpproxy.SetUnpublished(f.dontPublishLease),
		i2phttpproxy.SetReduceIdle(f.reduceIdle),
		i2phttpproxy.SetCompression(f.compression),
		i2phttpproxy.SetReduceIdleTime(uint(f.reduceIdleTime)),
		i2phttpproxy.SetReduceIdleQuantity(uint(f.reduceIdleQuantity)),
		i2phttpproxy.SetCloseIdle(f.closeIdle),
		i2phttpproxy.SetCloseIdleTime(uint(f.closeIdleTime)),
		i2phttpproxy.SetKeysPath(f.keyspath),
		i2phttpproxy.SetName(tunName),
	)
	if err != nil {
		return nil, err
	}
	return handler, nil
}

func (handler *SAMBrowserHTTPProxy) Load() (samtunnel.SAMTunnel, error) {
	var err error
	handler.proxyList[handler.tunName], err = i2phttpproxy.NewHttpProxy(
		i2phttpproxy.SetHost(handler.SamHost),
		i2phttpproxy.SetPort(handler.SamPort),
		i2phttpproxy.SetProxyAddr(handler.Target()),
		i2phttpproxy.SetControlAddr(handler.ControlAddr()),
		i2phttpproxy.SetDebug(handler.debug),
		i2phttpproxy.SetInLength(uint(handler.inLength)),
		i2phttpproxy.SetOutLength(uint(handler.outLength)),
		i2phttpproxy.SetInQuantity(uint(handler.inQuantity)),
		i2phttpproxy.SetOutQuantity(uint(handler.outQuantity)),
		i2phttpproxy.SetInBackups(uint(handler.inBackups)),
		i2phttpproxy.SetOutBackups(uint(handler.outBackups)),
		i2phttpproxy.SetInVariance(handler.inVariance),
		i2phttpproxy.SetOutVariance(handler.outVariance),
		i2phttpproxy.SetUnpublished(handler.dontPublishLease),
		i2phttpproxy.SetReduceIdle(handler.reduceIdle),
		i2phttpproxy.SetCompression(handler.compression),
		i2phttpproxy.SetReduceIdleTime(uint(handler.reduceIdleTime)),
		i2phttpproxy.SetReduceIdleQuantity(uint(handler.reduceIdleQuantity)),
		i2phttpproxy.SetCloseIdle(handler.closeIdle),
		i2phttpproxy.SetCloseIdleTime(uint(handler.closeIdleTime)),
		i2phttpproxy.SetKeysPath(handler.keyspath),
		i2phttpproxy.SetName(handler.tunName),
	)
	if err != nil {
		return nil, err
	}
	return handler, nil
}

func NewBrowserHttpProxy(opts ...func(*SAMBrowserHTTPProxy) error) (*SAMBrowserHTTPProxy, error) {
	var handler SAMBrowserHTTPProxy
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
	return l.(*SAMBrowserHTTPProxy), nil
}
