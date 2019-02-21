package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"
)

import (
	"github.com/eyedeekay/goSam"
	"github.com/eyedeekay/httptunnel"
	"github.com/eyedeekay/littleboss"
)

func main() {
	lb := littleboss.New("i2p-http-tunnel")
	ln := lb.Listener("http", "tcp", "127.0.0.1:7950", "The address of the proxy")
	//cln := lb.Listener("http", "tcp", "127.0.0.1:7951", "The address of the proxy controller")
	lb.Run(func(ctx context.Context) {
		proxyMain(ctx, ln.Listener())
	})
}

func proxyMain(ctx context.Context, ln net.Listener) {
	sam, err := goSam.NewClientFromOptions(
		goSam.SetHost("127.0.0.1"),
		goSam.SetPort("7656"),
		goSam.SetUnpublished(true),
		goSam.SetInLength(uint(2)),
		goSam.SetOutLength(uint(2)),
		goSam.SetInQuantity(uint(4)),
		goSam.SetOutQuantity(uint(4)),
		goSam.SetInBackups(uint(2)),
		goSam.SetOutBackups(uint(2)),
		goSam.SetReduceIdle(true),
		goSam.SetReduceIdleTime(uint(2000000)),
	)
	if err != nil {
		log.Fatal(err)
	}

	handler := &i2phttpproxy.Proxy{
		Sam: sam,
		Client: &http.Client{
			Transport: &http.Transport{
				Dial:                  sam.Dial,
				MaxIdleConns:          0,
				MaxIdleConnsPerHost:   2,
				DisableKeepAlives:     false,
				ResponseHeaderTimeout: time.Second * 600,
				ExpectContinueTimeout: time.Second * 600,
				IdleConnTimeout:       time.Second * 600,
				TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: nil,
			Timeout:       time.Second * 600,
		},
	}

	go func() {
		log.Println("Starting proxy server on", ln.Addr())
		if err := http.Serve(ln, handler); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			log.Fatal("Serve:", err)
		}
	}()

	log.Println("Stopping proxy server on", ln.Addr())
	<-ctx.Done()
}
