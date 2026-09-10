package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestHiddifyOptionsUnmarshalRules(t *testing.T) {
	input := []byte(`{
		"rules": [{
			"list_order": 1,
			"enabled": true,
			"name": "custom route",
			"outbound": "direct",
			"rule_sets": ["https://example.com/rules.srs"],
			"network": 1,
			"protocols": [0, 1],
			"ip_cidrs": ["10.0.0.0/8"],
			"domains": ["example.com"],
			"domain_suffixes": [".example.org"]
		}]
	}`)

	var options HiddifyOptions
	if err := json.Unmarshal(input, &options); err != nil {
		t.Fatalf("unmarshal hiddify options: %v", err)
	}
	if len(options.Rules) != 1 {
		t.Fatalf("expected one rule, got %d", len(options.Rules))
	}

	rule := options.Rules[0]
	if !rule.Enabled || rule.Name != "custom route" || rule.Outbound != "direct" {
		t.Fatalf("unexpected rule metadata: %+v", rule)
	}
	if rule.Network != Network_tcp {
		t.Fatalf("expected tcp network, got %v", rule.Network)
	}
	if !reflect.DeepEqual(rule.Protocols, []Protocol{Protocol_tls, Protocol_http}) {
		t.Fatalf("unexpected protocols: %v", rule.Protocols)
	}
	if !reflect.DeepEqual(rule.Domains, []string{"example.com"}) {
		t.Fatalf("unexpected domains: %v", rule.Domains)
	}
	if !reflect.DeepEqual(rule.IpCidrs, []string{"10.0.0.0/8"}) {
		t.Fatalf("unexpected IP CIDRs: %v", rule.IpCidrs)
	}
}
