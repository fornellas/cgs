package main

import (
	"errors"

	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	grblMod "github.com/fornellas/cgs/grbl"
)

var ShellCmd = &cobra.Command{
	Use:   "shell path",
	Short: "Open Grbl serial connection at path and provide a shell prompt to send commands.",
	Args:  cobra.ExactArgs(1),
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		portName := args[0]

		ctx, logger := log.MustWithAttrs(
			cmd.Context(),
			"portName", portName,
		)
		cmd.SetContext(ctx)
		logger.Info("Running")

		grbl := grblMod.NewGrbl(portName)

		shell, err := grblMod.NewShell(grbl)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, shell.Close()) }()
		if err := shell.Execute(); err != nil {
			return err
		}

		return nil
	}),
}

func init() {
	RootCmd.AddCommand(ShellCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
	})
}
