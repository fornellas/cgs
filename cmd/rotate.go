package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fornellas/cgs/gcode"
)

var RotateCmd = &cobra.Command{
	Use:   "rotate [path]",
	Short: "Read g-code from given path and rotate X/Y coordinates.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		path := args[0]
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, f.Close()) }()

		w, err := outputValue.WriterCloser()
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, w.Close()) }()

		parser := gcode.NewParser(f)
		for {
			block, err := parser.Next()
			if err != nil {
				return err
			}
			if block == nil {
				return nil
			}
			fmt.Printf("<= %s\n", block.String())
			// FIXME G20 / G21 may change units half-way
			if err := block.RotateXY(rotateX, rotateY, rotateDegrees); err != nil {
				return err
			}
			fmt.Printf("=> %s\n", block.String())
			str := block.String()
			n, err := fmt.Fprintln(w, str)
			if err != nil {
				return err
			}
			if n != len(str)+1 {
				return fmt.Errorf("short write")
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
