package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	"github.com/fornellas/cgs/gcode"
)

var RotateCmd = &cobra.Command{
	Use:   "rotate [path]",
	Short: "Read g-code from given path and rotate X/Y coordinates.",
	Args:  cobra.ExactArgs(1),
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		path := args[0]

		ctx, logger := log.MustWithAttrs(
			cmd.Context(),
			"path", path,
			"x-center", rotateX,
			"y-center", rotateY,
			"degrees", rotateDegrees,
			"output", outputValue,
		)
		cmd.SetContext(ctx)
		logger.Info("Running")

		var f *os.File
		f, err = os.Open(path)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, f.Close()) }()

		var w io.WriteCloser
		w, err = outputValue.WriterCloser()
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, w.Close()) }()

		parser := gcode.NewParser(f)
		radians := rotateDegrees * math.Pi / 180.0
		rotateXY := gcode.NewRotateXY(parser, rotateX, rotateY, radians)
		for {
			block, err := rotateXY.Next()
			if err != nil {
				return err
			}
			if block == nil {
				logger.Info("Complete")
				return nil
			}
			str := block.String()
			var n int
			n, err = fmt.Fprintln(w, str)
			if err != nil {
				return err
			}
			if n != len(str)+1 {
				return fmt.Errorf("%s: short write", outputValue)
			}
		}
	}),
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
