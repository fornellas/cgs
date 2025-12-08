package main

import (
	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	grblMod "github.com/fornellas/cgs/grbl"
	tuiMod "github.com/fornellas/cgs/tui"
)

var displayStatusComms bool
var defaultDisplayStatusComms = false

var TuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open Grbl serial connection and provide a terminal user interface.",
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

		tui := tuiMod.NewTui(grbl, &tuiMod.TuiOptions{
			DisplayStatusComms: displayStatusComms,
			AppLogger:          logDebugFileLogger,
		})

		return tui.Run(ctx)
	}),
}

func init() {
	AddPortFlags(TuiCmd)

	TuiCmd.Flags().BoolVar(
		&displayStatusComms,
		"display-status-comms",
		defaultDisplayStatusComms,
		"Various status commands ($#, $$, $N, $I, $G, ?) are polled automatically; this option enables showing such communication (very noisy)",
	)

	RootCmd.AddCommand(TuiCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		displayStatusComms = defaultDisplayStatusComms
	})
}
