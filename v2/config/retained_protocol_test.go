package config

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/include"
)

func TestEmptyConfigurationReturnsErrorInsteadOfPanicking(t *testing.T) {
	ctx := include.Context(context.Background())
	for _, content := range []string{`{"outbounds":[]}`, `{"outbounds":[{"type":"direct","tag":"direct"}]}`} {
		result, err := BuildConfig(ctx, DefaultSdmOptions(), &ReadOptions{Content: content})
		if err == nil || result != nil {
			t.Fatal("empty configuration must not build a selectable proxy")
		}
	}
}

func TestRemovedProtocolsRejectedByBothImportFormats(t *testing.T) {
	ctx := include.Context(context.Background())
	for _, kind := range []string{"shadowsocks", "vmess", "trojan", "hysteria2", "tuic", "http", "socks", "wireguard"} {
		t.Run(kind, func(t *testing.T) {
			for _, content := range []string{
				`{"outbounds":[{"type":"` + kind + `","server":"127.0.0.1","server_port":443}]}`,
				"proxies:\n  - name: removed\n    type: " + kind + "\n    server: 127.0.0.1\n    port: 443\n",
			} {
				result, err := ParseConfig(ctx, &ReadOptions{Content: content}, false, DefaultSdmOptions(), false)
				if err == nil || result != nil {
					t.Fatal("removed protocol was accepted")
				}
			}
		})
	}
}
