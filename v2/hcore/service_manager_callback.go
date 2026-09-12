package hcore

import (
	"github.com/HZ-PRE/kuailei-core/v2/service_manager"
	"github.com/sagernet/sing-box/adapter"
)

type sdmMainServiceManager struct{}

var _ adapter.LifecycleService = (*sdmMainServiceManager)(nil)

func (h *sdmMainServiceManager) Name() string { return "sdmMainServiceManager" }
func (h *sdmMainServiceManager) Start(stage adapter.StartStage) error {
	if stage == adapter.StartStateStarted {
		return service_manager.OnMainServiceStart()
	}
	return nil
}

func (h *sdmMainServiceManager) Close() error {
	return service_manager.OnMainServiceClose()
}
