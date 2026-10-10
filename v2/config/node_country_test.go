package config

import (
	"context"
	"github.com/sagernet/sing-box/include"
	"testing"
)

func TestNodeCountrySurvivesParseAndBuild(t *testing.T) {
	ctx := include.Context(context.Background())
	content := `{"outbounds":[
 {"type":"vless","tag":"vless-node","country_code":" sg ","server":"127.0.0.1","server_port":443,"uuid":"00000000-0000-4000-8000-000000000001"},
 {"type":"ssplus","tag":"ss-node","country_code":"JP","server":"127.0.0.1","server_port":443,"method":"aes-256-gcm","user_id":1,"password":"test","iv_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","protocol":"auth_aes128_md5"},
 {"type":"vless","tag":"no-country","country_code":"invalid","server":"127.0.0.1","server_port":443,"uuid":"00000000-0000-4000-8000-000000000001"},
 {"type":"balancer","tag":"香港","country_code":" hk ","strategy":"lowest-delay","outbounds":["ss-node","vless-node"]}]}`
	for _, enabled := range []bool{false, true} {
		opts := DefaultSdmOptions()
		opts.EnableNodeIPInfo = enabled
		normalized, err := ParseConfigBytes(ctx, &ReadOptions{Content: content}, true, opts, false)
		if err != nil {
			t.Fatal(err)
		}
		built, err := BuildConfig(ctx, opts, &ReadOptions{Content: string(normalized)})
		if err != nil {
			t.Fatal(err)
		}
		countries := map[string]string{}
		for _, node := range built.Outbounds {
			countries[node.Tag] = node.CountryCode
		}
		if countries["vless-node"] != "SG" || countries["ss-node"] != "JP" || countries["no-country"] != "" || countries["香港"] != "HK" {
			t.Fatalf("country lost: %v", countries)
		}
		if built.Experimental.Monitoring.EnableNodeIPInfo != enabled {
			t.Fatal("IP permission lost")
		}
	}
}
