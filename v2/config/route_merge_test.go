package config

import (
	"context"
	"slices"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
)

func TestSelectBuildInputSectionsKeepsRouteRules(t *testing.T) {
	input := map[string]interface{}{
		"log":       map[string]interface{}{"level": "debug"},
		"outbounds": []interface{}{map[string]interface{}{"type": "direct", "tag": "proxy"}},
		"route": map[string]interface{}{
			"rules":    []interface{}{map[string]interface{}{"domain": []interface{}{"profile.example"}}},
			"rule_set": []interface{}{map[string]interface{}{"type": "remote", "tag": "profile-set"}},
			"final":    "proxy",
		},
	}

	selected := selectBuildInputSections(input)
	if _, exists := selected["log"]; exists {
		t.Fatal("unrelated full-config section should not be retained")
	}
	route, ok := selected["route"].(map[string]interface{})
	if !ok {
		t.Fatal("route section was not retained")
	}
	if route["rules"] == nil || route["rule_set"] == nil {
		t.Fatalf("route rules or rule sets were not retained: %#v", route)
	}
	if _, exists := route["final"]; exists {
		t.Fatal("route.final should continue to be managed by the app")
	}
}

func TestSetRoutingOptionsMergesAppAndProfileRules(t *testing.T) {
	hopts := DefaultSdmOptions()
	hopts.Rules = []Rule{{
		Enabled:          true,
		Name:             "app rule",
		Outbound:         Outbound_direct,
		RuleSets:         []string{"https://example.com/app.srs"},
		PackageNames:     []string{"app.example"},
		ProcessNames:     []string{"example.exe"},
		ProcessPaths:     []string{"C:/example.exe"},
		Network:          Network_tcp,
		PortRanges:       []string{"80:443"},
		SourcePortRanges: []string{"1000:2000"},
		Protocols:        []Protocol{Protocol_http},
		IpCidrs:          []string{"10.0.0.0/8"},
		SourceIpCidrs:    []string{"192.168.0.0/16"},
		Domains:          []string{"app.example"},
	}}
	profileRule := option.Rule{
		Type: C.RuleTypeDefault,
		DefaultOptions: option.DefaultRule{
			RawDefaultRule: option.RawDefaultRule{Domain: []string{"profile.example"}},
			RuleAction: option.RuleAction{
				Action:       C.RuleActionTypeRoute,
				RouteOptions: option.RouteActionOptions{Outbound: "profile-proxy"},
			},
		},
	}
	profileRuleSet := option.RuleSet{Type: C.RuleSetTypeInline, Tag: "profile-set"}
	options := &option.Options{DNS: &option.DNSOptions{}}

	if err := setRoutingOptions(options, hopts, &option.RouteOptions{
		Rules:   []option.Rule{profileRule},
		RuleSet: []option.RuleSet{profileRuleSet},
	}); err != nil {
		t.Fatalf("set routing options: %v", err)
	}

	appIndex, profileIndex := -1, -1
	for index, rule := range options.Route.Rules {
		if slices.Equal(rule.DefaultOptions.Domain, []string{"app.example"}) {
			appIndex = index
			got := rule.DefaultOptions.RawDefaultRule
			if !slices.Equal(got.Network, []string{"tcp"}) ||
				!slices.Equal(got.Protocol, []string{"http"}) ||
				!slices.Equal(got.IPCIDR, []string{"10.0.0.0/8"}) ||
				!slices.Equal(got.PackageName, []string{"app.example"}) ||
				len(got.RuleSet) != 1 {
				t.Fatalf("app rule fields were not fully converted: %#v", got)
			}
		}
		if slices.Equal(rule.DefaultOptions.Domain, []string{"profile.example"}) {
			profileIndex = index
		}
	}
	if appIndex == -1 || profileIndex == -1 || appIndex >= profileIndex {
		t.Fatalf("expected app rule before profile rule, got app=%d profile=%d", appIndex, profileIndex)
	}

	knownRuleSets := make(map[string]bool)
	for _, ruleSet := range options.Route.RuleSet {
		knownRuleSets[ruleSet.Tag] = true
	}
	if !knownRuleSets["profile-set"] || len(knownRuleSets) != 2 {
		t.Fatalf("expected app and profile rule sets, got %#v", knownRuleSets)
	}
}

func TestParseBuildConfigMergesConfigRouteRules(t *testing.T) {
	hopts := DefaultSdmOptions()
	hopts.Rules = []Rule{{
		Enabled:  true,
		Name:     "app rule",
		Outbound: Outbound_direct,
		Domains:  []string{"app.example"},
	}}
	content := `{
		"outbounds": [{"type": "direct", "tag": "profile-proxy"}],
		"route": {
			"rules": [{"domain": ["profile.example"], "outbound": "profile-proxy"}]
		}
	}`

	options, err := ParseBuildConfig(include.Context(context.Background()), hopts, &ReadOptions{Content: content})
	if err != nil {
		t.Fatalf("parse and build config: %v", err)
	}

	appIndex, profileIndex := -1, -1
	for index, rule := range options.Route.Rules {
		if slices.Equal(rule.DefaultOptions.Domain, []string{"app.example"}) {
			appIndex = index
		}
		if slices.Equal(rule.DefaultOptions.Domain, []string{"profile.example"}) {
			profileIndex = index
			if rule.DefaultOptions.RouteOptions.Outbound != "profile-proxy" {
				t.Fatalf("profile rule outbound changed: %q", rule.DefaultOptions.RouteOptions.Outbound)
			}
		}
	}
	if appIndex == -1 || profileIndex == -1 || appIndex >= profileIndex {
		t.Fatalf("expected app rule before config rule, got app=%d config=%d", appIndex, profileIndex)
	}
}
