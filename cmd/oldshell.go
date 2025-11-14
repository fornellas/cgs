package main

import (
	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	grblMod "github.com/fornellas/cgs/grbl"
	"github.com/fornellas/cgs/oldshell"
)

var enableStatusMessages bool
var defaultEnableStatusMessages = false

var OldShellCmd = &cobra.Command{
	Use:   "oldshell",
	Short: "Open Grbl serial connection and provide a shell prompt to send commands.",
	Args:  cobra.NoArgs,
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		ctx, logger := log.MustWithAttrs(
			cmd.Context(),
			"port-name", portName,
			"address", address,
		)
		cmd.SetContext(ctx)

		logger.Info("Running")

		openPortFn, err := GetOpenPortFn()
		if err != nil {
			return err
		}

		grbl := grblMod.NewGrbl(openPortFn)

		shell := oldshell.NewShell(grbl, enableStatusMessages)
		if err := shell.Execute(ctx); err != nil {
			return err
		}

		return nil
	}),
}

func init() {
	AddPortFlags(OldShellCmd)

	OldShellCmd.PersistentFlags().BoolVar(&enableStatusMessages, "enable-status-messages", defaultEnableStatusMessages, "Status report is polled regularly, this enables visualization of the query and response message.")

	RootCmd.AddCommand(OldShellCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		enableStatusMessages = defaultEnableStatusMessages
	})
}
