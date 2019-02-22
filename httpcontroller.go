package i2phttpproxy

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

type SAMHTTPController struct {
	ProxyServer *http.Server
}

func (p *SAMHTTPController) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	var err error
	if runtime.GOOS == "windows" {
		err = p.windowsRestart()
	} else {
		err = unixRestart()
	}
	if err != nil {
		log.Fatal(err)
	}
	wr.WriteHeader(http.StatusOK)
	wr.Write([]byte("200 - Restart OK!"))
}

func unixRestart() error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	cmnd := exec.Command(path, "-littleboss=reload")
	err = cmnd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (s *SAMHTTPController) windowsStart() error {
	var err error
	s.ProxyServer.Handler, err = NewHttpProxy(
		SetHost(s.ProxyServer.Handler.(*SAMHTTPProxy).SamHost),
		SetPort(s.ProxyServer.Handler.(*SAMHTTPProxy).SamPort),
		SetDebug(s.ProxyServer.Handler.(*SAMHTTPProxy).debug),
		SetInLength(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).inLength)),
		SetOutLength(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).outLength)),
		SetInQuantity(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).inQuantity)),
		SetOutQuantity(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).outQuantity)),
		SetInBackups(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).inBackups)),
		SetOutBackups(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).outBackups)),
		SetInVariance(s.ProxyServer.Handler.(*SAMHTTPProxy).inVariance),
		SetOutVariance(s.ProxyServer.Handler.(*SAMHTTPProxy).outVariance),
		SetUnpublished(s.ProxyServer.Handler.(*SAMHTTPProxy).dontPublishLease),
		SetReduceIdle(s.ProxyServer.Handler.(*SAMHTTPProxy).reduceIdle),
		SetReduceIdleTime(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).reduceIdleTime)),
		SetReduceIdleQuantity(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).reduceIdleQuantity)),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := s.ProxyServer.ListenAndServe(); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
	return nil
}

func (s *SAMHTTPController) windowsStop() error {
	err := s.ProxyServer.Shutdown(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func (s *SAMHTTPController) windowsRestart() error {
	err := s.windowsStop()
	if err != nil {
		return err
	}
	err = s.windowsStart()
	if err != nil {
		return err
	}
	return nil
}
