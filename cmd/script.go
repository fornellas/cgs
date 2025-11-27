package main

import (
	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"
	"github.com/traefik/yaegi/interp"
)

var ScriptCmd = &cobra.Command{
	Use:   "script path",
	Short: "Execute a script.",
	Args:  cobra.ExactArgs(1),
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		path := args[0]

		ctx, logger := log.MustWithAttrs(
			cmd.Context(),
			"path", path,
		)
		cmd.SetContext(ctx)

		interpreter := interp.New(interp.Options{})

		logger.Info("Running")
		if _, err := interpreter.EvalPathWithContext(ctx, path); err != nil {
			return err
		}

		return nil
	}),
}

func init() {
	RootCmd.AddCommand(ScriptCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
	})
}
