package config

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"io"

	"github.com/HZ-PRE/kuailei-core/v2/configvault"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
	SJ "github.com/sagernet/sing/common/json"
	"github.com/sdm/ray2sing/ray2sing"
	"github.com/xmdhs/clash2singbox/convert"
	clash2singmodel "github.com/xmdhs/clash2singbox/model"
	"github.com/xmdhs/clash2singbox/model/clash"
	"gopkg.in/yaml.v3"
)

//go:embed config.json.template
var configByte []byte

func ReadContent(ctx context.Context, opt *ReadOptions) ([]byte, error) {
	if opt.Content == "" {
		contentBytes, err := configvault.ReadFile(opt.Path)
		if err != nil {
			return nil, err
		}
		return contentBytes, nil
	}
	return []byte(opt.Content), nil
}

func ParseConfig(ctx context.Context, opt *ReadOptions, debug bool, configOpt *SdmOptions, fullConfig bool) (*option.Options, error) {
	content, err := ReadContent(ctx, opt)
	if err != nil {
		return nil, err
	}
	return parseConfigContent(ctx, content, debug, configOpt, fullConfig)
}

func ParseConfigBytes(ctx context.Context, opt *ReadOptions, debug bool, configOpt *SdmOptions, fullConfig bool) ([]byte, error) {

	options, err := ParseConfig(ctx, opt, debug, configOpt, fullConfig)
	if err != nil {
		return nil, err
	}

	return options.MarshalJSONContext(ctx)

}
func parseConfigContent(ctx context.Context, content []byte, debug bool, configOpt *SdmOptions, fullConfig bool) (*option.Options, error) {
	if configOpt == nil {
		configOpt = DefaultSdmOptions()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decoder := jsontext.NewDecoder(SJ.NewCommentFilter(bytes.NewReader(content)))
	kind := decoder.PeekKind()
	if kind == '{' || kind == '[' {
		normalized, err := normalizeJSONConfig(decoder, fullConfig || configOpt.EnableFullConfig)
		if err != nil {
			return nil, fmt.Errorf("[SingboxParser] invalid JSON: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return patchConfigStr(ctx, normalized, "SingboxParser", configOpt)
	}

	clashObj := clash.Clash{}
	if err := yaml.Unmarshal(content, &clashObj); err == nil && clashObj.Proxies != nil {
		if len(clashObj.Proxies) == 0 {
			return nil, fmt.Errorf("[ClashParser] no outbounds found")
		}
		converted, endpoints, err := convert.Clash2sing(clashObj, clash2singmodel.SINGLATEST)
		if err != nil {
			return nil, fmt.Errorf("[ClashParser] converting clash to sing-box error: %w", err)
		}
		output := configByte
		output, err = convert.Patch(output, converted, endpoints, "", "", nil)
		if err != nil {
			return nil, fmt.Errorf("[ClashParser] patching clash config error: %w", err)
		}
		return patchConfigStr(ctx, output, "ClashParser", configOpt)
	}

	v2ray, err := ray2sing.Ray2SingboxOptions(ctx, string(content), configOpt.UseXrayCoreWhenPossible)
	if err == nil {
		return patchConfigOptions(ctx, v2ray, "V2rayParser", configOpt)
	}

	return nil, fmt.Errorf("unable to determine config format")
}

// Preserve raw section values: node objects are decoded into typed options only
// once, and large integers never round-trip through float64. jsontext rejects
// duplicate names and invalid UTF-8, including inside discarded sections.
func normalizeJSONConfig(decoder *jsontext.Decoder, full bool) ([]byte, error) {
	var result []byte
	switch decoder.PeekKind() {
	case '[':
		value, err := decoder.ReadValue()
		if err != nil {
			return nil, err
		}
		result = append([]byte(`{"outbounds":`), value...)
		result = append(result, '}')
	case '{':
		var sections map[string]jsontext.Value
		if err := json.UnmarshalDecode(decoder, &sections); err != nil {
			return nil, err
		}
		if sections["outbounds"] == nil && sections["endpoints"] == nil {
			outbound, err := json.Marshal(sections)
			if err != nil {
				return nil, err
			}
			result = append([]byte(`{"outbounds":[`), outbound...)
			result = append(result, ']', '}')
		} else {
			if !full {
				var err error
				sections, err = selectBuildInputSections(sections)
				if err != nil {
					return nil, err
				}
			}
			var err error
			result, err = json.Marshal(sections)
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("expected a JSON object or array")
	}
	if _, err := decoder.ReadToken(); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return result, nil
}

func selectBuildInputSections(input map[string]jsontext.Value) (map[string]jsontext.Value, error) {
	selected := make(map[string]jsontext.Value, 3)
	for _, key := range []string{"outbounds", "endpoints"} {
		if value, exists := input[key]; exists {
			selected[key] = value
		}
	}
	if raw := input["route"]; len(raw) > 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		var route map[string]jsontext.Value
		if err := json.Unmarshal(raw, &route); err != nil {
			return nil, fmt.Errorf("route: %w", err)
		}
		selectedRoute := make(map[string]jsontext.Value, 2)
		for _, key := range []string{"rules", "rule_set"} {
			if value, exists := route[key]; exists {
				selectedRoute[key] = value
			}
		}
		if len(selectedRoute) > 0 {
			value, err := json.Marshal(selectedRoute)
			if err != nil {
				return nil, err
			}
			selected["route"] = value
		}
	}
	return selected, nil
}
func patchConfigStr(ctx context.Context, content []byte, name string, configOpt *SdmOptions) (*option.Options, error) {
	options := option.Options{}
	err := options.UnmarshalJSONContext(ctx, content)

	if err != nil {
		return nil, fmt.Errorf("[SingboxParser] unmarshal error: %w", err)
	}

	return patchConfigOptions(ctx, &options, name, configOpt)
}
func patchConfigOptions(ctx context.Context, options *option.Options, name string, configOpt *SdmOptions) (*option.Options, error) {
	if len(options.Endpoints) != 0 {
		return nil, fmt.Errorf("VPN endpoints are not supported")
	}
	if configOpt.Warp.EnableWarp || configOpt.Warp2.EnableWarp {
		return nil, fmt.Errorf("WARP is not supported")
	}
	return validateResult(ctx, options, name)
}

func validateResult(ctx context.Context, options *option.Options, name string) (*option.Options, error) {
	err := libbox.CheckConfigOptions(options)
	if err != nil {
		return nil, fmt.Errorf("[%s] invalid sing-box config: %w", name, err)
	}
	return options, nil
}
