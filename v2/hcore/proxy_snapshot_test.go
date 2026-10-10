package hcore

import (
	"fmt"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
)

type snapshotGroup struct {
	*countryOutbound
	tags     []string
	selected string
}

func (g *snapshotGroup) All() []string { return g.tags }
func (g *snapshotGroup) Now() string   { return g.selected }

func snapshotFixture(count int) ([]adapter.Outbound, func(string) (adapter.Outbound, bool)) {
	nodes := make(map[string]adapter.Outbound)
	var outbounds []adapter.Outbound
	var tags []string
	for i := 0; i < count; i++ {
		tag := fmt.Sprintf("node-%d", i)
		node := &countryOutbound{Adapter: outbound.NewAdapter("vless", tag, []string{"tcp"}, nil)}
		nodes[tag] = node
		outbounds = append(outbounds, node)
		tags = append(tags, tag)
	}
	for i := 0; i < 2; i++ {
		group := &snapshotGroup{countryOutbound: &countryOutbound{Adapter: outbound.NewAdapter("balancer", fmt.Sprintf("group-%d", i), []string{"tcp"}, nil)}, tags: tags, selected: tags[i]}
		nodes[group.Tag()] = group
		outbounds = append(outbounds, group)
	}
	return outbounds, func(tag string) (adapter.Outbound, bool) { n, ok := nodes[tag]; return n, ok }
}

func TestMainSnapshotOnlyBuildsSelectedMetadata(t *testing.T) {
	nodes, lookup := snapshotFixture(1000)
	calls := 0
	h := new(SdmInstance)
	result := collectProxyGroups(nodes, lookup, func(n adapter.Outbound) *OutboundInfo { calls++; return h.GetProxyInfo(nil, n) }, nil, true)
	if calls != 2 || len(result.Items) != 2 {
		t.Fatalf("metadata calls=%d groups=%d", calls, len(result.Items))
	}
	for _, g := range result.Items {
		if len(g.Items) != 1 || g.Items[0].Tag != g.Selected || !g.Items[0].IsSelected {
			t.Fatal("wrong selected member")
		}
	}
}

func TestSnapshotSharedNodesKeepIndependentSelection(t *testing.T) {
	nodes, lookup := snapshotFixture(3)
	h := new(SdmInstance)
	result := collectProxyGroups(nodes, lookup, func(n adapter.Outbound) *OutboundInfo { return h.GetProxyInfo(nil, n) }, nil, false)
	if !result.Items[0].Items[0].IsSelected || result.Items[1].Items[0].IsSelected {
		t.Fatal("group selection flags alias")
	}
	if result.Items[0].Items[1].IsSelected || !result.Items[1].Items[1].IsSelected {
		t.Fatal("second selection incorrect")
	}
	if result.Items[0].Items[0] == result.Items[1].Items[0] {
		t.Fatal("mutable protobuf shared")
	}
}

func BenchmarkMainProxySnapshot(b *testing.B) {
	for _, count := range []int{100, 1000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			nodes, lookup := snapshotFixture(count)
			h := new(SdmInstance)
			info := func(n adapter.Outbound) *OutboundInfo { return h.GetProxyInfo(nil, n) }
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				collectProxyGroups(nodes, lookup, info, nil, true)
			}
		})
	}
}
