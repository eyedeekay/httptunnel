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
	"crawshaw.io/littleboss"
	"github.com/danfragoso/thdwb/ketchup"
	"github.com/danfragoso/thdwb/mustard"
	"github.com/danfragoso/thdwb/sauce"
	. "github.com/eyedeekay/httptunnel"
)

var (
	tunnelName           = flag.String("service-name", "sam-http-proxy", "Name of the service(can be anything)")
	samHostString        = flag.String("bridge-host", "127.0.0.1", "host: of the SAM bridge")
	samPortString        = flag.String("bridge-port", "7656", ":port of the SAM bridge")
	watchProfiles        = flag.String("watch-profiles", "~/.mozilla/.firefox.profile.i2p.default/user.js,~/.mozilla/.firefox.profile.i2p.debug/user.js", "Monitor and control these Firefox profiles")
	destfile             = flag.String("dest-file", "invalid.tunkey", "Use a long-term destination key")
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
	runArguments         = flag.String("run-arguments", "", "Pass arguments to run-command")
	suppressLifetime     = flag.Bool("suppress-lifetime-output", false, "Suppress \"Tunnel lifetime\" output")
	runQuiet             = flag.Bool("run-quiet", false, "Suppress all non-command output")
	outproxy             = flag.String("outproxy-addr", "false.i2p", "Use this address as an outproxy, either a base32 address or a local HTTP proxy")
	socks                = flag.Bool("outproxy-socks", true, "Use a SOCKS outproxy")
)

var addr string

func main() {
	lb := littleboss.New(*tunnelName)
	proxyaddr := "127.0.0.1:7950"
	controladdr := "127.0.0.1:7951"
	for _, flag := range os.Args {
		if flag == "-run-command" {
			proxyaddr = "127.0.0.1:0"
			controladdr = "127.0.0.1:0"
		}
	}
	ln := lb.Listener("proxy-addr", "tcp", proxyaddr, "The address of the proxy")
	cln := lb.Listener("control-addr", "tcp", controladdr, "The address of the controller")
	lb.Run(func(ctx context.Context) {
		proxyMain(ctx, ln.Listener(), cln.Listener())
	})
}

func proxyMain(ctx context.Context, ln net.Listener, cln net.Listener) {
	flag.Parse()
	/*if *runCommand != "" {
		*suppressLifetime = true
		*runQuiet = true
		*debugConnection = false
	}*/
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
		SetProxyAddr(ln.Addr().String()),
		SetControlAddr(cln.Addr().String()),
		SetOutProxy(*outproxy),
		SetOutProxySocks(*socks),
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
		SetKeysPath(*destfile),
	)
	Quiet = *runQuiet
	if err != nil {
		log.Fatal(err)
	}

	ctrlsrv := &http.Server{
		ReadHeaderTimeout: 600 * time.Second,
		WriteTimeout:      600 * time.Second,
		Addr:              cln.Addr().String(),
	}
	ctrlsrv.Handler, err = NewSAMHTTPController(profiles, srv)

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

	os.Setenv("http_proxy", "http://"+ln.Addr().String())
	os.Setenv("https_proxy", "http://"+ln.Addr().String())
	os.Setenv("ftp_proxy", "http://"+ln.Addr().String())
	os.Setenv("all_proxy", "http://"+ln.Addr().String())
	os.Setenv("HTTP_PROXY", "http://"+ln.Addr().String())
	os.Setenv("HTTPS_PROXY", "http://"+ln.Addr().String())
	os.Setenv("FTP_PROXY", "http://"+ln.Addr().String())
	os.Setenv("ALL_PROXY", "http://"+ln.Addr().String())

	log.Println("Waiting a few seconds for the proxy to start")
	time.Sleep(time.Duration(10 * time.Second))

	//log.Println("Launching ", *runCommand, "with proxy http://"+ln.Addr().String())
	url := flag.Arg(0)
	resource := sauce.GetResource(url)
	htmlString := string(resource.Body)
	parsedDocument := ketchup.ParseDocument(htmlString)

	mustard.RenderDocument(parsedDocument, url)
	return

	<-ctx.Done()
}

func counter() {
	var x int
	for {
		if !*suppressLifetime {
			log.Println("Identity is", x, "minute(s) old")
			time.Sleep(1 * time.Minute)
			x++
		}
	}
}
