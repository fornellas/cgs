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
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		path := args[0]

		ctx, logger := log.MustWithAttrs(
			cmd.Context(),
			"path", path,
			"output", outputValue,
		)
		cmd.SetContext(ctx)
		logger.Info("Running")

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
			eof, block, _, err := parser.Next()
			if err != nil {
				return err
			}
			if eof {
				return nil
			}
			if block == nil {
				continue
			}
			line := block.String()
			n, err := fmt.Fprintln(w, line)
			if err != nil {
				return err
			}
			if n != len(line)+1 {
				return fmt.Errorf("short write")
			}
		}
	}),
}

func init() {
	AddOutputFlags(CompactCmd)
	RootCmd.AddCommand(CompactCmd)
}
