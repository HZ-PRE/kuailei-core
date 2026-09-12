//go:build with_wireguard

package test

import (
	"os"
	"testing"

	"github.com/HZ-PRE/kuailei-core/v2/profile"
	"github.com/sagernet/sing-box/experimental/libbox"
)

func TestAddByContent(t *testing.T) {
	ctx := libbox.BaseContext(nil)
	content, err := os.ReadFile("testdata/warp")
	if err != nil {
		t.Fatalf("read test profile: %v", err)
	}
	entity, err := profile.AddByContent(ctx, string(content), "", false)
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}
	if entity == nil {
		t.Fatal("expected a profile entity")
	}
	t.Cleanup(func() {
		if err := profile.DeleteById(entity.Id); err != nil {
			t.Errorf("delete test profile: %v", err)
		}
	})
	if entity.Id == "" {
		t.Error("expected a generated profile ID")
	}
	if entity.LastUpdate <= 0 {
		t.Errorf("expected a valid update timestamp, got %d", entity.LastUpdate)
	}
	if entity.Name != "" {
		t.Errorf("expected the supplied empty profile name, got %q", entity.Name)
	}
}
