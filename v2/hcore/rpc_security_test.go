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

	"github.com/HZ-PRE/kuailei-core/v2/hello"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRPCBootstrapIdentityAndListenScope(t *testing.T) {
	secret := strings.Repeat("test-only-bootstrap", 4)
	a, _, err := rpcCertificate(secret)
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := rpcCertificate(secret)
	c, _, _ := rpcCertificate(secret + "other")
	if !bytes.Equal(a, b) || bytes.Equal(a, c) {
		t.Fatal("bootstrap identity mismatch")
	}
	if _, _, err := rpcCertificate(""); err == nil {
		t.Fatal("empty bootstrap accepted")
	}
	for _, address := range []string{"0.0.0.0:17078", "[::]:17078", "192.0.2.1:17078", "localhost:17078"} {
		if validateRPCListen(address) == nil {
			t.Fatal("non numeric loopback accepted")
		}
	}
	for _, address := range []string{"127.0.0.1:17078", "[::1]:17078"} {
		if err := validateRPCListen(address); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRPCRejectsCaptureAndUnauthorizedClients(t *testing.T) {
	secret := strings.Repeat("test-bootstrap", 4)
	opts, pem, err := secureRPCOptions(secret)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer(opts...)
	hello.RegisterHelloServer(server, &hello.HelloService{})
	captured := &wireCapture{Listener: listener}
	go server.Serve(captured)
	defer server.Stop()
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pem)
	trusted := credentials.NewTLS(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS13})
	connect := func(creds credentials.TransportCredentials) *grpc.ClientConn {
		conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(creds))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	client := hello.NewHelloClient(connect(trusted))
	for _, token := range []string{"", "wrong", secret} {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		}
		_, err := client.SayHello(ctx, &hello.HelloRequest{Name: "synthetic-node-credential"})
		if token == secret && err != nil {
			t.Fatal(err)
		}
		if token != secret && status.Code(err) != codes.Unauthenticated {
			t.Fatalf("unary auth failed: %v", err)
		}
		stream, err := client.SayHelloStream(ctx)
		if err == nil {
			_ = stream.Send(&hello.HelloRequest{Name: "synthetic-node-credential"})
			_, err = stream.Recv()
			_ = stream.CloseSend()
		}
		if token == secret && err != nil {
			t.Fatal(err)
		}
		if token != secret && status.Code(err) != codes.Unauthenticated {
			t.Fatalf("stream auth failed: %v", err)
		}
		cancel()
	}
	other, _, _ := rpcCertificate(secret + "other")
	wrongPool := x509.NewCertPool()
	wrongPool.AppendCertsFromPEM(other)
	for _, creds := range []credentials.TransportCredentials{
		insecure.NewCredentials(),
		credentials.NewTLS(&tls.Config{RootCAs: wrongPool}),
		credentials.NewTLS(&tls.Config{RootCAs: pool, MaxVersion: tls.VersionTLS12}),
	} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+secret)
		_, err := hello.NewHelloClient(connect(creds)).SayHello(ctx, &hello.HelloRequest{Name: "app"})
		cancel()
		if err == nil {
			t.Fatal("plaintext, wrong pin or obsolete TLS accepted")
		}
	}
	captured.mu.Lock()
	defer captured.mu.Unlock()
	if captured.data.Len() == 0 {
		t.Fatal("no wire data captured")
	}
	for _, sensitive := range []string{secret, "synthetic-node-credential"} {
		if bytes.Contains(captured.data.Bytes(), []byte(sensitive)) {
			t.Fatal("credential visible in captured RPC traffic")
		}
	}
}

// Observe real TCP bytes below TLS, including both directions.
type wireCapture struct {
	net.Listener
	mu   sync.Mutex
	data bytes.Buffer
}

func (w *wireCapture) Accept() (net.Conn, error) {
	c, err := w.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &capturedConn{Conn: c, owner: w}, nil
}

type capturedConn struct {
	net.Conn
	owner *wireCapture
}

func (c *capturedConn) record(p []byte) {
	c.owner.mu.Lock()
	defer c.owner.mu.Unlock()
	c.owner.data.Write(p)
}
func (c *capturedConn) Read(p []byte) (int, error) {
	n, e := c.Conn.Read(p)
	c.record(p[:n])
	return n, e
}
func (c *capturedConn) Write(p []byte) (int, error) {
	n, e := c.Conn.Write(p)
	c.record(p[:n])
	return n, e
}
