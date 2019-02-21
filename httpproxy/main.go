package main

import (
	"context"
	"log"
	"net"
	"net/http"
)

import (
	"github.com/eyedeekay/littleboss"
)

func main() {
	lb := littleboss.New("i2p-http-tunnel")
	ln := lb.Listener("http", "tcp", "127.0.0.1:7950", "The address of the proxy")
	lb.Run(func(ctx context.Context) {
		proxyMain(ctx, ln.Listener())
	})
}

func proxyMain(ctx context.Context, ln net.Listener) {
	handler, err := NewHttpProxy()
	if err != nil {
		log.Fatal(err)
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
