package config

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func TestServerBalancerStrategySurvivesBuild(t *testing.T) {
	ctx := include.Context(context.Background())
	content := `{"outbounds":[
{"type":"vless","tag":"node","server":"127.0.0.1","server_port":443,"uuid":"00000000-0000-4000-8000-000000000001"},
{"type":"balancer","tag":"region","outbounds":["node"],"strategy":"lowest-delay","tolerance":1,"delay_acceptable_ratio":2,"interrupt_exist_connections":true}]}`
	opts := DefaultSdmOptions()
	opts.BalancerStrategy = "round-robin"
	normalized, err := ParseConfigBytes(ctx, &ReadOptions{Content: content}, true, opts, false)
	if err != nil {
		t.Fatal(err)
	}
	built, err := BuildConfig(ctx, opts, &ReadOptions{Content: string(normalized)})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, out := range built.Outbounds {
		if out.Tag != "region" {
			if group, ok := out.Options.(*option.BalancerOutboundOptions); ok && group.Strategy == "lowest-delay" && group.Tolerance != 0 {
				t.Fatal("generated lowest-delay group must default to strict selection")
			}
			continue
		}
		group, ok := out.Options.(*option.BalancerOutboundOptions)
		if !ok || group.Strategy != "lowest-delay" || !group.InterruptExistConnections || group.Tolerance != 1 || group.DelayAcceptableRatio != 2 {
			t.Fatalf("server group options changed: %#v", out.Options)
		}
		found = true
	}
	if !found {
		t.Fatal("server group missing")
	}
}
