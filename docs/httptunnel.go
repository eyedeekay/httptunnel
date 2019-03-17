package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// tutorial line 48
import (
	"github.com/eyedeekay/gosam"
)

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// tutorial line 94
var hopHeaders = []string{
	"Accept-Language",
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Proxy-Connection",
	"Trailers",
	"Upgrade",
	"X-Forwarded-For",
	"X-Real-IP",
}

// tutorial line 107
func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
	// tutorial line 117
	if header.Get("User-Agent") != "MYOB/6.66 (AN/ON)" {
		header.Set("User-Agent", "MYOB/6.66 (AN/ON)")
	}
}

// tutorial line 55
type Proxy struct {
	Sam    *gosam.Client
	Client *http.Client
}

// NewClient is on 181, 246
func (p *Proxy) NewClient() *http.Client {
	return &http.Client{
		// tutorial line 187
		Transport: &http.Transport{
			DialContext: p.Sam.DialContext,
			//tutorial line 195
			MaxConnsPerHost:       1,
			MaxIdleConns:          0,
			MaxIdleConnsPerHost:   1,
			DisableKeepAlives:     false,
			ResponseHeaderTimeout: time.Second * 600,
			IdleConnTimeout:       time.Second * 300,
			TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: nil,
		Timeout:       time.Second * 600,
	}
}

// ServeHTTP is on line 63, 126
func (p *Proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, " ", req.Method, " ", req.URL)

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" && !strings.HasSuffix(req.URL.Host, ".i2p") {
		msg := "unsupported protocal scheme " + req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	// tutorial line 137
	delHopHeaders(req.Header)

	p.get(wr, req)
}

// tutorial lines 147, 155
func (p *Proxy) get(wr http.ResponseWriter, req *http.Request) {
	req.RequestURI = ""
	resp, err := p.Client.Do(req)
	if err != nil {
		log.Println("ServeHTTP:", err)
		return
	}
	defer resp.Body.Close()

	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

// main is on tutorial line 214
func main() {
	var addr = flag.String("addr", "127.0.0.1:7950", "The addr of the application.")
	flag.Parse()

	// tutorial line 71, 222
	sam, err := gosam.NewClientFromOptions(
		gosam.SetHost("127.0.0.1"),
		gosam.SetPort("7656"),
		gosam.SetUnpublished(true),
		gosam.SetInLength(uint(2)),
		gosam.SetOutLength(uint(2)),
		gosam.SetInQuantity(uint(1)),
		gosam.SetOutQuantity(uint(1)),
		gosam.SetInBackups(uint(1)),
		gosam.SetOutBackups(uint(1)),
		gosam.SetReduceIdle(true),
		gosam.SetReduceIdleTime(uint(2000000)),
	)
	if err != nil {
		log.Fatal(err)
	}
	handler := &Proxy{
		Sam: sam,
	}
	// tutorial line 245
	handler.Client = handler.NewClient()

	// tutorial line 252
	log.Println("Starting proxy server on", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
