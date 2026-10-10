package hcore

import (
	"github.com/sagernet/sing-box/daemon"
	"testing"
)

func TestMemoryBudget(t *testing.T) {
	const mib = 1024 * 1024
	for _, tc := range []struct {
		total  uint64
		mobile bool
		want   int64
	}{
		{0, true, 128 * mib}, {2048 * mib, true, 128 * mib}, {4096 * mib, true, 256 * mib}, {32768 * mib, true, 512 * mib},
		{0, false, 256 * mib}, {32768 * mib, false, 1024 * mib},
	} {
		if got := memoryBudget(tc.total, tc.mobile); got != tc.want {
			t.Fatalf("budget %+v: %d", tc, got)
		}
	}
}

func TestStatusUsesCurrentProfile(t *testing.T) {
	h := &SdmInstance{StartedService: new(daemon.StartedService)}
	name := "in-memory-current-profile"
	h.currentProfile.Store(&name)
	for i := 0; i < 3; i++ {
		if got := h.readStatus(nil); got.CurrentProfile != name {
			t.Fatalf("profile: %q", got.CurrentProfile)
		}
	}
	name = "switched-profile"
	h.currentProfile.Store(&name)
	if got := h.readStatus(&SystemInfo{CurrentProfile: "old-profile"}); got.CurrentProfile != name {
		t.Fatal("stale status profile")
	}
}
