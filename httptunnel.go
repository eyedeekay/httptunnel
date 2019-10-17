package i2phttpproxy

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/eyedeekay/goSam"
	"github.com/eyedeekay/goSam/compat"
	"github.com/eyedeekay/httptunnel/common"
	"github.com/eyedeekay/sam-forwarder/config"
	"github.com/eyedeekay/sam-forwarder/hashhash"
	"github.com/eyedeekay/sam-forwarder/i2pkeys"
	"github.com/eyedeekay/sam-forwarder/interface"
	"github.com/eyedeekay/sam-forwarder/tcp"
	"github.com/eyedeekay/sam3/i2pkeys"
	"github.com/mwitkow/go-http-dialer"
	"github.com/phayes/freeport"
)

type SAMHTTPProxy struct {
	goSam          *goSam.Client
	Hasher         *hashhash.Hasher
	client         *http.Client
	outproxyclient *http.Client
	transport      *http.Transport
	rateLimiter    *rate.Limiter
	outproxy       *samforwarder.SAMClientForwarder
	outproxydialer proxy.Dialer
	outproxytype   string

	UseOutProxy string

	dialed bool
	debug  bool
	up     bool

	Conf *i2ptunconf.Conf
}

var Quiet bool

func plog(in ...interface{}) {
	if !Quiet {
		log.Println(in...)
	}
}

func (f *SAMHTTPProxy) Config() *i2ptunconf.Conf {
	return f.Conf
}

func (f *SAMHTTPProxy) print() []string {
	return strings.Split(f.Print(), " ")
}

func (f *SAMHTTPProxy) GetType() string {
	return "httpclient"
}

func (f *SAMHTTPProxy) ID() string {
	return f.Conf.TunName
}

func (f *SAMHTTPProxy) Keys() i2pkeys.I2PKeys {
	k, _ := samkeys.DestToKeys(f.goSam.Destination())
	return k
}

func (f *SAMHTTPProxy) Props() map[string]string {
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

func (p *SAMHTTPProxy) Cleanup() {
	p.Close()
}

func (p *SAMHTTPProxy) Print() string {
	return p.goSam.Print()
}

func (p *SAMHTTPProxy) Search(search string) string {
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

func (p *SAMHTTPProxy) Target() string {
	return p.Conf.TargetHost + ":" + p.Conf.TargetPort
}

func (p *SAMHTTPProxy) Base32() string {
	return p.goSam.Base32()
}

// Base32Readable returns the base32 address where the local service is being
// forwarded, but as a list of English words(More languages later if it works)
// instead of as a hash for person-to-person transmission.
func (f *SAMHTTPProxy) Base32Readable() string {
	b32 := strings.Replace(f.Base32(), ".b32.i2p", "", 1)
	rhash, _ := f.Hasher.Friendly(b32)
	return rhash + " " + strconv.Itoa(len(b32))
}

func (p *SAMHTTPProxy) Base64() string {
	return p.goSam.Base64()
}

func (p *SAMHTTPProxy) Serve() error {
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

func (p *SAMHTTPProxy) Close() error {
	p.up = false
	return p.goSam.Close()
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
	return fmt.Sprintf("%s:%s", p.Conf.SamHost, p.Conf.SamPort)
}

func (p *SAMHTTPProxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	plog(req.RemoteAddr, " ", req.Method, " ", req.URL)
	p.Save()
	/*if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		if !(req.Method == http.MethodConnect) {
			msg := "Unsupported protocol scheme " + req.URL.Scheme
			http.Error(wr, msg, http.StatusBadRequest)
			plog(msg)
			return
		}
	}*/

	if req.URL.Host == p.Conf.ControlHost+":"+p.Conf.ControlPort {
		p.reset(wr, req)
		return
	}

	if !strings.HasSuffix(req.URL.Host, ".i2p") && p.UseOutProxy == "" {
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
		if !strings.HasSuffix(req.URL.Host, ".i2p") && p.UseOutProxy != "" {
			p.outproxyget(wr, req)
			return
		} else {
			plog("No outproxy configured ", p.UseOutProxy, p.outproxy.Target())
			return
		}
		p.connect(wr, req)
		return
	}

}

// SetupProxy sets proxy environment variables based on the settings in use by the proxy
func (p *SAMHTTPProxy) SetupProxy() {
	SetupProxy(p.Target())
}

// SetupProxy sets proxy environment variables based on a string address
func SetupProxy(addr string) {
	os.Setenv("http_proxy", "http://"+addr)
	os.Setenv("https_proxy", "http://"+addr)
	os.Setenv("ftp_proxy", "http://"+addr)
	os.Setenv("all_proxy", "http://"+addr)
	os.Setenv("HTTP_PROXY", "http://"+addr)
	os.Setenv("HTTPS_PROXY", "http://"+addr)
	os.Setenv("FTP_PROXY", "http://"+addr)
	os.Setenv("ALL_PROXY", "http://"+addr)
}

// UnProxyLocal adds local IP addresses to an un-proxied range. It is not used
// here but is useful for some servers.
func UnProxyLocal(additionalAddresses []string) {
	add := ""
	for _, v := range additionalAddresses {
		add += "," + v
	}
	os.Setenv("no_proxy", "127.0.0.1,localhost"+add)
	os.Setenv("NO_PROXY", "127.0.0.1,localhost"+add)
}

func (p *SAMHTTPProxy) reset(wr http.ResponseWriter, req *http.Request) {
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

func (p *SAMHTTPProxy) outproxyget(wr http.ResponseWriter, req *http.Request) {
	plog("CONNECT via outproxy to", req.URL.Host)
	dest_conn, err := p.outproxydialer.Dial("tcp", req.URL.String())
	if err != nil {
		if !Quiet {
			plog(err.Error())
		}
		return
	}
	hijacker, ok := wr.(http.Hijacker)
	if !ok {
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		if !Quiet {
			plog(err.Error())
		}
		return
	}
	go proxycommon.Transfer(dest_conn, client_conn)
	go proxycommon.Transfer(client_conn, dest_conn)
}

func (p *SAMHTTPProxy) get(wr http.ResponseWriter, req *http.Request) {
	req.RequestURI = ""
	proxycommon.DelHopHeaders(req.Header)
	plog("Getting i2p page")
	//p.client = p.freshClient()
	resp, err := p.client.Do(req)
	if err != nil {
		msg := "Proxy Error " + err.Error()
		if !Quiet {
			plog(msg)
		}
		return
	}
	defer resp.Body.Close()

	proxycommon.CopyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func (p *SAMHTTPProxy) connect(wr http.ResponseWriter, req *http.Request) {
	plog("CONNECT via i2p to", req.URL.Host)
	dest_conn, err := p.goSam.Dial("tcp", req.URL.Host)
	if err != nil {
		if !Quiet {
			plog(err.Error())
		}
		return
	}
	wr.WriteHeader(http.StatusOK)
	hijacker, ok := wr.(http.Hijacker)
	if !ok {
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		if !Quiet {
			plog(err.Error())
		}
		return
	}
	go proxycommon.Transfer(dest_conn, client_conn)
	go proxycommon.Transfer(client_conn, dest_conn)
}

func (f *SAMHTTPProxy) Up() bool {
	return f.up
}

func (p *SAMHTTPProxy) Save() string {
	if p.Conf.KeyFilePath != "invalid.tunkey" {
		if _, err := os.Stat(p.Conf.KeyFilePath); os.IsNotExist(err) {
			if p.goSam != nil {
				if p.goSam.Destination() != "" {
					ioutil.WriteFile(p.Conf.KeyFilePath, []byte(p.goSam.Destination()), 0644)
					p.Conf.ClientDest = p.goSam.Destination()
					return p.goSam.Destination()
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

func (p *SAMHTTPProxy) GuaranteePrefix(str string) string {
	if strings.HasPrefix(p.outproxytype, str) {
		return str
	}
	return p.outproxytype + str
}

func (handler *SAMHTTPProxy) Load() (samtunnel.SAMTunnel, error) {
	var err error
	handler.Conf.ClientDest = handler.Save()
	handler.goSam, err = goSam.NewClientFromOptions(
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
		goSam.SetCloseIdle(handler.Conf.CloseIdle),
		goSam.SetCloseIdleTime(uint(handler.Conf.CloseIdleTime)),
		goSam.SetCompression(handler.Conf.UseCompression),
		goSam.SetDebug(handler.debug),
		goSam.SetLocalDestination(handler.Conf.ClientDest),
	)
	if err != nil {
		return nil, err
	}
	handler.transport = handler.freshTransport()
	handler.client = handler.freshClient()
	if handler.UseOutProxy != "" {
		if strings.HasSuffix(handler.UseOutProxy, ".i2p") {
			plog("Configuring an outproxy,", handler.UseOutProxy)
			config := handler.Conf
			port, err := freeport.GetFreePort()
			if err != nil {
				return nil, err
			}
			config.TargetPort = strconv.Itoa(port)
			config.TunName = handler.Conf.TunName + "-outproxy"
			config.ClientDest = handler.UseOutProxy
			handler.outproxy, err = samforwarder.NewSAMClientForwarderFromOptions(
				samforwarder.SetClientSaveFile(config.SaveFile),
				samforwarder.SetClientFilePath(config.SaveDirectory),
				samforwarder.SetClientHost(config.TargetHost),
				samforwarder.SetClientPort(config.TargetPort),
				samforwarder.SetClientSAMHost(config.SamHost),
				samforwarder.SetClientSAMPort(config.SamPort),
				samforwarder.SetClientSigType(config.SigType),
				samforwarder.SetClientName(config.TunName),
				samforwarder.SetClientInLength(config.InLength),
				samforwarder.SetClientOutLength(config.OutLength),
				samforwarder.SetClientInVariance(config.InVariance),
				samforwarder.SetClientOutVariance(config.OutVariance),
				samforwarder.SetClientInQuantity(config.InQuantity),
				samforwarder.SetClientOutQuantity(config.OutQuantity),
				samforwarder.SetClientInBackups(config.InBackupQuantity),
				samforwarder.SetClientOutBackups(config.OutBackupQuantity),
				samforwarder.SetClientEncrypt(config.EncryptLeaseSet),
				samforwarder.SetClientLeaseSetKey(config.LeaseSetKey),
				samforwarder.SetClientLeaseSetPrivateKey(config.LeaseSetPrivateKey),
				samforwarder.SetClientLeaseSetPrivateSigningKey(config.LeaseSetPrivateSigningKey),
				samforwarder.SetClientAllowZeroIn(config.InAllowZeroHop),
				samforwarder.SetClientAllowZeroOut(config.OutAllowZeroHop),
				samforwarder.SetClientFastRecieve(config.FastRecieve),
				samforwarder.SetClientCompress(config.UseCompression),
				samforwarder.SetClientReduceIdle(config.ReduceIdle),
				samforwarder.SetClientReduceIdleTimeMs(config.ReduceIdleTime),
				samforwarder.SetClientReduceIdleQuantity(config.ReduceIdleQuantity),
				samforwarder.SetClientCloseIdle(config.CloseIdle),
				samforwarder.SetClientCloseIdleTimeMs(config.CloseIdleTime),
				samforwarder.SetClientAccessListType(config.AccessListType),
				samforwarder.SetClientAccessList(config.AccessList),
				samforwarder.SetClientMessageReliability(config.MessageReliability),
				samforwarder.SetClientPassword(config.KeyFilePath),
				samforwarder.SetClientDestination(config.ClientDest),
			)
			if err != nil {
				return nil, err
			}
			go handler.outproxy.Serve()
			if handler.outproxytype == "http://" {
				proxyURL, err := url.Parse(handler.GuaranteePrefix(handler.outproxy.Target()))
				if err != nil {
					return nil, err
				}
				handler.outproxydialer = http_dialer.New(proxyURL, http_dialer.WithTls(&tls.Config{}))
				if err != nil {
					return nil, err
				}
				handler.outproxyclient = &http.Client{
					Transport: &http.Transport{
						Dial:            handler.outproxydialer.Dial,
						TLSNextProto:    make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
						TLSClientConfig: &tls.Config{},
					},
					Timeout:       time.Second * 300,
					CheckRedirect: nil,
				}
			} else {
				handler.outproxydialer, err = proxy.SOCKS5("tcp", handler.outproxy.Target(), nil, nil)
				if err != nil {
					return nil, err
				}
				handler.outproxyclient = &http.Client{
					Transport: &http.Transport{
						Dial:            handler.outproxydialer.Dial,
						TLSNextProto:    make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
						TLSClientConfig: &tls.Config{},
					},
					Timeout:       time.Second * 300,
					CheckRedirect: nil,
				}
			}
			plog("setup outproxy on", handler.outproxy.Target())
		} else {
			if handler.outproxytype == "http://" {
				proxyURL, err := url.Parse(handler.GuaranteePrefix(handler.UseOutProxy))
				if err != nil {
					return nil, err
				}
				handler.outproxydialer = http_dialer.New(proxyURL, http_dialer.WithTls(&tls.Config{}))
				if err != nil {
					return nil, err
				}
				handler.outproxyclient = &http.Client{
					Transport: &http.Transport{
						Dial:            handler.outproxydialer.Dial,
						TLSNextProto:    make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
						TLSClientConfig: &tls.Config{},
					},
					Timeout:       time.Second * 300,
					CheckRedirect: nil,
				}
			} else {
				handler.outproxydialer, err = proxy.SOCKS5("tcp", handler.UseOutProxy, nil, proxy.Direct)
				if err != nil {
					return nil, err
				}
				handler.outproxyclient = &http.Client{
					Transport: &http.Transport{
						Dial:            handler.outproxydialer.Dial,
						TLSNextProto:    make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
						TLSClientConfig: &tls.Config{},
					},
					Timeout:       time.Second * 300,
					CheckRedirect: nil,
				}
			}
			plog("setup outproxy on", handler.GuaranteePrefix(handler.UseOutProxy))
		}
	}
	handler.Hasher, err = hashhash.NewHasher(len(strings.Replace(handler.Base32(), ".b32.i2p", "", 1)))
	if err != nil {
		return nil, err
	}
	handler.up = true
	return handler, nil
}

func NewHttpProxy(opts ...func(*SAMHTTPProxy) error) (*SAMHTTPProxy, error) {
	var handler SAMHTTPProxy
	handler.Conf = &i2ptunconf.Conf{}
	handler.Conf.SamHost = "127.0.0.1"
	handler.Conf.SamPort = "7656"
	handler.Conf.ControlHost = "127.0.0.1"
	handler.Conf.ControlPort = "7951"
	handler.UseOutProxy = ""
	handler.outproxytype = "http://"
	for _, o := range opts {
		if err := o(&handler); err != nil {
			return nil, err
		}
	}
	l, e := handler.Load()
	if e != nil {
		return nil, e
	}
	return l.(*SAMHTTPProxy), nil
}
