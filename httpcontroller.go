package i2phttpproxy

import (
	"log"
	"net/http"
	"os"
	"os/exec"
)

type SAMHTTPController struct {
}

func (p *SAMHTTPController) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	err := unixRestart()
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
