package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/cgs/gcode"
)

var CompactCmd = &cobra.Command{
	Use:   "compact [path]",
	Short: "Read g-code from given path and compact it by stripping spaces and comments.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		ctx := cmd.Context()
		ctx, logger := log.MustWithGroupAttrs(
			ctx, "compact",
			"path", path,
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

func init() {
	AddOutputFlags(CompactCmd)
	RootCmd.AddCommand(CompactCmd)
}
