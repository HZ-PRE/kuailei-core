package ray2sing

import (
	"context"
	"fmt"
	"runtime"

	"strconv"
	"strings"

	_ "github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	T "github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

var configTypes = map[string]ParserFunc{"vless://": VlessSingbox, "svless://": VlessSingbox}

func GenerateConfigLite(input string, _ bool) (*option.Options, error) {
	outbounds := []T.Outbound{}
	for _, config := range expandDecodedConfig(input) {
		if strings.Contains(config, " -> ") {
			return nil, fmt.Errorf("proxy chains are not supported")
		}
		var parser ParserFunc
		for prefix, candidate := range configTypes {
			if strings.HasPrefix(config, prefix) {
				parser = candidate
				break
			}
		}
		if parser == nil {
			return nil, fmt.Errorf("unsupported proxy protocol; use SS+ JSON or VLESS")
		}
		out, err := parser(config)
		if err != nil {
			return nil, fmt.Errorf("invalid VLESS configuration: %w", err)
		}
		if out.Tag == "" {
			out.Tag = out.Type
		}
		out.Tag += " § " + strconv.Itoa(len(outbounds))
		outbounds = append(outbounds, *out)
	}
	if len(outbounds) == 0 {
		return nil, fmt.Errorf("no outbounds found")
	}
	return &option.Options{Outbounds: outbounds}, nil
}

func Ray2Singbox(ctx context.Context, configs string, useXrayWhenPossible bool) (out []byte, err error) {
	convertedData, err := Ray2SingboxOptions(ctx, configs, useXrayWhenPossible)
	if err != nil {
		return nil, err
	}
	return convertedData.MarshalJSONContext(ctx)
}
func Ray2SingboxOptions(ctx context.Context, configs string, useXrayWhenPossible bool) (out *option.Options, err error) {
	defer func() {
		if r := recover(); r != nil {
			out = nil
			stackTrace := make([]byte, 1024)
			s := runtime.Stack(stackTrace, false)
			stackStr := fmt.Sprint(string(stackTrace[:s]))
			err = E.New("Error in Parsing", r, "Stack trace:", stackStr)

		}
	}()

	configs, _ = decodeBase64IfNeeded(configs)

	convertedData, err := GenerateConfigLite(configs, useXrayWhenPossible)
	return convertedData, err
}
