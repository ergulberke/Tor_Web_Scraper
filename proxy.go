package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

type TorCheckResp struct {
	IsTor bool   `json:"IsTor"`
	IP    string `json:"IP"`
}

func NewTorHTTPClient(socksAddr string, timeout time.Duration) (*http.Client, error) {
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ResponseHeaderTimeout: 25 * time.Second,
	}

	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}, nil
}

func CheckTorIP(client *http.Client) (TorCheckResp, error) {
	req, err := http.NewRequest("GET", "https://check.torproject.org/api/ip", nil)
	if err != nil {
		return TorCheckResp{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return TorCheckResp{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return TorCheckResp{}, errors.New("tor check failed: non-2xx")
	}

	var out TorCheckResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return TorCheckResp{}, err
	}
	return out, nil
}

//port otomatik bulma

func isSocksAlive(addr string) bool {
	c, err := net.DialTimeout("tcp", addr, 1200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// önce 9050 sonra 9150 dene
func AutoDetectSocks(defaultAddr string) string {
	if defaultAddr != "" && defaultAddr != "auto" {
		if isSocksAlive(defaultAddr) {
			return defaultAddr
		}
	}

	candidates := []string{"127.0.0.1:9050", "127.0.0.1:9150"}
	for _, c := range candidates {
		if isSocksAlive(c) {
			return c
		}
	}

	if defaultAddr == "" || defaultAddr == "auto" {
		return "127.0.0.1:9050"
	}
	return defaultAddr
}
