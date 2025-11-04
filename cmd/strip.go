package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/fornellas/cgs/gcode"
)

var stripPath string
var defaultStripPath = ""

var StripCmd = &cobra.Command{
	Use:   "strip path",
	Short: "Strip given g-code file of comments and spaces.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		path := args[0]
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, f.Close()) }()

		var w io.Writer
		if len(stripPath) > 0 {
			var wf *os.File
			wf, err = os.OpenFile(path, os.O_CREATE, os.FileMode(0644))
			if err != nil {
				return err
			}
			w = wf
			defer func() { err = errors.Join(err, wf.Close()) }()
		} else {
			w = os.Stdout
		}

		parser := gcode.NewParser(f)
		for {
			token, err := parser.Next()
			if err != nil {
				return err
			}
			if token == nil {
				return nil
			}
			fmt.Fprintln(w, token.String())
		}
	},
}

func init() {
	StripCmd.PersistentFlags().StringVarP(
		&stripPath, "output", "o", defaultStripPath,
		"Write stripped g-code to this file instead of stdout",
	)

	RootCmd.AddCommand(StripCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
		stripPath = defaultStripPath
	})
}
