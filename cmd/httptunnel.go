package main

import (
	"crypto/tls"
	"flag"
	"github.com/eyedeekay/goSam"
	"log"
	"net/http"
    "time"
    "github.com/eyedeekay/httptunnel"
)

func main() {
	var addr = flag.String("addr", "127.0.0.1:7844", "The addr of the application.")
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
		Client: &http.Client{
			Transport: &http.Transport{
				Dial:                  sam.Dial,
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
		},
	}

	log.Println("Starting proxy server on", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
