package i2pbrowserproxy

import (
	"bytes"
    "encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
    "strings"
)

// Create a struct that models the structure of a user, both in the request body, and in the DB
type Credentials struct {
	User string
	Site string
}

func ProxyBasicAuth(r *http.Request) (username, password string, ok bool) {
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" {
		return
	}
    return parseBasicAuth(auth)
}

func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true

}
func DecodeIdentity(body *http.Request) (*http.Request, *Credentials, error) {
	var creds Credentials
	bb, err := ioutil.ReadAll(body.Body)
	if err != nil {
		return body, &creds, err
	}

	req, err := http.NewRequest(body.Method, body.URL.String(), bytes.NewReader(bb))
	if err != nil {
		return req, &creds, err
	}
	var ok bool
	creds.User, creds.Site, ok = ProxyBasicAuth(body)
	if ok {
		log.Println("OK", creds.User, creds.Site)
	} else {
		log.Println("NOT OK", creds.User, creds.Site)
	}
	return req, &creds, nil
}

func (m *SAMMultiProxy) Signin(w http.ResponseWriter, r *http.Request) (*samClient, *http.Request) {
	if m.aggressive {
		return m.findClient(r.Host), r
	}
	r, creds, err := DecodeIdentity(r)
	if err != nil {
		if err.Error() == "EOF" {
			log.Println("No auth parameters passed, falling back to general")
			return m.clients["general"], r
		}
		w.WriteHeader(http.StatusBadRequest)
		return nil, nil
	}
	return m.findClient(creds.Site), r
}
