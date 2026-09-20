package hcore

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// A high-entropy native bootstrap secret gives both mobile core processes the
// same pinned identity, without writing a TLS private key to the profile DB.
func rpcCertificate(secret string) ([]byte, tls.Certificate, error) {
	if len(secret) < 32 {
		return nil, tls.Certificate{}, fmt.Errorf("RPC bootstrap secret is required")
	}
	seed := sha256.Sum256([]byte("sdm/local-rpc/tls/v1\x00" + secret))
	key, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), seed[:])
	for err != nil {
		seed = sha256.Sum256(seed[:])
		key, err = ecdsa.ParseRawPrivateKey(elliptic.P256(), seed[:])
	}
	template := &x509.Certificate{
		SerialNumber: new(big.Int).SetBytes(seed[:16]), Subject: pkix.Name{CommonName: "localhost"},
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		NotBefore: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), NotAfter: time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), deterministicCertificateSigner{key})
	if err != nil {
		return nil, tls.Certificate{}, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

// Go's standard RFC 6979 implementation keeps the pinned certificate identical
// in the app and extension without persisting or transmitting its private key.
// TLS handshake signatures still use the ordinary ecdsa.PrivateKey signer.
type deterministicCertificateSigner struct{ *ecdsa.PrivateKey }

func (s deterministicCertificateSigner) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.PrivateKey.Sign(nil, digest, opts)
}

func secureRPCOptions(secret string) ([]grpc.ServerOption, []byte, error) {
	certPEM, cert, err := rpcCertificate(secret)
	if err != nil {
		return nil, nil, err
	}
	authorized := func(ctx context.Context) error {
		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get("authorization")
		if len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), []byte("Bearer "+secret)) != 1 {
			return status.Error(codes.Unauthenticated, "local RPC authentication required")
		}
		return nil
	}
	return []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}})),
		grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			if err := authorized(ctx); err != nil {
				return nil, err
			}
			return handler(ctx, req)
		}),
		grpc.StreamInterceptor(func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			if err := authorized(stream.Context()); err != nil {
				return err
			}
			return handler(srv, stream)
		}),
	}, certPEM, nil
}

func validateRPCListen(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid RPC listen address")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("RPC must listen on a numeric loopback address")
	}
	return nil
}
