package main

import (
	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	controlMod "github.com/fornellas/cgs/control"
	grblMod "github.com/fornellas/cgs/grbl"
)

var displayStatusComms bool
var defaultDisplayStatusComms = false

var ControlCmd = &cobra.Command{
	Use:   "control",
	Short: "Open Grbl serial connection and provide a terminal control interface.",
	Args:  cobra.NoArgs,
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		ctx, _ := log.MustWithAttrs(
			cmd.Context(),
			"port-name", portName,
			"address", address,
			"timeout", timeout,
			"display-status-comms", displayStatusComms,
		)
		cmd.SetContext(ctx)

		openPortFn, err := GetOpenPortFn()
		if err != nil {
			return err
		}

		grbl := grblMod.NewGrbl(openPortFn)

		control := controlMod.NewControl(grbl, &controlMod.ControlOptions{
			DisplayStatusComms: displayStatusComms,
			AppLogger:          logDebugFileLogger,
		})

		return control.Run(ctx)
	}),
}

func init() {
	AddPortFlags(ControlCmd)

	ControlCmd.Flags().BoolVar(
		&displayStatusComms,
		"display-status-comms",
		defaultDisplayStatusComms,
		"Various status commands ($#, $$, $N, $I, $G, ?) are polled automatically; this option enables showing such communication (very noisy)",
	)

	RootCmd.AddCommand(ControlCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		displayStatusComms = defaultDisplayStatusComms
	})
}
