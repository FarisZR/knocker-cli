package util

import (
	"io/ioutil"
	"net/http"
	"strings"
)

type IPGetter interface {
	GetPublicIP(url string) (string, error)
}

type ipGetter struct{}

func NewIPGetter() IPGetter {
	return &ipGetter{}
}

func (g *ipGetter) GetPublicIP(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(ip)), nil
}