package hcore

import (
	"strings"
	"time"

	hcommon "github.com/HZ-PRE/kuailei-core/v2/hcommon"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/monitoring"
	G "github.com/sagernet/sing-box/protocol/group"
	"github.com/sagernet/sing-box/protocol/group/balancer"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func (h *SdmInstance) GetProxyInfo(url_test_history *adapter.URLTestHistory, detour adapter.Outbound) *OutboundInfo {
	// historyStorage := h.UrlTestHistory()
	// if historyStorage == nil {
	// 	return nil
	// }

	out := &OutboundInfo{}
	// realTag := ""

	out.Tag = detour.Tag()
	out.Type = detour.DisplayType()
	if group, isGroup := detour.(adapter.OutboundGroup); isGroup {
		out.IsGroup = true
		gnow := group.Now()
		out.GroupSelectedTag = &gnow
	}
	out.TagDisplay = TrimTagName(out.Tag)
	out.CountryCode = h.outboundCountryCode(detour)

	if tag := monitoring.RealTag(detour); tag != "" {
		dtag := TrimTagName(tag)
		out.GroupSelectedTagDisplay = &dtag
		if balancer, ok := detour.(*balancer.Balancer); ok {
			if stg := balancer.Strategy(); stg != "lowest-delay" {
				out.GroupSelectedTagDisplay = &stg
			}
		}
	}
	// realTag = adapter.OutboundTag(detour)

	// realTag = out.Tag

	// url_test_history := historyStorage.LoadURLTestHistory(realTag)
	if trafficManager := h.TrafficManager(); trafficManager != nil {
		up, down := trafficManager.OutboundUsage(out.Tag)
		out.Upload = up
		out.Download = down

	}
	if url_test_history != nil {
		out.UrlTestTime = timestamppb.New(url_test_history.Time)
		out.UrlTestDelay = int32(url_test_history.Delay)
		if url_test_history.IsFromCache {
			out.UrlTestDelay = 0
		}
		if h.nodeIPInfoEnabled() && url_test_history.IpInfo != nil {
			out.Ipinfo = &IpInfo{
				Ip:          url_test_history.IpInfo.IP,
				CountryCode: url_test_history.IpInfo.CountryCode,
				Region:      url_test_history.IpInfo.Region,
				City:        url_test_history.IpInfo.City,
				Asn:         int32(url_test_history.IpInfo.ASN),
				Org:         url_test_history.IpInfo.Org,
				Latitude:    url_test_history.IpInfo.Latitude,
				Longitude:   url_test_history.IpInfo.Longitude,
				PostalCode:  url_test_history.IpInfo.PostalCode,
			}
		}

	}
	if deps := detour.Dependencies(); len(deps) == 1 {
		out.Detour = deps[0]
	}
	return out
}

func (h *SdmInstance) nodeIPInfoEnabled() bool {
	if ctx := h.Context(); ctx != nil {
		if monitor := monitoring.Get(ctx); monitor != nil {
			return monitor.NodeIPInfoEnabled()
		}
	}
	return false
}

func (h *SdmInstance) outboundCountryCode(detour adapter.Outbound) string {
	if box := h.Box(); box != nil {
		return configuredCountryCode(detour, box.Outbound().Outbound)
	}
	return configuredCountryCode(detour, nil)
}

func configuredCountryCode(detour adapter.Outbound, lookup func(string) (adapter.Outbound, bool)) string {
	// Server metadata is authoritative, including a group's own country code.
	// Only unlabelled selection groups follow their current selection. Do not
	// scan members: round-robin groups have no single selected country.
	for depth := 0; detour != nil && depth < 32; depth++ {
		if metadata, ok := detour.(interface{ CountryCode() string }); ok && metadata.CountryCode() != "" {
			return metadata.CountryCode()
		}
		group, ok := detour.(adapter.OutboundGroup)
		if !ok || lookup == nil {
			return ""
		}
		selected := group.Now()
		if selected == "" || selected == detour.Tag() {
			return ""
		}
		detour, _ = lookup(selected)
	}
	return ""
}
func (h *SdmInstance) GetAllProxiesInfo(history func(string) *adapter.URLTestHistory, onlyGroupitems bool) *OutboundGroupList {
	ctx, box := h.Context(), h.Box()
	if ctx == nil || box == nil {
		return nil
	}
	return collectProxyGroups(box.Outbound().Outbounds(), box.Outbound().Outbound,
		func(node adapter.Outbound) *OutboundInfo { return h.GetProxyInfo(history(node.Tag()), node) },
		service.FromContext[adapter.CacheFile](ctx), onlyGroupitems)
}

// Build metadata lazily: the current-node RPC needs only selected members,
// not an OutboundInfo and timestamp for every configured node.
func collectProxyGroups(outbounds []adapter.Outbound, lookup func(string) (adapter.Outbound, bool),
	info func(adapter.Outbound) *OutboundInfo, cache adapter.CacheFile, onlySelected bool) *OutboundGroupList {
	converted := make(map[string]*OutboundInfo)
	memberInfo := func(tag string) *OutboundInfo {
		if cached, ok := converted[tag]; ok {
			// Selection belongs to the containing group. Sharing a mutable message
			// here would let a later group overwrite an earlier group's flags.
			return proto.Clone(cached).(*OutboundInfo)
		}
		node, ok := lookup(tag)
		if !ok {
			return nil
		}
		result := info(node)
		if result.Detour != "" {
			if detour, ok := lookup(result.Detour); ok {
				result.TagDisplay += " → " + TrimTagName(detour.Tag())
				result.Type += " → " + detour.DisplayType()
			}
		}
		if onlySelected && result.GroupSelectedTagDisplay != nil && result.TagDisplay != *result.GroupSelectedTagDisplay {
			result.TagDisplay += " → " + *result.GroupSelectedTagDisplay
		}
		converted[tag] = result
		return result
	}
	result := &OutboundGroupList{}
	for _, node := range outbounds {
		source, ok := node.(adapter.OutboundGroup)
		if !ok {
			continue
		}
		selected := source.Now()
		group := &OutboundGroup{Tag: source.Tag(), Type: source.Type(), Selected: selected}
		_, group.Selectable = source.(*G.Selector)
		if cache != nil {
			group.IsExpand, _ = cache.LoadGroupExpand(group.Tag)
		}
		tags := source.All()
		if onlySelected {
			if selected == "" {
				continue
			}
			tags = []string{selected}
		}
		for _, tag := range tags {
			member := memberInfo(tag)
			if member == nil {
				continue
			}
			member.IsSelected = tag == selected
			member.IsVisible = !strings.Contains(tag, "§hide§")
			group.Items = append(group.Items, member)
		}
		if len(group.Items) > 0 {
			result.Items = append(result.Items, group)
		}
	}
	return result
}

func TrimTagName(tag string) string {
	return strings.Trim(strings.SplitN(tag, "§", 2)[0], " ")
}

func (s *CoreService) OutboundsInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
	return static.AllProxiesInfoStream(stream, false)
}

func (s *CoreService) MainOutboundsInfo(req *hcommon.Empty, stream grpc.ServerStreamingServer[OutboundGroupList]) error {
	return static.AllProxiesInfoStream(stream, true)
}

func (h *SdmInstance) AllProxiesInfoStream(stream grpc.ServerStreamingServer[OutboundGroupList], onlyMain bool) error {
	// stream.Send(&OutboundGroupList{})
	h.MakeSureContextIsNew(stream.Context())

	if ctx, urlTestHistory := h.Context(), h.UrlTestHistory(); ctx != nil && urlTestHistory != nil {
		monitor := monitoring.Get(ctx)

		stream.Send(h.GetAllProxiesInfo(monitor.OutboundHistory, onlyMain))

		urltestch, err := monitor.SubscribeGroup("")
		if err != nil {
			Log(LogLevel_ERROR, LogType_CORE, "failed to send outbounds info: ", err)
			// return err
		}
		defer monitor.UnsubscribeGroup("", urltestch)

		// timer2 := time.NewTicker(10 * time.Second)
		// defer timer2.Stop()
		debounceWindow := 1000 * time.Millisecond
		var (
			timer   *time.Timer
			timerCh <-chan time.Time
		)
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()
		for {
			select {
			case <-stream.Context().Done():
				return nil
			case <-ctx.Done():
				return nil
			case _, ok := <-urltestch:
				if !ok {
					return nil
				}
				if timer == nil {
					timer = time.NewTimer(debounceWindow)
					timerCh = timer.C
				}
			case <-timerCh:
				if err := stream.Send(h.GetAllProxiesInfo(monitor.OutboundHistory, onlyMain)); err != nil {
					Log(LogLevel_ERROR, LogType_CORE, "failed to send outbounds info: ", err)
					// return err
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer = nil
				timerCh = nil
			}
		}
	}

	return E.New("sdm service not found")
}
