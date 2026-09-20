package mobile

import (
	"testing"

	"github.com/HZ-PRE/kuailei-core/v2/hcore"
	"google.golang.org/protobuf/proto"
)

func TestSystemInfoCanBeDecodedBeforeServiceStarts(t *testing.T) {
	encoded, err := GetSystemInfo()
	if err != nil {
		t.Fatal(err)
	}
	var snapshot hcore.SystemInfo
	if err := proto.Unmarshal(encoded, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Goroutines <= 0 || snapshot.UplinkTotal != 0 || snapshot.DownlinkTotal != 0 {
		t.Fatalf("unexpected stopped-service snapshot: %+v", &snapshot)
	}
}
