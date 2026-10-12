package cmd

import (
	"fmt"
	"github.com/HZ-PRE/kuailei-core/v2/hcore/tunnelservice"
	"github.com/spf13/cobra"
)

var commandService = &cobra.Command{
	Use:       "tunnel stop/uninstall",
	Short:     "Remove the retired tunnel service; use built-in TUN for new connections",
	ValidArgs: []string{"run", "start", "stop", "install", "uninstall", "activate", "deactivate", "exit"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]
		if arg != "stop" && arg != "uninstall" {
			return fmt.Errorf("legacy unauthenticated tunnel service is disabled; use the app's built-in TUN mode")
		}
		code, out := tunnelservice.StartTunnelService(arg)
		if code != 0 {
			return fmt.Errorf("%s", out)
		}
		cmd.Println(out)
		return nil
	},
}
