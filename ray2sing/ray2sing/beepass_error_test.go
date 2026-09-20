package ray2sing

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFetchSSConfStatusAndQuery(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != "test" {
			t.Error("subscription query was lost")
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	transport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = transport }()
	u, _ := url.Parse(strings.Replace(server.URL, "https://", "ssconf://", 1) + "?token=test#label")
	content, err := fetchSSConf(u)
	if err == nil || !strings.Contains(err.Error(), "403") || content != nil {
		t.Fatalf("expected HTTP status error: %q, %v", content, err)
	}
}
