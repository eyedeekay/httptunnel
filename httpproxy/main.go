package main

import (
	"context"
	"crypto/tls"
	"flag"
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
	flagHTTPS := lb.Listener("http", "tcp", "127.0.0.1:7950", "The address of the application")
	lb.Run(func(ctx context.Context) {
		proxyMain(ctx, flagHTTPS.Listener())
	})
}

func proxyMain(ctx context.Context, ln net.Listener) {
	flag.Parse()

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
			log.Fatal("ListenAndServe:", err)
		}
	}()

	<-ctx.Done()
	handler.Shutdown(ctx)
}
