package ray2sing

import (
	"testing"

	"github.com/sagernet/sing-box/experimental/libbox"
)

func TestRay2SingboxInvalidInputReturnsError(t *testing.T) {
	for _, input := range []string{"not-a-proxy", "ss://%zz"} {
		t.Run(input, func(t *testing.T) {
			result, err := Ray2Singbox(libbox.BaseContext(nil), input, false)
			if err == nil || result != nil {
				t.Fatalf("expected conversion error and nil result, got %q, %v", result, err)
			}
		})
	}
}
