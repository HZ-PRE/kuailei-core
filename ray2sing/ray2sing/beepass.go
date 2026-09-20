package ray2sing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	T "github.com/sagernet/sing-box/option"
)

type beepassData struct {
	Server     string `json:"server"`
	ServerPort string `json:"server_port"`
	Password   string `json:"password"`
	Method     string `json:"method"`
	Prefix     string `json:"prefix"`
	Name       string `json:"name"`
}

func fetchSSConf(parsedURL *url.URL) ([]byte, error) {

	// Construct the HTTP URL
	httpURL := *parsedURL
	httpURL.Scheme = "https"
	httpURL.Fragment = ""

	// Make the HTTP request
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(httpURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ssconf request failed: HTTP %d", resp.StatusCode)
	}

	// Read the response body
	const maxConfigSize = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxConfigSize+1))
	if err != nil {
		return nil, err
	}

	if len(body) > maxConfigSize {
		return nil, fmt.Errorf("ssconf response exceeds 1 MiB")
	}
	return body, nil
}
func parseAndFetchBeePass(body []byte) (*beepassData, error) {

	// Decode JSON
	var config beepassData
	err := json.Unmarshal(body, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func BeepassSingbox(beepassUrl string) (*T.Outbound, error) {
	parsedURL, err := url.Parse(beepassUrl)
	if err != nil {
		return nil, err
	}
	body, err := fetchSSConf(parsedURL)
	if err != nil {
		return nil, err
	}
	decoded, err := parseAndFetchBeePass(body)
	if err != nil {
		return ShadowsocksSingbox(strings.TrimSpace(string(body)))
		// return nil, err
	}
	if decoded.Name == "" {
		decoded.Name = parsedURL.Fragment
	}
	result := T.Outbound{
		Type: "shadowsocks",
		Tag:  decoded.Name,
		Options: &T.ShadowsocksOutboundOptions{
			ServerOptions: T.ServerOptions{
				Server:     decoded.Server,
				ServerPort: toUInt16(decoded.ServerPort, 443),
			},
			Method:   decoded.Method,
			Password: decoded.Password,
		},
	}

	return &result, nil
}
