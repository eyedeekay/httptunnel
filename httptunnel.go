package i2phttpproxy

import (
	"io"
	"log"
	"net/http"
	"strings"
    "crypto/tls"
    "time"
    "github.com/eyedeekay/goSam"
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

type Proxy struct {
    Sam    *goSam.Client
	Client *http.Client
}

func (p *Proxy) freshClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial:                  p.Sam.Dial,
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

func (p *Proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, " ", req.Method, " ", req.URL)

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" && !strings.HasSuffix(req.URL.Host, ".i2p") {
		msg := "unsupported protocal scheme " + req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	req.RequestURI = ""

	delHopHeaders(req.Header)

	go p.get(wr, req)
}

func (p *Proxy) get(wr http.ResponseWriter, req *http.Request) {
    Client := p.freshClient()
	resp, err := Client.Do(req)
    //resp, err := p.Client.Do(req)
	if err != nil {
		//http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Println("ServeHTTP:", err)
		return
	}
	defer resp.Body.Close()

	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}
