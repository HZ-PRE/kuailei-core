package config

import (
	"context"
	"encoding/json"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func TestRoutingModeBuildAndSwitchBack(t *testing.T) {
	hopts := DefaultSdmOptions()
	hopts.BypassLAN = true
	hopts.BlockAds = true
	hopts.BlockQuic = true
	hopts.Region = "cn"
	hopts.EnableTun = true
	hopts.Rules = []Rule{{Enabled: true, Outbound: Outbound_direct, Domains: []string{"app.example"}}}
	content := `{
   "outbounds": [{"type":"socks","tag":"test-node","server":"127.0.0.1","server_port":1080}],
   "route": {
     "rules": [{"domain":["profile.example"],"action":"reject"}],
     "rule_set": [{"type":"remote","tag":"profile-set","format":"binary","url":"https://example.com/test.srs"}]
   }
 }`
	ctx := include.Context(context.Background())
	for _, mode := range []string{"", "global", "rule"} {
		t.Run(mode, func(t *testing.T) {
			if err := json.Unmarshal([]byte(`{"routing-mode":"`+mode+`"}`), hopts); err != nil {
				t.Fatal(err)
			}
			options, err := ParseBuildConfig(ctx, hopts, &ReadOptions{Content: content})
			if err != nil {
				t.Fatal(err)
			}
			if options.Route.Final != OutboundSelectTag {
				t.Fatalf("final = %q", options.Route.Final)
			}
			if !options.Route.AutoDetectInterface || options.Route.DefaultDomainResolver.Server != DNSDirectTag {
				t.Fatal("lost TUN loop prevention or node DNS bootstrap")
			}
			if mode == "global" {
				if len(options.Route.Rules) != 2 || len(options.Route.RuleSet) != 0 {
					t.Fatalf("business routing leaked into global mode: %+v", options.Route)
				}
				if options.Route.Rules[0].DefaultOptions.Action != C.RuleActionTypeSniff ||
					options.Route.Rules[1].DefaultOptions.Action != C.RuleActionTypeHijackDNS {
					t.Fatal("lost DNS sniffing/hijacking")
				}
				for _, rule := range options.DNS.Rules {
					r := rule.DefaultOptions
					if len(r.RuleSet) != 0 || len(r.DomainSuffix) != 0 || r.Action == C.RuleActionTypePredefined {
						t.Fatalf("business DNS rule leaked into global mode: %+v", r)
					}
				}
			} else {
				app, profile, lan, block := false, false, false, false
				for _, rule := range options.Route.Rules {
					r := rule.DefaultOptions
					app = app || (len(r.Domain) == 1 && r.Domain[0] == "app.example")
					profile = profile || (len(r.Domain) == 1 && r.Domain[0] == "profile.example")
					lan = lan || r.IPIsPrivate
					block = block || r.Action == C.RuleActionTypeReject
				}
				if !app || !profile || !lan || !block || len(options.Route.RuleSet) < 3 {
					t.Fatal("rule mode did not restore configured rules")
				}
			}
		})
	}
}

func TestGlobalModeUsesSelectedProxyAndPreservesFakeDNS(t *testing.T) {
	previous := OutboundMainDetour
	OutboundMainDetour = WARPConfigTag
	defer func() { OutboundMainDetour = previous }()
	hopts := DefaultSdmOptions()
	hopts.RoutingMode = "global"
	hopts.EnableFakeDNS = true
	// Invalid business rules must not be converted or loaded in global mode.
	hopts.Rules = []Rule{{Enabled: true, RuleSets: []string{"invalid-rule-set"}}}
	options := &option.Options{}
	ips := map[string][]string{}
	if err := setDns(options, hopts, &ips); err != nil {
		t.Fatal(err)
	}
	if err := setRoutingOptions(options, hopts, nil); err != nil {
		t.Fatal(err)
	}
	if options.Route.Final != OutboundSelectTag {
		t.Fatal("global mode bypassed current selection")
	}
	fake := false
	for _, rule := range options.DNS.Rules {
		fake = fake || rule.DefaultOptions.RouteOptions.Server == DNSFakeTag
	}
	if !fake {
		t.Fatal("fake DNS rule lost")
	}
	for _, server := range options.DNS.Servers {
		if server.Tag == DNSRemoteTag || server.Tag == DNSRemoteTagFallback {
			data, err := json.Marshal(server.Options)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]any
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			if fields["detour"] != OutboundSelectTag {
				t.Fatalf("DNS bypassed selection: %s", data)
			}
		}
	}
}
