package main

import (
	"errors"
	"fmt"
	"io"
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
		defer func() {
			logger.Error("defer r")
			err = errors.Join(err, f.Close())
		}()

		var w io.WriteCloser
		w, err = outputValue.WriterCloser()
		if err != nil {
			return err
		}
		defer func() {
			logger.Error("defer w")
			err = errors.Join(err, w.Close())
		}()

		parser := gcode.NewParser(f)
		initialModalGroup := parser.ModalGroup.Copy()
		for {
			var block *gcode.Block
			block, err = parser.Next()
			if err != nil {
				return err
			}
			if block == nil {
				logger.Info("Completed")
				return
			}
			if !parser.ModalGroup.Units.Equal(initialModalGroup.Units) {
				return fmt.Errorf("line %d: unit change not supported", parser.Lexer.Line)
			}

			oldBlockStr := block.String()
			// FIXME if in relative mode, if X or Y is missing, it must be set to the current value
			if err = block.RotateXY(rotateX, rotateY, rotateDegrees); err != nil {
				return err
			}
			fmt.Printf("%s => %s\n", oldBlockStr, block.String())
			str := block.String()
			var n int
			n, err = fmt.Fprintln(w, str)
			if err != nil {
				return err
			}
			if n != len(str)+1 {
				return fmt.Errorf("short write")
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
