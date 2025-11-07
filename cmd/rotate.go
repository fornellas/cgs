package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/cgs/gcode"
)

var RotateCmd = &cobra.Command{
	Use:   "rotate [path]",
	Short: "Read g-code from given path and rotate X/Y coordinates.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ctx := cmd.Context()
		ctx, logger := log.MustWithGroupAttrs(
			ctx, "rotate",
			"path", path,
			"x-center", rotateX,
			"y-center", rotateY,
			"degrees", rotateDegrees,
			"output", outputValue,
		)
		logger.Info("Running")

		f, err := os.Open(path)
		if err != nil {
			ExitError(ctx, err)
		}
		defer func() { err = errors.Join(err, f.Close()) }()

		w, err := outputValue.WriterCloser()
		if err != nil {
			ExitError(ctx, err)
		}
		defer func() { err = errors.Join(err, w.Close()) }()

		parser := gcode.NewParser(f)
		for {
			block, err := parser.Next()
			if err != nil {
				ExitError(ctx, err)
			}
			if block == nil {
				Exit(0)
			}
			fmt.Printf("<= %s\n", block.String())
			// FIXME G20 / G21 may change units half-way
			if err := block.RotateXY(rotateX, rotateY, rotateDegrees); err != nil {
				ExitError(ctx, err)
			}
			fmt.Printf("=> %s\n", block.String())
			str := block.String()
			n, err := fmt.Fprintln(w, str)
			if err != nil {
				ExitError(ctx, err)
			}
			if n != len(str)+1 {
				ExitError(ctx, fmt.Errorf("short write"))
			}
		}
	},
}

var rotateX float64
var defaultRotateX float64 = 0

var rotateY float64
var defaultRotateY float64 = 0

var rotateDegrees float64
var defaultRotateDegrees float64 = 0

func init() {
	RotateCmd.PersistentFlags().Float64VarP(&rotateX, "x-center", "x", defaultRotateX, "X coordinate for center of rotation")
	RotateCmd.PersistentFlags().Float64VarP(&rotateY, "y-center", "y", defaultRotateY, "Y coordinate for center of rotation")
	RotateCmd.PersistentFlags().Float64VarP(&rotateDegrees, "degrees", "d", defaultRotateDegrees, "Degrees to rotate counterclockwise")

	AddOutputFlags(RotateCmd)
	RootCmd.AddCommand(RotateCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		rotateX = defaultRotateX
		rotateY = defaultRotateY
		rotateDegrees = defaultRotateDegrees
	})
}
