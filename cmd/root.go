package cmd

import (
	"fmt"
	"github.com/hadi77ir/go-logging"
	"github.com/hadi77ir/wsproxy/pkg/proxy"
	"github.com/hadi77ir/wsproxy/pkg/utils"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	// socks5 handler
	_ "github.com/hadi77ir/wsproxy/pkg/socks5"
)

const HelpFooter = "\n" +
	"Submit bug reports, questions, discussions through our repository's\n" +
	"issue tracker at https://github.com/hadi77ir/wsproxy/issues\n"
const GoVersionFormat = "Built with Go toolchain %s"

var RootCmd = &cobra.Command{
	Use:   "wsproxy [flags] LOCAL REMOTE",
	Short: "wsproxy - yet another websockify implementation, with additional functionality",
	Long: `wsproxy is a utility suited to forward (sometimes called "tunneling")
connections of one transport over another. for example: WebSocket.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger := cmd.Context().Value("logger").(logging.Logger)
		utils.SetMaxProcs(logger)
	},
	Args:       cobra.MinimumNArgs(2),
	ArgAliases: []string{"local", "remote"},
	Run: func(cmd *cobra.Command, args []string) {
		logger := cmd.Context().Value("logger").(logging.Logger)

		// Handle SIGINT and SIGTERM.
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		if len(args) != 2 {
			logger.Log(logging.ErrorLevel, "incorrect number of args: =", len(args), ", has to be = 2")
			return
		}

		local := args[0]
		remote := args[1]

		localOptions, err := cmd.Flags().GetStringArray("lo")
		if err != nil {
			logger.Log(logging.ErrorLevel, "error reading local options:", err)
			return
		}
		remoteOptions, err := cmd.Flags().GetStringArray("ro")
		if err != nil {
			logger.Log(logging.ErrorLevel, "error reading remote options:", err)
			return
		}

		parsedLocalOptions, err := utils.ParseTransportParamsFromFlags(localOptions)
		if err != nil {
			logger.Log(logging.ErrorLevel, "error reading local transport parameters:", err)
			return
		}
		parsedRemoteOptions, err := utils.ParseTransportParamsFromFlags(remoteOptions)
		if err != nil {
			logger.Log(logging.ErrorLevel, "error parsing remote transport parameters:", err)
			return
		}

		localEndpoint := proxy.Endpoint{Addr: local, TransportParams: parsedLocalOptions}
		remoteEndpoint := proxy.Endpoint{Addr: remote, TransportParams: parsedRemoteOptions}
		instance := proxy.NewProxy(localEndpoint, remoteEndpoint, logger, sigChan)

		if err := instance.Run(); err != nil {
			logger.Log(logging.ErrorLevel, err)
		}

		logger.Log(logging.InfoLevel, "Goodbye!")
	},
}

func getVersion() string {
	version := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}
	return version
}

func init() {
	RootCmd.Version = getVersion()
	toolchainVersion := fmt.Sprintf(GoVersionFormat, runtime.Version()) + "\n"
	RootCmd.SetVersionTemplate(RootCmd.Name() + " " + getVersion() + "\n")
	RootCmd.SetHelpTemplate(RootCmd.VersionTemplate() + "\n" + RootCmd.HelpTemplate() + HelpFooter + "\n" + toolchainVersion)
	RootCmd.SetVersionTemplate(RootCmd.VersionTemplate() + toolchainVersion)

	// Flags
	RootCmd.Flags().StringArrayP("lo", "l", nil, "transport parameters for local endpoint, one at a time")
	RootCmd.Flags().StringArrayP("ro", "r", nil, "transport parameters for remote endpoint, one at a time")
}
