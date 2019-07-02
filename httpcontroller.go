package i2phttpproxy

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	//	"os"
	//	"os/exec"
)

//import (
//"github.com/eyedeekay/sam-forwarder/interface"
//)

type SAMHTTPController struct {
	ProxyServer   *SAMHTTPProxy
	Profiles      []string
	savedProfiles []string
}

var ctx context.Context

func NewSAMHTTPController(profiles []string, ProxyServer *SAMHTTPProxy) (*SAMHTTPController, error) {
	var c SAMHTTPController

	c.ProxyServer = ProxyServer
	//    c.ProxyServer.Cleanup()

	for index, profile := range profiles {
		if profile != "" {
			if bytes, err := ioutil.ReadFile(profile); err == nil {
				if string(bytes) != "" {
					log.Println("Monitoring Firefox Profile", index, "at:", profile)
					c.Profiles = append(c.Profiles, profile)
					c.savedProfiles = append(c.savedProfiles, string(bytes))
				}
			} else {
				return nil, err
			}
		}
	}
	return &c, nil
}

func (p *SAMHTTPController) restoreProfiles() error {
	for index, profile := range p.Profiles {
		if err := ioutil.WriteFile(profile, []byte(p.savedProfiles[index]), 0644); err == nil {
			log.Println("Resetting Firefox Profile", index, "at:", profile)
		} else {
			return err
		}
	}
	return nil
}

func (p *SAMHTTPController) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	wr.Header().Set("Content-Type", "text/html; charset=utf-8")
	wr.Header().Set("Access-Control-Allow-Origin", "*")
	wr.WriteHeader(http.StatusOK)
	wr.Write([]byte("200 - Restart from " + req.Header.Get("Origin") + "OK!"))
	log.Println("attempting restart")
	var err error
	p.ProxyServer, err = p.Restart(p.ProxyServer)
	if err != nil {
		log.Fatal(err)
	}
	//if err := p.ProxyServer.Server.ListenAndServe(); err != nil {
		//log.Fatal("ListenAndServe:", err)
	//}
}

func (s *SAMHTTPController) Start() (*SAMHTTPProxy, error) {
	log.Println("Starting proxy")
	return NewHttpProxy(
		SetHost(s.ProxyServer.SamHost),
		SetPort(s.ProxyServer.SamPort),
		SetDebug(s.ProxyServer.debug),
		SetProxyAddr(s.ProxyServer.Target()),
		SetControlAddr(s.ProxyServer.ControlAddr()),
		SetProfiles(s.Profiles),
		SetInLength(uint(s.ProxyServer.inLength)),
		SetOutLength(uint(s.ProxyServer.outLength)),
		SetInQuantity(uint(s.ProxyServer.inQuantity)),
		SetOutQuantity(uint(s.ProxyServer.outQuantity)),
		SetInBackups(uint(s.ProxyServer.inBackups)),
		SetOutBackups(uint(s.ProxyServer.outBackups)),
		SetInVariance(s.ProxyServer.inVariance),
		SetOutVariance(s.ProxyServer.outVariance),
		SetUnpublished(s.ProxyServer.dontPublishLease),
		SetReduceIdle(s.ProxyServer.reduceIdle),
		SetReduceIdleTime(uint(s.ProxyServer.reduceIdleTime)),
		SetReduceIdleQuantity(uint(s.ProxyServer.reduceIdleQuantity)),
		SetCloseIdle(s.ProxyServer.closeIdle),
		SetCloseIdleTime(uint(s.ProxyServer.closeIdleTime)),
		SetKeysPath(s.ProxyServer.keyspath),
	)
}

func (s *SAMHTTPController) Stop(ProxyServer *SAMHTTPProxy) error {
	log.Println("Stopping proxy")
	s.ProxyServer.Cleanup()
	err := s.restoreProfiles()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (s *SAMHTTPController) Restart(ProxyServer *SAMHTTPProxy) (*SAMHTTPProxy, error) {
	log.Println("Ordering restart")
	err := s.Stop(ProxyServer)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return s.Start()
}
