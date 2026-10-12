package config

import (
	"encoding/json"
	"net/netip"
	"strings"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func TestPortSwitchesFromAppJSON(t *testing.T) {
	opts := DefaultSdmOptions()
	err := json.Unmarshal([]byte(`{"enable-mixed-port":false,"enable-tproxy-port":false,"enable-redirect-port":false,"enable-direct-port":false}`), opts)
	if err != nil {
		t.Fatal(err)
	}
	var built option.Options
	if err = setInbound(&built, opts); err != nil {
		t.Fatal(err)
	}
	if len(built.Inbounds) != 0 {
		t.Fatalf("disabled ports still created inbounds: %+v", built.Inbounds)
	}
	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"enable-mixed-port":false`) {
		t.Fatal("explicit false lost during serialization")
	}
}

func TestLocalInboundDefaultsAndOptIn(t *testing.T) {
	for _, tun := range []bool{false, true} {
		opts := DefaultSdmOptions()
		opts.EnableTun = tun
		var built option.Options
		if err := setInbound(&built, opts); err != nil {
			t.Fatal(err)
		}
		for _, inbound := range built.Inbounds {
			if tun {
				if inbound.Type != C.TypeTun {
					t.Fatalf("TUN default opened %s", inbound.Type)
				}
			} else {
				if inbound.Type != C.TypeMixed {
					t.Fatalf("default opened optional %s", inbound.Type)
				}
				listen := inbound.Options.(*option.HTTPMixedInboundOptions).Listen
				if !netip.Addr(*listen).IsLoopback() {
					t.Fatal("non-loopback mixed listener")
				}
			}
		}
	}
	opts := DefaultSdmOptions()
	on, off := true, false
	opts.EnableMixedPort, opts.EnableDirectPort = &off, &on
	var built option.Options
	if err := setInbound(&built, opts); err != nil {
		t.Fatal(err)
	}
	if len(built.Inbounds) == 0 {
		t.Fatal("explicit DNS port opt-in ignored")
	}
	for _, inbound := range built.Inbounds {
		if inbound.Type != C.TypeDirect {
			t.Fatalf("unexpected inbound %s", inbound.Type)
		}
		if !netip.Addr(*inbound.Options.(*option.DirectInboundOptions).Listen).IsLoopback() {
			t.Fatal("non-loopback direct listener")
		}
	}
}

func TestUnsafeInboundConfigurationRejected(t *testing.T) {
	opts := DefaultSdmOptions()
	opts.AllowConnectionFromLAN = true
	if err := setInbound(new(option.Options), opts); err == nil {
		t.Fatal("LAN sharing accepted")
	}
	opts.AllowConnectionFromLAN = false
	off := false
	opts.EnableMixedPort, opts.SetSystemProxy = &off, true
	if err := setInbound(new(option.Options), opts); err == nil {
		t.Fatal("system proxy without listener accepted")
	}
	opts.SetSystemProxy = false
	opts.EnableMixedPort, opts.MixedPort = nil, 0
	var built option.Options
	if err := setInbound(&built, opts); err != nil {
		t.Fatal(err)
	}
	if len(built.Inbounds) != 0 {
		t.Fatal("zero port created an ephemeral listener")
	}
}

func TestClashSecretAndLegacyRPC(t *testing.T) {
	seen := make(map[string]bool)
	for range 16 {
		opts := DefaultSdmOptions()
		var built option.Options
		if err := setExperimental(&built, opts); err != nil {
			t.Fatal(err)
		}
		secret := built.Experimental.ClashAPI.Secret
		if len(secret) != 32 || seen[secret] {
			t.Fatal("invalid/repeated generated secret")
		}
		seen[secret] = true
		if built.Experimental.ClashAPI.ExternalController != "127.0.0.1:16756" {
			t.Fatal("management listener exposed")
		}
	}
	opts := DefaultSdmOptions()
	opts.ClashApiSecret = "sdm"
	if err := setExperimental(new(option.Options), opts); err == nil {
		t.Fatal("weak secret accepted")
	}
	if err := StartGRPCServer(0); err == nil {
		t.Fatal("legacy unauthenticated server started")
	}
}
