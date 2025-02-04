package router

import (
	"io"
	"net/http"
)

var serverHost string = ""
var serverKey string = ""

func InitializeHost(host, key string) {
	serverHost = host
	serverKey = key
}

func getUrl(url string) string {
	return serverHost + url
}

func NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, getUrl(url), body)
	if err != nil {
		return nil, err
	}
	// set serverKey
	return req, nil
}
