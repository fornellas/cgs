package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	slogxtCobra "github.com/fornellas/slogxt/cobra"
	"github.com/fornellas/slogxt/log"
)

var logDebugPath string
var logDebugFile io.WriteCloser
var logDebugFileLogger *slog.Logger
var defaultLogDebugPath = ""

func getCmdChainStr(cmd *cobra.Command) string {
	cmdChain := []string{cmd.Name()}
	for {
		parentCmd := cmd.Parent()
		if parentCmd == nil {
			break
		}
		cmdChain = append([]string{parentCmd.Name()}, cmdChain...)
		cmd = parentCmd
	}
	return "⚙️ " + strings.Join(cmdChain, " ")
}

var RootCmd = &cobra.Command{
	Use:   "cgs",
	Short: "CLI G-Code Sender",
	Args:  cobra.NoArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Environment Flags
		// Inspired by https://github.com/spf13/viper/issues/671#issuecomment-671067523
		v := viper.New()
		v.SetEnvPrefix("CGS")
		v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		v.AutomaticEnv()
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed && v.IsSet(f.Name) {
				cmd.Flags().Set(f.Name, fmt.Sprintf("%v", v.Get(f.Name)))
			}
		})

		// Logging
		logger := slogxtCobra.GetLogger(cmd.OutOrStderr()).
			WithGroup(getCmdChainStr(cmd))
		ctx := log.WithLogger(cmd.Context(), logger)
		cmd.SetContext(ctx)

		if logDebugPath != "" {
			var err error
			logDebugFile, err = os.OpenFile(logDebugPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			debugFileHandler := log.NewTerminalLineHandler(logDebugFile, &log.TerminalHandlerOptions{
				HandlerOptions: slog.HandlerOptions{
					Level: slog.LevelDebug,
				},
				ForceColor: true,
			}).WithGroup(getCmdChainStr(cmd))
			logDebugFileLogger = slog.New(debugFileHandler)

			logger := slog.New(log.NewMultiHandler(debugFileHandler, logger.Handler()))
			ctx = log.WithLogger(cmd.Context(), logger)
			cmd.SetContext(ctx)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if logDebugFile != nil {
			return logDebugFile.Close()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			logger := log.MustLogger(cmd.Context())
			logger.Error("Failed to display help", "err", err)
		}
		Exit(1)
	},
}

var resetFlagsFns = []func(){
	func() { slogxtCobra.Reset() },
}

func ResetFlags() {
	for _, resetFlagFn := range resetFlagsFns {
		resetFlagFn()
	}
}

func init() {
	slogxtCobra.AddLoggerFlags(RootCmd)

	TuiCmd.PersistentFlags().StringVarP(
		&logDebugPath, "log-debug-path", "", defaultLogDebugPath,
		"Truncate file and write debugging logging to it.",
	)

	resetFlagsFns = append(resetFlagsFns, func() {
		logDebugPath = defaultLogDebugPath
	})
}
