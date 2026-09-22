package hcore

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HZ-PRE/kuailei-core/v2/accountcrypto"
	"github.com/HZ-PRE/kuailei-core/v2/hcommon"
	"github.com/HZ-PRE/kuailei-core/v2/hello"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRPCServerBootstrapLifecycle(t *testing.T) {
	// Exercise the same legacy mode number used by desktop FFI. Despite its
	// name it must now supply a certificate and require authenticated TLS.
	mode := SetupMode_GRPC_NORMAL_INSECURE
	secret := strings.Repeat("test-only-bootstrap", 4)
	if cert := GetGrpcServerPublicKey(); len(cert) != 0 {
		t.Fatal("unexpected certificate before bootstrap")
	}
	if _, err := StartGrpcServerByMode("127.0.0.1:0", mode, ""); err == nil {
		t.Fatal("empty bootstrap accepted")
	}
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	address := occupied.Addr().String()
	if _, err = StartGrpcServerByMode(address, mode, secret); err == nil {
		t.Fatal("occupied listener accepted")
	}
	if len(GetGrpcServerPublicKey()) != 0 || grpcServerRunning(mode) {
		t.Fatal("failed startup published a partial bootstrap")
	}
	occupied.Close()
	server, err := StartGrpcServerByMode(address, mode, secret)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		CloseGrpcServer(mode)
		CloseGrpcServer(SetupMode_GRPC_BACKGROUND_INSECURE)
		rpcMu.Lock()
		rpcPublicCertificate, rpcBootstrapSecret = nil, ""
		rpcMu.Unlock()
	})
	cert := GetGrpcServerPublicKey()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(cert) {
		t.Fatal("bootstrap did not publish a certificate")
	}
	cert[0] ^= 1
	if bytes.Equal(cert, GetGrpcServerPublicKey()) {
		t.Fatal("getter exposed mutable certificate storage")
	}
	repeated, err := StartGrpcServerByMode(address, mode, secret)
	if err != nil || repeated != server {
		t.Fatalf("repeat bootstrap: %v", err)
	}
	if _, err = StartGrpcServerByMode(address, mode, secret+"changed"); err == nil {
		t.Fatal("accepted a changed bootstrap secret")
	}
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS13})))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := accountcrypto.NewAccountCryptoClient(conn).GetAppAesKey(ctx, &hcommon.Empty{}); status.Code(err) != codes.Unauthenticated {
		t.Fatal("account key RPC accepted unauthenticated access")
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+secret)
	if _, err := accountcrypto.NewAccountCryptoClient(conn).GetAppAesKey(ctx, &hcommon.Empty{}); err != nil && status.Code(err) != codes.FailedPrecondition {
		t.Fatal("account key RPC was not registered on the authenticated core server")
	}
	if _, err = hello.NewHelloClient(conn).SayHello(ctx, &hello.HelloRequest{Name: "bootstrap-regression"}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				GetGrpcServerPublicKey()
			}
		}()
	}
	for range 5 {
		if _, err := StartGrpcServerByMode("127.0.0.1:0", SetupMode_GRPC_BACKGROUND_INSECURE, secret); err != nil {
			t.Error(err)
		}
		CloseGrpcServer(SetupMode_GRPC_BACKGROUND_INSECURE)
	}
	wg.Wait()
	if !pool.AppendCertsFromPEM(GetGrpcServerPublicKey()) {
		t.Fatal("background restart lost identity")
	}
}
