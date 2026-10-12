package tunnelservice

import "errors"

var errLegacyTunnelRPC = errors.New("legacy unauthenticated tunnel RPC is disabled; use the app's built-in TUN mode")

// Kept for API compatibility. Cleanup uses OS service control (stop/uninstall),
// never the old unauthenticated TCP endpoint.
func ActivateTunnelService(_ *TunnelStartRequest) error { return errLegacyTunnelRPC }
func DeactivateTunnelServiceForce() error               { return errLegacyTunnelRPC }
func DeactivateTunnelService() error                    { return errLegacyTunnelRPC }
func ExitTunnelService() (bool, error)                  { return false, errLegacyTunnelRPC }
