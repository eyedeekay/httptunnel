package i2phttpproxy

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

type SAMHTTPController struct {
	ProxyServer   *http.Server
	Profiles      []string
	savedProfiles []string
}

func NewSAMHTTPController(profiles []string, proxy *http.Server) (*SAMHTTPController, error) {
	var c SAMHTTPController
	if proxy != nil {
		c.ProxyServer = proxy
	}
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
	var err error
	wr.Header().Set("Content-Type", "text/html; charset=utf-8")
	wr.Header().Set("Access-Control-Allow-Origin", "*")
	wr.WriteHeader(http.StatusOK)
	wr.Write([]byte("200 - Restart from " + req.Header.Get("Origin") + "OK!"))
	log.Println("attempting restart")
	if runtime.GOOS == "windows" {
		err = p.windowsRestart()
	} else {
		err = unixRestart()
	}
	if err != nil {
		log.Fatal(err)
	}
	err = p.restoreProfiles()
	if err != nil {
		log.Fatal(err)
	}
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
		SetHost(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.SamHost),
		SetPort(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.SamPort),
		SetDebug(s.ProxyServer.Handler.(*SAMHTTPProxy).debug),
		SetInLength(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.InLength)),
		SetOutLength(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.OutLength)),
		SetInQuantity(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.InQuantity)),
		SetOutQuantity(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.OutQuantity)),
		SetInBackups(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.InBackupQuantity)),
		SetOutBackups(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.OutBackupQuantity)),
		SetInVariance(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.InVariance),
		SetOutVariance(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.OutVariance),
		SetUnpublished(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.Client),
		SetReduceIdle(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.ReduceIdle),
		SetReduceIdleTime(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.ReduceIdleTime)),
		SetReduceIdleQuantity(uint(s.ProxyServer.Handler.(*SAMHTTPProxy).Conf.ReduceIdleQuantity)),
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
