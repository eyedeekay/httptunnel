package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	var addr = flag.String("addr", "127.0.0.1:7950", "The addr of the application.")
	flag.Parse()

	handler, err := NewHttpProxy()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting proxy server on", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
