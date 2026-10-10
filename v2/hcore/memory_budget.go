package hcore

import (
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/common/memory"
)

// Reserve headroom for Flutter, sockets and native allocations. libbox uses
// three quarters of this budget as its Go soft limit; this is not an RSS cap.
func coreMemoryBudget() int64 {
	if C.IsIos {
		return 0 // Keep the NetworkExtension-specific policy in libbox.
	}
	return memoryBudget(memory.Total(), C.IsAndroid)
}

func memoryBudget(total uint64, mobile bool) int64 {
	const mib = 1024 * 1024
	low, high := uint64(256*mib), uint64(1024*mib)
	if mobile {
		low, high = 128*mib, 512*mib
	}
	return int64(min(high, max(low, total/16)))
}
