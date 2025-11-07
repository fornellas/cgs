package main

import (
	"os"

	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"
)

// This is to be used in place of os.Exit() to aid writing test assertions on exit code.
var Exit func(int) = func(code int) { os.Exit(code) }

// GetRunFn returns a function suitable for usage with cobra.Command.Run. It runs fn, and if it
// errors, the error is logged then Exit(1) is called.
func GetRunFn(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if err := fn(cmd, args); err != nil {
			logger := log.MustLogger(cmd.Context())
			logger.Error(err.Error())
			Exit(1)
		}
	}
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
