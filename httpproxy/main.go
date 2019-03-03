package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	watchProfiles        = flag.String("watch-profiles", "~/.mozilla/.firefox.profile.i2p.default/user.js,~/.mozilla/.firefox.profile.i2p.debug/user.js", "Monitor and control these Firefox profiles")
	debugConnection      = flag.Bool("conn-debug", false, "Print connection debug info")
	inboundTunnelLength  = flag.Int("in-tun-length", 2, "Tunnel Length(default 3)")
	outboundTunnelLength = flag.Int("out-tun-length", 2, "Tunnel Length(default 3)")
	inboundTunnels       = flag.Int("in-tunnels", 2, "Inbound Tunnel Count(default 2)")
	outboundTunnels      = flag.Int("out-tunnels", 2, "Outbound Tunnel Count(default 2)")
	inboundBackups       = flag.Int("in-backups", 1, "Inbound Backup Count(default 1)")
	outboundBackups      = flag.Int("out-backups", 1, "Inbound Backup Count(default 1)")
	inboundVariance      = flag.Int("in-variance", 0, "Inbound Backup Count(default 0)")
	outboundVariance     = flag.Int("out-variance", 0, "Inbound Backup Count(default 0)")
	dontPublishLease     = flag.Bool("no-publish", true, "Don't publish the leaseset(Client mode)")
	encryptLease         = flag.Bool("encrypt-lease", false, "Encrypt the leaseset(default false, inert)")
	reduceIdle           = flag.Bool("reduce-idle", false, "Reduce tunnels on extended idle time")
	closeIdle            = flag.Bool("close-idle", false, "Close tunnels on extended idle time")
	closeIdleTime        = flag.Int("close-idle-time", 3000000, "Reduce tunnels after time(Ms)")
	useCompression       = flag.Bool("use-compression", true, "Enable gzip compression")
	reduceIdleTime       = flag.Int("reduce-idle-time", 2000000, "Reduce tunnels after time(Ms)")
	reduceIdleQuantity   = flag.Int("reduce-idle-tunnels", 1, "Reduce tunnels to this level")
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

	profiles := strings.Split(*watchProfiles, ",")

	srv := &http.Server{
		ReadTimeout:  600 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         ln.Addr().String(),
	}
	var err error
	srv.Handler, err = NewHttpProxy(
		SetHost(*samHostString),
		SetPort(*samPortString),
		SetControlAddr(cln.Addr().String()),
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
		SetCompression(*useCompression),
		SetReduceIdleTime(uint(*reduceIdleTime)),
		SetReduceIdleQuantity(uint(*reduceIdleQuantity)),
		SetCloseIdle(*closeIdle),
		SetCloseIdleTime(uint(*closeIdleTime)),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctrlsrv := &http.Server{
		ReadHeaderTimeout: 600 * time.Second,
		WriteTimeout:      600 * time.Second,
		Addr:              cln.Addr().String(),
	}
	ctrlsrv.Handler, err = NewSAMHTTPController(profiles, nil)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for sig := range c {
			if sig == os.Interrupt {
				srv.Handler.(*SAMHTTPProxy).Close()
				srv.Shutdown(ctx)
				ctrlsrv.Shutdown(ctx)
			}
		}
	}()

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
