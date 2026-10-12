package tunnelservice

import "testing"

func TestLegacyTunnelCannotExposeControlPort(t *testing.T) {
	service := new(sdmNext)
	for _, addr := range []string{"127.0.0.1:18020", "0.0.0.0:18020", "[::]:18020"} {
		server, err := service.StartTunnelGrpcServer(addr)
		if err == nil || server != nil {
			t.Fatal("legacy RPC accepted", addr)
		}
	}
	if err := service.Stop(nil); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"run", "start", "install"} {
		if code, _ := StartTunnelService(command); code == 0 {
			t.Fatal("legacy service enabled", command)
		}
	}
}
