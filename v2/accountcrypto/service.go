package accountcrypto

import (
	"context"
	"encoding/base64"

	"github.com/HZ-PRE/kuailei-core/v2/hcommon"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Set at core build time with -ldflags -X. Never log the value.
var appAESKeyBase64 string

type Service struct {
	UnimplementedAccountCryptoServer
}

func (s *Service) GetAppAesKey(context.Context, *hcommon.Empty) (*hcommon.Response, error) {
	key, err := base64.StdEncoding.DecodeString(appAESKeyBase64)
	if err != nil || len(key) != 16 {
		return nil, status.Error(codes.FailedPrecondition, "account AES-128-GCM requires a Base64-encoded 16-byte key")
	}
	return &hcommon.Response{Code: hcommon.ResponseCode_OK, Message: appAESKeyBase64}, nil
}
