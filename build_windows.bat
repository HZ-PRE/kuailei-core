@echo off
set GOOS=windows
set GOARCH=amd64
set CC=x86_64-w64-mingw32-gcc
set CGO_ENABLED=1
go run ./cli tunnel exit
del bin\sdm-core.dll bin\SdmCli.exe
set CGO_LDFLAGS=
go build -trimpath -tags with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_grpc -ldflags="-w -s" -buildmode=c-shared -o bin/sdm-core.dll ./platform/desktop
go get github.com/akavel/rsrc
go install github.com/akavel/rsrc

rsrc  -ico .\assets\sdm-cli.ico -o cli\bydll\cli.syso

copy bin\sdm-core.dll .
set CGO_LDFLAGS="sdm-core.dll"
go build  -o bin/SdmCli.exe ./cli/bydll/
del sdm-core.dll
echo Done.
exit /b 0