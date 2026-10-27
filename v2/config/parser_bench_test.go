package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
)

func BenchmarkParseConfig(b *testing.B) {
	for _, count := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			var input strings.Builder
			input.WriteString(`{"outbounds":[`)
			for i := range count {
				if i != 0 {
					input.WriteByte(',')
				}
				fmt.Fprintf(&input, `{"type":"direct","tag":"node-%d"}`, i)
			}
			input.WriteString(`]}`)
			content := []byte(input.String())
			ctx := libbox.BaseContext(nil)
			b.ReportAllocs()
			b.SetBytes(int64(len(content)))
			b.ResetTimer()
			for b.Loop() {
				result, err := parseConfigContent(ctx, content, false, nil, false)
				if err != nil || len(result.Outbounds) != count {
					b.Fatalf("parse: %v", err)
				}
			}
		})
	}
}
