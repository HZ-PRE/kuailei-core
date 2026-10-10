package hcore

import (
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/sdm/ipinfo"
	"testing"
)

type countryOutbound struct {
	adapter.Outbound
	outbound.Adapter
}

// Explicit methods resolve the embedded interface/adapter ambiguity.
func (n *countryOutbound) Tag() string            { return n.Adapter.Tag() }
func (n *countryOutbound) Type() string           { return n.Adapter.Type() }
func (n *countryOutbound) Network() []string      { return n.Adapter.Network() }
func (n *countryOutbound) Dependencies() []string { return nil }
func (n *countryOutbound) DisplayType() string    { return n.Adapter.DisplayType() }
func (n *countryOutbound) IsReady() bool          { return true }

func TestConfiguredCountryIndependentOfCachedIP(t *testing.T) {
	node := &countryOutbound{Adapter: outbound.NewAdapter("vless", "node", []string{"tcp"}, nil)}
	node.SetCountryCode(" jp ")
	h := &SdmInstance{}
	result := h.GetProxyInfo(&adapter.URLTestHistory{Delay: 42, IpInfo: &ipinfo.IpInfo{IP: "192.0.2.1", CountryCode: "US"}}, node)
	if result.CountryCode != "JP" {
		t.Fatal("configured flag was replaced by IP location")
	}
	if result.Ipinfo != nil {
		t.Fatal("cached IP leaked without permission")
	}
	if result.UrlTestDelay != 42 {
		t.Fatal("latency was removed")
	}
}

type countryGroup struct {
	*countryOutbound
	selected string
	members  []string
}

func (g *countryGroup) Now() string { return g.selected }
func (g *countryGroup) All() []string {
	if g.members != nil {
		return g.members
	}
	return []string{g.selected}
}

func TestConfiguredCountryFollowsNestedSelection(t *testing.T) {
	node := func(tag, code string) *countryOutbound {
		out := &countryOutbound{Adapter: outbound.NewAdapter("vless", tag, []string{"tcp"}, nil)}
		out.SetCountryCode(code)
		return out
	}
	automatic := &countryGroup{countryOutbound: node("automatic", ""), selected: "jp"}
	selector := &countryGroup{countryOutbound: node("select", ""), selected: "automatic"}
	nodes := map[string]adapter.Outbound{"jp": node("jp", "JP"), "sg": node("sg", "SG"), "automatic": automatic, "select": selector}
	lookup := func(tag string) (adapter.Outbound, bool) { out, ok := nodes[tag]; return out, ok }
	if got := configuredCountryCode(selector, lookup); got != "JP" {
		t.Fatalf("nested group flag: %s", got)
	}
	automatic.selected = "sg"
	if got := configuredCountryCode(selector, lookup); got != "SG" {
		t.Fatalf("stale group flag: %s", got)
	}
	automatic.selected = "select"
	if got := configuredCountryCode(selector, lookup); got != "" {
		t.Fatal("cycle should not invent a country")
	}
}

func TestConfiguredGroupCountryIsAuthoritative(t *testing.T) {
	node := func(tag, code string) *countryOutbound {
		out := &countryOutbound{Adapter: outbound.NewAdapter("ssplus", tag, []string{"tcp", "udp"}, nil)}
		out.SetCountryCode(code)
		return out
	}
	hk := &countryGroup{countryOutbound: node("香港", " hk "), selected: "other", members: []string{"other"}}
	fastest := &countryGroup{countryOutbound: node("极速模式", ""), selected: "香港"}
	automatic := &countryGroup{countryOutbound: node("自动选择", ""), members: []string{"香港"}}
	nodes := map[string]adapter.Outbound{"香港": hk, "other": node("other", "US")}
	lookups := 0
	lookup := func(tag string) (adapter.Outbound, bool) { lookups++; out, ok := nodes[tag]; return out, ok }
	if got := configuredCountryCode(hk, lookup); got != "HK" || lookups != 0 {
		t.Fatalf("configured group should not inspect members: code=%q lookups=%d", got, lookups)
	}
	if got := configuredCountryCode(fastest, lookup); got != "HK" {
		t.Fatalf("generated selection group must use selected group's configured country: %q", got)
	}
	hk.selected = ""
	if got := configuredCountryCode(hk, lookup); got != "HK" {
		t.Fatalf("configured round-robin group country = %q", got)
	}
	lookups = 0
	if got := configuredCountryCode(automatic, lookup); got != "" || lookups != 0 {
		t.Fatalf("unlabelled round-robin group should not infer members country: code=%q lookups=%d", got, lookups)
	}
	fastest.selected = "missing"
	if got := configuredCountryCode(fastest, lookup); got != "" {
		t.Fatalf("missing selection country = %q", got)
	}
	if got := configuredCountryCode(nil, lookup); got != "" {
		t.Fatalf("nil outbound country = %q", got)
	}
}
