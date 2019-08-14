package i2pbrowserproxy

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Credentials struct {
	User string
	Site string
}

// This part is copied directly from the Go source code https://golang.org/src/net/http/request.go?s=29249:29315#L872
// https://golang.org/LICENSE
/*
Copyright (c) 2009 The Go Authors. All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

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

// End copied part

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
