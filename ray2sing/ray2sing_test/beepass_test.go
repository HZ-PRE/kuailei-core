package ray2sing_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HZ-PRE/ray2sing/ray2sing"
)

func TestBeePass(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"server":"beacomf.xyz","server_port":"8080","method":"chacha20-ietf-poly1305","password":"test-password"}`))
	}))
	defer server.Close()
	transport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = transport }()
	url := strings.Replace(server.URL, "https://", "ssconf://", 1) + "#BeePass"

	// Define the expected JSON structure
	expectedJSON := `{
		"outbounds": [
			{
				"type": "shadowsocks",
				"tag": "BeePass § 0",
				"server": "beacomf.xyz",
				"server_port": 8080,
				"method": "chacha20-ietf-poly1305",
				"password": "test-password"
			}
		]
	}`
	ray2sing.CheckUrlAndJson(url, expectedJSON, t)
}
