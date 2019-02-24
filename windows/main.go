package main

import (
	"flag"
	"log"
	"net/http"
	"strings"
	"time"
)

import (
	. "github.com/eyedeekay/httptunnel"
)

var (
	//tunnelName           = flag.String("service-name", "sam-http-proxy", "Name of the service(can be anything)")
	samHostString        = flag.String("bridge-host", "127.0.0.1", "host: of the SAM bridge")
	samPortString        = flag.String("bridge-port", "7656", ":port of the SAM bridge")
	proxHostString       = flag.String("proxy-host", "127.0.0.1", "host: of the HTTP proxy")
	proxPortString       = flag.String("proxy-port", "7950", ":port of the HTTP proxy")
	controlHostString    = flag.String("control-host", "127.0.0.1", "The host of the controller")
	controlPortString    = flag.String("control-host", "7951", "The port of the controller")
	watchProfiles        = flag.String("watch-profiles", "~/.mozilla/.firefox.profile.i2p.default/user.js,~/.mozilla/.firefox.profile.i2p.debug/user.js", "Monitor and control these Firefox profiles")
	debugConnection      = flag.Bool("conn-debug", false, "Print connection debug info")
	inboundTunnelLength  = flag.Int("in-tun-length", 3, "Tunnel Length(default 3)")
	outboundTunnelLength = flag.Int("out-tun-length", 3, "Tunnel Length(default 3)")
	inboundTunnels       = flag.Int("in-tunnels", 4, "Inbound Tunnel Count(default 8)")
	outboundTunnels      = flag.Int("out-tunnels", 2, "Outbound Tunnel Count(default 8)")
	inboundBackups       = flag.Int("in-backups", 3, "Inbound Backup Count(default 3)")
	outboundBackups      = flag.Int("out-backups", 3, "Inbound Backup Count(default 3)")
	inboundVariance      = flag.Int("in-variance", 3, "Inbound Backup Count(default 3)")
	outboundVariance     = flag.Int("out-variance", 3, "Inbound Backup Count(default 3)")
	dontPublishLease     = flag.Bool("no-publish", true, "Don't publish the leaseset(Client mode)")
	//encryptLease      = flag.Bool("encrypt-lease", true, "Encrypt the leaseset(")
	reduceIdle         = flag.Bool("reduce-idle", false, "Reduce tunnels on extended idle time")
	reduceIdleTime     = flag.Int("reduce-idle-time", 2000000, "Reduce tunnels after time(Ms)")
	reduceIdleQuantity = flag.Int("reduce-idle-tunnels", 1, "Reduce tunnels to this level")
)

func main() {
	flag.Parse()
	addr := *proxHostString + ":" + *proxPortString

	srv := &http.Server{
		ReadTimeout:  600 * time.Second,
		WriteTimeout: 600 * time.Second,
		Addr:         addr,
	}
	profiles := strings.Split(*watchProfiles, ",")
	go SetupController(srv, *controlHostString+":"+*controlPortString, profiles)
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

	go counter()

	log.Println("Starting proxy server on", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func counter() {
	var x int
	for {
		log.Println("Identity is", x, "minute(s) old")
		time.Sleep(1 * time.Minute)
		x++
	}
}

func SetupController(srv *http.Server, addr string, profiles []string) {
	ctrlsrv := &http.Server{
		ReadTimeout:  600 * time.Second,
		WriteTimeout: 600 * time.Second,
		Addr:         addr,
	}
	var err error
	ctrlsrv.Handler, err = NewSAMHTTPController(profiles, srv)

	log.Println("Starting control server on", addr)
	if err := ctrlsrv.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			return
		}
		log.Fatal("Serve:", err)
	}
	log.Println("Stopping control server on", addr)
}
