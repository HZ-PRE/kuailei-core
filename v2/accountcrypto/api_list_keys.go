package accountcrypto

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"

	"github.com/HZ-PRE/kuailei-core/v2/hcommon"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const apiListKeyID = "prod-2026-01"
const apiListPublicKeyBase64 = "MCowBQYDK2VwAyEAXlfVeWzoh1xPeqEkO7iTaZEdYro0M4lKoCbfgTI/yLg="

func (s *Service) GetApiListKeys(ctx context.Context, request *hcommon.Empty) (*hcommon.Response, error) {
	// The publisher uses the account API's AES-128 key for list encryption.
	// Read it through the same validation path; do not maintain a second copy.
	accountKey, err := s.GetAppAesKey(ctx, request)
	if err != nil {
		return nil, err
	}
	der, err := base64.StdEncoding.DecodeString(apiListPublicKeyBase64)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "API list verification key is invalid")
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	publicKey, ok := parsed.(ed25519.PublicKey)
	if err != nil || !ok {
		return nil, status.Error(codes.FailedPrecondition, "API list verification key is invalid")
	}
	data, err := json.Marshal(map[string]string{
		"keyId":     apiListKeyID,
		"aesKey":    accountKey.Message,
		"publicKey": base64.StdEncoding.EncodeToString(publicKey),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "API list key configuration unavailable")
	}
	return &hcommon.Response{Code: hcommon.ResponseCode_OK, Message: string(data)}, nil
}
