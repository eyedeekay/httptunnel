package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"time"
)

import (
	. "github.com/eyedeekay/httptunnel"
	"github.com/eyedeekay/littleboss"
)

var (
	tunnelName           = flag.String("service-name", "sam-http-proxy", "Name of the service(can be anything)")
	samHostString        = flag.String("bridge-host", "127.0.0.1", "host: of the SAM bridge")
	samPortString        = flag.String("bridge-port", "7656", ":port of the SAM bridge")
	debugConnection      = flag.Bool("conn-debug", false, "Print connection debug info")
	inboundTunnelLength  = flag.Int("in-tun-length", 2, "Tunnel Length(default 3)")
	outboundTunnelLength = flag.Int("out-tun-length", 2, "Tunnel Length(default 3)")
	inboundTunnels       = flag.Int("in-tunnels", 2, "Inbound Tunnel Count(default 8)")
	outboundTunnels      = flag.Int("out-tunnels", 2, "Outbound Tunnel Count(default 8)")
	inboundBackups       = flag.Int("in-backups", 1, "Inbound Backup Count(default 3)")
	outboundBackups      = flag.Int("out-backups", 1, "Inbound Backup Count(default 3)")
	inboundVariance      = flag.Int("in-variance", 0, "Inbound Backup Count(default 3)")
	outboundVariance     = flag.Int("out-variance", 0, "Inbound Backup Count(default 3)")
	dontPublishLease     = flag.Bool("no-publish", true, "Don't publish the leaseset(Client mode)")
	//encryptLease      = flag.Bool("encrypt-lease", true, "Encrypt the leaseset(")
	reduceIdle         = flag.Bool("reduce-idle", false, "Reduce tunnels on extended idle time")
	reduceIdleTime     = flag.Int("reduce-idle-time", 2000000, "Reduce tunnels after time(Ms)")
	reduceIdleQuantity = flag.Int("reduce-idle-tunnels", 1, "Reduce tunnels to this level")
)

var addr string

func main() {
	lb := littleboss.New(*tunnelName)
	ln := lb.Listener("proxy-addr", "tcp", "127.0.0.1:7950", "The address of the proxy")
	cln := lb.Listener("control-addr", "tcp", "127.0.0.1:7951", "The address of the controller")
	lb.Run(func(ctx context.Context) {
		proxyMain(ctx, ln.Listener(), cln.Listener())
	})
}

func proxyMain(ctx context.Context, ln net.Listener, cln net.Listener) {
	flag.Parse()
	srv := &http.Server{
		ReadTimeout:  600 * time.Second,
		WriteTimeout: 600 * time.Second,
		Addr:         addr,
	}
	var err error
	srv.Handler, err = NewHttpProxy(
		SetHost(*samHostString),
		SetPort(*samPortString),
		SetDebug(*debugConnection),
		SetInLength(uint(*inboundTunnelLength)),
		SetOutLength(uint(*outboundTunnelLength)),
		SetInQuantity(uint(*inboundTunnels)),
		SetOutQuantity(uint(*outboundTunnels)),
		SetInBackups(uint(*inboundBackups)),
		SetOutBackups(uint(*outboundBackups)),
		SetInVariance(*inboundVariance),
		SetOutVariance(*outboundVariance),
		SetUnpublished(*dontPublishLease),
		SetReduceIdle(*reduceIdle),
		SetReduceIdleTime(uint(*reduceIdleTime)),
		SetReduceIdleQuantity(uint(*reduceIdleQuantity)),
	)
	if err != nil {
		log.Fatal(err)
	}
	ctrlsrv := &http.Server{
		ReadTimeout:  600 * time.Second,
		WriteTimeout: 600 * time.Second,
		Addr:         addr,
		Handler:      &SAMHTTPController{},
	}

	go func() {
		log.Println("Starting control server on", cln.Addr())
		if err := ctrlsrv.Serve(cln); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			log.Fatal("Serve:", err)
		}
		log.Println("Stopping control server on", cln.Addr())
	}()

	go func() {
		log.Println("Starting proxy server on", ln.Addr())
		if err := srv.Serve(ln); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			log.Fatal("Serve:", err)
		}
		log.Println("Stopping proxy server on", ln.Addr())
	}()

	go counter()

	<-ctx.Done()
}

func counter() {
	var x int
	for {
		log.Println("Identity is", x, "minute(s) old")
		time.Sleep(1 * time.Minute)
		x++
	}
}
