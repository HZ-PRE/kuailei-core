package accountcrypto

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/HZ-PRE/kuailei-core/v2/hcommon"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetAppAesKey(t *testing.T) {
	previous := appAESKeyBase64
	t.Cleanup(func() { appAESKeyBase64 = previous })
	svc := &Service{}
	for _, invalid := range []string{"", "not-base64", base64.StdEncoding.EncodeToString(make([]byte, 24)), base64.StdEncoding.EncodeToString(make([]byte, 32))} {
		appAESKeyBase64 = invalid
		if _, err := svc.GetAppAesKey(context.Background(), &hcommon.Empty{}); status.Code(err) != codes.FailedPrecondition {
			t.Fatal("invalid key configuration accepted")
		}
	}
	appAESKeyBase64 = base64.StdEncoding.EncodeToString(make([]byte, 16))
	response, err := svc.GetAppAesKey(context.Background(), &hcommon.Empty{})
	if err != nil || response.Code != hcommon.ResponseCode_OK || response.Message != appAESKeyBase64 {
		t.Fatal("valid key configuration was not returned")
	}
}
