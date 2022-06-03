package i2pbrowserproxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eyedeekay/goSam"
	"golang.org/x/time/rate"

	//"github.com/eyedeekay/goSam/compat"
	proxycommon "github.com/eyedeekay/httptunnel/common"

	i2ptunconf "github.com/eyedeekay/sam-forwarder/config"
	"github.com/eyedeekay/sam-forwarder/hashhash"

	sfi2pkeys "github.com/eyedeekay/sam-forwarder/i2pkeys"

	"github.com/eyedeekay/i2pkeys"
	samtunnel "github.com/eyedeekay/sam-forwarder/interface"
)

type samClient struct {
	goSam       *goSam.Client
	client      *http.Client
	transport   *http.Transport
	rateLimiter *rate.Limiter
}

type SAMMultiProxy struct {
	clients map[string]*samClient
	Hasher  *hashhash.Hasher

	Conf *i2ptunconf.Conf

	useOutProxy bool

	dialed     bool
	debug      bool
	up         bool
	aggressive bool
	recent     string
}

var Quiet bool

func (f *SAMMultiProxy) Config() *i2ptunconf.Conf {
	return f.Conf
}

func (f *SAMMultiProxy) findClient(key string) *samClient {
	var err error
	log.Println("finding client", key)
	for site, proxy := range f.clients {
		if site == key {
			log.Println("found client", site)
			f.recent = key
			return proxy
		}
	}
	f.clients[key], err = f.freshSAMClient(key)
	if err != nil {
		log.Fatal(err)
	}
	return f.clients[key]
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
	return f.Conf.TunName
}

func (p *SAMMultiProxy) Keys() i2pkeys.I2PKeys {
	//k, _ := samkeys.DestToKeys(
	k := p.findClient(p.recent).goSam.PrivateAddr()
	//)
	return k
}

func (f *SAMMultiProxy) Props() map[string]string {
	r := make(map[string]string)
	print := f.print()
	print = append(print, "base32="+f.Base32())
	print = append(print, "base64="+f.Base64())
	print = append(print, "base32words="+f.Base32Readable())
	for _, prop := range print {
		k, v := sfi2pkeys.Prop(prop)
		r[k] = v
	}
	return r
}

func (p *SAMMultiProxy) Cleanup() {
	p.Close()
}

func (p *SAMMultiProxy) Print() string {
	return p.findClient(p.recent).goSam.Print()
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
	return p.Conf.TargetHost + ":" + p.Conf.TargetPort
}

func (p *SAMMultiProxy) Base32() string {
	return p.findClient(p.recent).goSam.Base32()
}

// Base32Readable returns the base32 address where the local service is being
// forwarded, but as a list of English words(More languages later if it works)
// instead of as a hash for person-to-person transmission.
func (f *SAMMultiProxy) Base32Readable() string {
	b32 := strings.Replace(f.Base32(), ".b32.i2p", "", 1)
	rhash, _ := f.Hasher.Friendly(b32)
	return rhash + " " + strconv.Itoa(len(b32))
}

func (p *SAMMultiProxy) Base64() string {
	return p.findClient(p.recent).goSam.Base64()
}

func (p *SAMMultiProxy) Serve() error {
	ln, err := net.Listen("tcp", p.Conf.TargetHost+":"+p.Conf.TargetPort)
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
	for _, v := range p.clients {
		if err := v.goSam.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (p *SAMMultiProxy) freshTransport(key string) *http.Transport {
	//var err error
	if p.clients[key] == nil {
		p.clients[key] = &samClient{}
		p.clients[key].goSam, _ = p.freshGoSAMClient()
	}

	t := http.Transport{
		DialContext:           p.clients[key].goSam.DialContext,
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

func (p *SAMMultiProxy) freshClient(key string) *http.Client {
	return &http.Client{
		Transport:     p.freshTransport(key),
		Timeout:       time.Second * 300,
		CheckRedirect: nil,
	}
}

func (p *SAMMultiProxy) freshSAMClient(key string) (*samClient, error) {
	var s samClient
	var err error
	s.goSam, err = p.freshGoSAMClient()
	if err != nil {
		return nil, err
	}
	s.transport = p.freshTransport(key)
	s.client = p.freshClient(key)
	return &s, nil
}

func (p *SAMMultiProxy) freshGoSAMClient() (*goSam.Client, error) {
	return goSam.NewClientFromOptions(
		goSam.SetHost(p.Conf.SamHost),
		goSam.SetPort(p.Conf.SamPort),
		goSam.SetUnpublished(p.Conf.Client),
		goSam.SetInLength(uint(p.Conf.InLength)),
		goSam.SetOutLength(uint(p.Conf.OutLength)),
		goSam.SetInQuantity(uint(p.Conf.InQuantity)),
		goSam.SetOutQuantity(uint(p.Conf.OutQuantity)),
		goSam.SetInBackups(uint(p.Conf.InBackupQuantity)),
		goSam.SetOutBackups(uint(p.Conf.OutBackupQuantity)),
		goSam.SetReduceIdle(p.Conf.ReduceIdle),
		goSam.SetReduceIdleTime(uint(p.Conf.ReduceIdleTime)),
		goSam.SetReduceIdleQuantity(uint(p.Conf.ReduceIdleQuantity)),
		goSam.SetCompression(p.Conf.UseCompression),
		goSam.SetDebug(p.debug),
		goSam.SetLocalDestination(p.Conf.ClientDest),
	)
}

//return the combined host:port of the SAM bridge
func (p *SAMMultiProxy) samaddr() string {
	return fmt.Sprintf("%s:%s", p.Conf.SamHost, p.Conf.SamPort)
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
		if req.URL.Host == p.Conf.ControlHost+":"+p.Conf.ControlPort {
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
	plog("Validating control access from", req.RemoteAddr, p.Conf.ControlHost+":"+p.Conf.ControlPort)
	if strings.SplitN(req.RemoteAddr, ":", 2)[0] == p.Conf.ControlHost {
		plog("Validated control access from", req.RemoteAddr, p.Conf.ControlHost+":"+p.Conf.ControlPort)
		resp, err := http.Get("http://" + p.Conf.ControlHost + ":" + p.Conf.ControlPort)
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
	client, req := p.Signin(wr, req)
	proxycommon.DelHopHeaders(req.Header)
	resp, err := client.client.Do(req)
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
	client, req := p.Signin(wr, req)
	dest_conn, err := client.goSam.Dial("tcp", req.URL.Host)
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
	if p.Conf.KeyFilePath != "invalid.tunkey" {
		if _, err := os.Stat(p.Conf.KeyFilePath); os.IsNotExist(err) {
			if p.findClient(p.recent).goSam != nil {
				if p.findClient(p.recent).goSam.Destination() != "" {
					ioutil.WriteFile(p.Conf.KeyFilePath, []byte(p.findClient(p.recent).goSam.Destination()), 0644)
					p.Conf.ClientDest = p.findClient(p.recent).goSam.Destination()
					return p.findClient(p.recent).goSam.Destination()
				}
			}
		} else {
			if keys, err := ioutil.ReadFile(p.Conf.KeyFilePath); err == nil {
				p.Conf.ClientDest = string(keys)
				return string(keys)
			}
		}
	}
	return ""
}

func (handler *SAMMultiProxy) Load() (samtunnel.SAMTunnel, error) {
	var err error
	handler.clients = make(map[string]*samClient)
	handler.Conf.ClientDest = handler.Save()
	handler.clients["general"] = &samClient{}
	handler.clients["general"].goSam, err = goSam.NewClientFromOptions(
		goSam.SetHost(handler.Conf.SamHost),
		goSam.SetPort(handler.Conf.SamPort),
		goSam.SetUnpublished(handler.Conf.Client),
		goSam.SetInLength(uint(handler.Conf.InLength)),
		goSam.SetOutLength(uint(handler.Conf.OutLength)),
		goSam.SetInQuantity(uint(handler.Conf.InQuantity)),
		goSam.SetOutQuantity(uint(handler.Conf.OutQuantity)),
		goSam.SetInBackups(uint(handler.Conf.InBackupQuantity)),
		goSam.SetOutBackups(uint(handler.Conf.OutBackupQuantity)),
		goSam.SetReduceIdle(handler.Conf.ReduceIdle),
		goSam.SetReduceIdleTime(uint(handler.Conf.ReduceIdleTime)),
		goSam.SetReduceIdleQuantity(uint(handler.Conf.ReduceIdleQuantity)),
		goSam.SetCompression(handler.Conf.UseCompression),
		goSam.SetDebug(handler.debug),
		goSam.SetLocalDestination(handler.Conf.ClientDest),
	)
	if err != nil {
		return nil, err
	}
	handler.clients["general"].transport = handler.freshTransport("general")
	handler.clients["general"].client = handler.freshClient("general")
	handler.Hasher, err = hashhash.NewHasher(len(strings.Replace(handler.Base32(), ".b32.i2p", "", 1)))
	if err != nil {
		return nil, err
	}
	handler.up = true
	return handler, nil
}

func NewHttpProxy(opts ...func(*SAMMultiProxy) error) (*SAMMultiProxy, error) {
	var handler SAMMultiProxy
	handler.Conf = &i2ptunconf.Conf{}
	handler.Conf.SamHost = "127.0.0.1"
	handler.Conf.SamPort = "7656"
	handler.recent = "general"
	handler.aggressive = false
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
