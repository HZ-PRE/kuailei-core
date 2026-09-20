package config

import (
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
)

func TestParseConfigSingleOutboundAndArray(t *testing.T) {
	for _, content := range []string{
		`{"type":"direct","tag":"test"}`,
		`[{"type":"direct","tag":"test"}]`,
	} {
		opts, err := ParseConfig(libbox.BaseContext(nil), &ReadOptions{Content: content}, false, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(opts.Outbounds) != 1 || opts.Outbounds[0].Tag != "test" {
			t.Fatalf("lost outbound: %+v", opts.Outbounds)
		}
	}
}

func TestParseConfigPreservesRequestedFullConfig(t *testing.T) {
	for _, enableOption := range []bool{false, true} {
		hopts := DefaultSdmOptions()
		hopts.EnableFullConfig = enableOption
		opts, err := ParseConfig(libbox.BaseContext(nil), &ReadOptions{Content: `{"log":{"level":"error"},"outbounds":[{"type":"direct"}]}`,
		}, false, hopts, !enableOption)
		if err != nil {
			t.Fatal(err)
		}
		if opts.Log == nil || opts.Log.Level != "error" {
			t.Fatal("full-config flag/options were discarded")
		}
	}
}
