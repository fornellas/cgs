package main

import (
	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	grblMod "github.com/fornellas/cgs/grbl"
)

var enableStatusMessages bool
var defaultEnableStatusMessages = false

var ShellCmd = &cobra.Command{
	Use:   "shell",
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

		shell := grblMod.NewShell(grbl, enableStatusMessages)
		if err := shell.Execute(ctx); err != nil {
			return err
		}

		return nil
	}),
}

func init() {
	AddPortFlags(ShellCmd)

	ShellCmd.PersistentFlags().BoolVar(&enableStatusMessages, "enable-status-messages", defaultEnableStatusMessages, "Status report is polled regularly, this enables visualization of the query and response message.")

	RootCmd.AddCommand(ShellCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		enableStatusMessages = defaultEnableStatusMessages
	})
}
