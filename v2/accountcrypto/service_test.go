package accountcrypto

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

func TestGetApiListKeys(t *testing.T) {
	previous := appAESKeyBase64
	t.Cleanup(func() { appAESKeyBase64 = previous })
	svc := &Service{}
	for _, value := range []string{"", "bad", base64.StdEncoding.EncodeToString(make([]byte, 32))} {
		appAESKeyBase64 = value
		if _, err := svc.GetApiListKeys(context.Background(), &hcommon.Empty{}); status.Code(err) != codes.FailedPrecondition {
			t.Fatal("invalid list AES key accepted")
		}
	}
	appAESKeyBase64 = base64.StdEncoding.EncodeToString(make([]byte, 16))
	response, err := svc.GetApiListKeys(context.Background(), &hcommon.Empty{})
	if err != nil {
		t.Fatal("configured API list keys unavailable")
	}
	var keys map[string]string
	if json.Unmarshal([]byte(response.Message), &keys) != nil {
		t.Fatal("invalid key response")
	}
	public, err := base64.StdEncoding.DecodeString(keys["publicKey"])
	if err != nil || len(public) != 32 || keys["keyId"] != apiListKeyID || keys["aesKey"] != appAESKeyBase64 {
		t.Fatal("invalid list key contract")
	}
	// A key rotation must be reflected by both RPCs without separate provisioning.
	rotated := make([]byte, 16)
	rotated[0] = 42
	appAESKeyBase64 = base64.StdEncoding.EncodeToString(rotated)
	response, err = svc.GetApiListKeys(context.Background(), &hcommon.Empty{})
	if err != nil || json.Unmarshal([]byte(response.Message), &keys) != nil || keys["aesKey"] != appAESKeyBase64 {
		t.Fatal("list key did not follow account key rotation")
	}
}
