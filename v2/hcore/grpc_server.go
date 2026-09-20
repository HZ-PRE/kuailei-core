package hcore

/*
#include "stdint.h"
*/

import (
	"fmt"
	"io"

	"net"
	"os"
	sync "sync"
	"time"

	"github.com/HZ-PRE/kuailei-core/v2/config"
	"github.com/HZ-PRE/kuailei-core/v2/db"
	"github.com/HZ-PRE/kuailei-core/v2/ezytel"
	hcommon "github.com/HZ-PRE/kuailei-core/v2/hcommon"
	"github.com/HZ-PRE/kuailei-core/v2/hello"
	hutils "github.com/HZ-PRE/kuailei-core/v2/hutils"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	E "github.com/sagernet/sing/common/exceptions"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

type CoreService struct {
	UnimplementedCoreServer
}

func Setup(params *SetupRequest, platformInterface libbox.PlatformInterface) (setupErr error) {
	defer config.DeferPanicToError("setup", func(err error) {
		setupErr = err
		Log(LogLevel_FATAL, LogType_CORE, err.Error())
	})
	mu.Lock()
	defer mu.Unlock()
	if grpcServerRunning(params.Mode) {
		if _, err := StartGrpcServerByMode(params.Listen, params.Mode, params.Secret); err != nil {
			return err
		}
		Log(LogLevel_WARNING, LogType_CORE, "grpcServer already started")
		return nil
	}
	static.BaseContext = libbox.BaseContext(platformInterface)
	static.debug = params.Debug
	static.globalPlatformInterface = platformInterface
	tcpConn := true // runtime.GOOS == "windows" // TODO add TVOS
	libbox.Setup(
		&libbox.SetupOptions{
			BasePath:    params.BasePath,
			WorkingPath: params.WorkingDir,
			TempPath:    params.TempDir,
			// IsTVOS:          !tcpConn,
			FixAndroidStack: params.FixAndroidStack,
			LogMaxLines:     100,
			Debug:           params.Debug,
		})

	hutils.RedirectStderr(fmt.Sprint(params.WorkingDir, "/data/stderr", params.Mode, ".log"))

	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("libbox.Setup success %s %s %s %v", params.BasePath, params.WorkingDir, params.TempDir, tcpConn))

	sWorkingPath = params.WorkingDir
	os.Chdir(sWorkingPath)
	sTempPath = params.TempDir
	sUserID = os.Getuid()
	sGroupID = os.Getgid()

	var defaultWriter io.Writer
	if !params.Debug {
		defaultWriter = io.Discard
	}
	factory, err := log.New(
		log.Options{
			DefaultWriter: defaultWriter,
			BaseTime:      time.Now(),
			Observable:    true,
			// Options: option.LogOptions{
			// 	Disabled: false,
			// 	Level:    "trace",
			// 	Output:   "stdout",
			// },
		})
	static.CoreLogFactory = factory

	if err != nil {
		return E.Cause(err, "create logger")
	}

	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("StartGrpcServerByMode %s %d\n", params.Listen, params.Mode))
	switch params.Mode {
	case SetupMode_OLD:
		statusPropagationPort = int64(params.FlutterStatusPort)
	// case SetupMode_GRPC_BACKGROUND_INSECURE:
	default:
		_, err := StartGrpcServerByMode(params.Listen, params.Mode, params.Secret)
		if err != nil {
			return err
		}
	}
	settings := db.GetTable[hcommon.AppSettings]()
	val, err := settings.Get("SdmSettingsJson")
	Log(LogLevel_DEBUG, LogType_CORE, "SdmSettingsJson", val, err)
	if val == nil || err != nil {
		// if params.Mode == SetupMode_GRPC_BACKGROUND_INSECURE {
		_, err := ChangeSdmSettings(&ChangeSdmSettingsRequest{SdmSettingsJson: ""}, false)
		if err != nil {
			Log(LogLevel_ERROR, LogType_CORE, E.Cause(err, "ChangeSdmSettings").Error())
		}
	} else {
		// settings := db.GetTable[hcommon.AppSettings]()
		_, err := ChangeSdmSettings(&ChangeSdmSettingsRequest{SdmSettingsJson: val.Value.(string)}, false)
		if err != nil {
			Log(LogLevel_ERROR, LogType_CORE, E.Cause(err, "ChangeSdmSettings").Error())
		}

	}
	return InitSdmService()
}

func StartGrpcServer(listenAddressG string, service string) (*grpc.Server, error) {
	return nil, fmt.Errorf("unauthenticated legacy RPC is disabled; use native bootstrap")
}

func StartCoreGrpcServer(listenAddressG string) (*grpc.Server, error) {
	return StartGrpcServer(listenAddressG, "core")
}

func StartHelloGrpcServer(listenAddressG string) (*grpc.Server, error) {
	return StartGrpcServer(listenAddressG, "hello")
}

var (
	rpcPublicCertificate []byte
	rpcBootstrapSecret   string
	grpcServer           = make(map[SetupMode]*grpc.Server)
	mu                   sync.Mutex
	rpcMu                sync.Mutex
)

func grpcServerRunning(mode SetupMode) bool {
	rpcMu.Lock()
	defer rpcMu.Unlock()
	return grpcServer[mode] != nil
}

// StartGrpcServerByMode starts a gRPC server on the specified address with pinned TLS and per-call authentication.
func StartGrpcServerByMode(listenAddressG string, mode SetupMode, secret string) (*grpc.Server, error) {
	rpcMu.Lock()
	defer rpcMu.Unlock()
	if err := validateRPCListen(listenAddressG); err != nil {
		return nil, err
	}
	if rpcPublicCertificate != nil && secret != rpcBootstrapSecret {
		return nil, fmt.Errorf("RPC bootstrap identity cannot change within the process")
	}
	if server := grpcServer[mode]; server != nil {
		return server, nil
	}
	opts, cert, err := secureRPCOptions(secret)
	if err != nil {
		return nil, err
	}
	server := grpc.NewServer(opts...)
	// Register your gRPC service here
	RegisterCoreServer(server, &CoreService{})
	hello.RegisterHelloServer(server, &hello.HelloService{})
	ezytel.RegisterEzytelServer(server, ezytel.NewEzytelService(""))
	// Listen on the provided address
	lis, err := net.Listen("tcp", listenAddressG)
	if err != nil {
		Log(LogLevel_ERROR, LogType_CORE, fmt.Sprintf("failed to listen on %s: %v\n", listenAddressG, err))
		return nil, err
	}
	rpcPublicCertificate = cert
	rpcBootstrapSecret = secret
	grpcServer[mode] = server
	Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("grpcServer started on %s\n", listenAddressG))
	log.Info("Server listening on ", lis.Addr())

	// Run the server in a goroutine
	go func() {
		defer config.DeferPanicToError("grpcsetup", func(err error) {
			Log(LogLevel_FATAL, LogType_CORE, err.Error())
			<-time.After(5 * time.Second)
		})
		if err := server.Serve(lis); err != nil {
			Log(LogLevel_DEBUG, LogType_CORE, fmt.Sprintf("failed to serve: %v\n", err))
		}
		Log(LogLevel_DEBUG, LogType_CORE, "Server stopped")
	}()

	return grpcServer[mode], nil
}

// GetGrpcServerPublicKey returns the gRPC server's public key.
func GetGrpcServerPublicKey() []byte {
	rpcMu.Lock()
	defer rpcMu.Unlock()
	return append([]byte(nil), rpcPublicCertificate...)
}
func AddGrpcClientPublicKey(clientPublicKey []byte) error {
	return fmt.Errorf("use authenticated native RPC bootstrap")
}

func CloseGrpcServer(mode SetupMode) {
	rpcMu.Lock()
	defer rpcMu.Unlock()
	if server, ok := grpcServer[mode]; ok && server != nil {
		server.Stop()
		delete(grpcServer, mode)
	}
}
