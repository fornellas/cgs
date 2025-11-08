package main

import (
	"errors"
	"fmt"

	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"
	"go.bug.st/serial"

	"github.com/fornellas/cgs/grbl"
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

		mode := &serial.Mode{
			BaudRate: 115200,
		}
		var port serial.Port
		port, err = serial.Open(portName, mode)
		if err != nil {
			return fmt.Errorf("%s: %w", portName, err)
		}
		defer func() { err = errors.Join(err, port.Close()) }()

		shell, err := grbl.NewShell(port)
		if err != nil {
			return err
		}
		defer shell.Close()

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
