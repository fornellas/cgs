package main

import (
	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	controlMod "github.com/fornellas/cgs/control"
	grblMod "github.com/fornellas/cgs/grbl"
)

var displayStatusComms bool
var defaultDisplayStatusComms = false

var displayGcodeParserStateComms bool
var defaultDisplayGcodeParserStateComms = false

var displayGcodeParamStateComms bool
var defaultDisplayGcodeParamStateComms = false

var ControlCmd = &cobra.Command{
	Use:   "control",
	Short: "Open Grbl serial connection and provide a terminal control interface.",
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

		control := controlMod.NewControl(grbl, &controlMod.ControlOptions{
			DisplayStatusComms:           displayStatusComms,
			DisplayGcodeParserStateComms: displayGcodeParserStateComms,
			DisplayGcodeParamStateComms:  displayGcodeParamStateComms,
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
		"Display status report query real-time commands and status report push messages; this is always automatically polled and can be noisy",
	)

	ControlCmd.Flags().BoolVar(
		&displayGcodeParserStateComms,
		"display-gcode-parser-state-comms",
		defaultDisplayGcodeParserStateComms,
		"Display G-Code Parser State commands and report push messages; this is always automatically polled and can be noisy",
	)

	ControlCmd.Flags().BoolVar(
		&displayGcodeParamStateComms,
		"display-gcode-param-state-comms",
		defaultDisplayGcodeParamStateComms,
		"Display G-Code Param State commands and report push messages; this is always automatically polled and can be noisy",
	)

	RootCmd.AddCommand(ControlCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		displayStatusComms = defaultDisplayStatusComms
	})
}
