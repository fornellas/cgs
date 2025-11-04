package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fornellas/cgs/gcode"
)

var CompactCmd = &cobra.Command{
	Use:   "compact [path]",
	Short: "Read g-code from given path and compact it by stripping spaces and comments.",
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
	AddOutputFlags(CompactCmd)
	RootCmd.AddCommand(CompactCmd)
}
