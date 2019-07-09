package i2pbrowserproxy

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	//"fmt"
	"net/http"
	//"time"
	"log"
)

// Create a struct that models the structure of a user, both in the request body, and in the DB
type Credentials struct {
	Site string `json:"identity"`
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func DecodeIdentity(body *http.Request) (*http.Request, *Credentials, error) {
	var creds Credentials
	bb, err := ioutil.ReadAll(body.Body)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(body.Method, body.URL.String(), bytes.NewReader(bb))
	if err != nil {
		return nil, nil, err
	}

	err = json.NewDecoder(bytes.NewReader(bb)).Decode(&creds)
	if err != nil {
		return nil, nil, err
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
