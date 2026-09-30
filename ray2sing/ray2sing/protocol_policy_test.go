package ray2sing

import (
	"encoding/base64"
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

func TestRemovedLinksAreNeverSilentlyImported(t *testing.T) {
	const valid = "vless://00000000-0000-4000-8000-000000000001@127.0.0.1:443#retained"
	for _, scheme := range []string{"ss", "ssr", "vmess", "trojan", "hysteria", "hy2", "hysteria2", "tuic", "ssh", "naive", "socks", "http", "https", "wireguard", "wg", "warp", "awg", "xvless", "xvmess", "mieru", "psiphon", "dnstt"} {
		t.Run(scheme, func(t *testing.T) {
			removed := scheme + "://test@127.0.0.1:443"
			for _, text := range []string{removed, valid + "\n" + removed, removed + "\n" + valid} {
				for _, input := range []string{text, base64.StdEncoding.EncodeToString([]byte(text))} {
					out, err := GenerateConfigLite(input, false)
					if err == nil || out != nil {
						t.Fatal("removed protocol was silently accepted")
					}
				}
			}
		})
	}
}

func TestVLESSXHTTPImportRoundTrip(t *testing.T) {
	result, err := GenerateConfigLite("vless://00000000-0000-4000-8000-000000000001@127.0.0.1:443?type=xhttp&mode=auto&path=%2Faudit#xhttp", false)
	if err != nil {
		t.Fatal(err)
	}
	outbound, ok := result.Outbounds[0].Options.(*option.VLESSOutboundOptions)
	if !ok || outbound.Transport == nil {
		t.Fatal("VLESS transport missing")
	}
	encoded, err := json.Marshal(outbound.Transport)
	if err != nil {
		t.Fatal(err)
	}
	var restored option.V2RayTransportOptions
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Type != "xhttp" || restored.XHTTPOptions.XPaddingBytes.From != 100 {
		t.Fatal("XHTTP defaults lost during import")
	}
}

func TestVLESSImportOrder(t *testing.T) {
	input := "vless://00000000-0000-4000-8000-000000000001@127.0.0.1:443#first\nvless://00000000-0000-4000-8000-000000000002@127.0.0.1:443#second"
	for _, text := range []string{input, base64.StdEncoding.EncodeToString([]byte(input))} {
		result, err := GenerateConfigLite(text, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Outbounds) != 2 || result.Outbounds[0].Tag != "first § 0" || result.Outbounds[1].Tag != "second § 1" {
			t.Fatal("VLESS import order changed")
		}
	}
}
