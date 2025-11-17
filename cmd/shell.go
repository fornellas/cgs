package main

import (
	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	grblMod "github.com/fornellas/cgs/grbl"
	shellMod "github.com/fornellas/cgs/shell"
)

var displayStatusComms bool
var defaultDisplayStatusComms = false

var displayGcodeParserStateComms bool
var defaultDisplayGcodeParserStateComms = false

var ShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open Grbl serial connection and provide a shell prompt to send commands.",
	Args:  cobra.NoArgs,
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		ctx, _ := log.MustWithAttrs(
			cmd.Context(),
			"port-name", portName,
			"address", address,
		)
		cmd.SetContext(ctx)

		openPortFn, err := GetOpenPortFn()
		if err != nil {
			return err
		}

		grbl := grblMod.NewGrbl(openPortFn)

		shell := shellMod.NewShell(grbl, displayStatusComms, displayGcodeParserStateComms)

		return shell.Run(ctx)
	}),
}

func init() {
	AddPortFlags(ShellCmd)

	ShellCmd.Flags().BoolVar(
		&displayStatusComms,
		"display-status-comms",
		defaultDisplayStatusComms,
		"Display status report query real-time commands and status report push messages; this is always automatically polled and can be noisy",
	)

	ShellCmd.Flags().BoolVar(
		&displayGcodeParserStateComms,
		"display-gcode-parser-state-comms",
		defaultDisplayGcodeParserStateComms,
		"Display G-Code Parser State real-time commands and report push messages; this is always automatically polled and can be noisy",
	)

	RootCmd.AddCommand(ShellCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		displayStatusComms = defaultDisplayStatusComms
	})
}
