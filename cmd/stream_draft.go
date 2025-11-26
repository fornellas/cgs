package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	grblMod "github.com/fornellas/cgs/grbl"
)

var StreamCmd = &cobra.Command{
	Use:   "stream [path]",
	Short: "Stream program at given file via Grbl serial connection.",
	Args:  cobra.ExactArgs(1),
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		path := args[0]

		ctx, logger := log.MustWithAttrs(
			cmd.Context(),
			"port-name", portName,
			"address", address,
			"timeout", timeout,
			"file", path,
		)
		cmd.SetContext(ctx)

		openPortFn, err := GetOpenPortFn()
		if err != nil {
			return err
		}

		grbl := grblMod.NewGrbl(openPortFn)

		logger.Info("Connecting")
		pushMessageCh, err := grbl.Connect(ctx)
		if err != nil {
			return fmt.Errorf("failed to connect to Grbl: %w", err)
		}
		defer func() { err = errors.Join(err, grbl.Disconnect(ctx)) }()
		// FIXME
		grbl.SendCommand(ctx, "$X")
		grbl.SendCommand(ctx, "$C")

		go func() {
			for message := range pushMessageCh {
				logger.Warn("Grbl push message", "message", message)
			}
		}()

		logger.Info("Opening path")
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer func() { err = errors.Join(err, file.Close()) }()

		logger.Info("Streaming")
		streamErr := grbl.StreamProgram(ctx, file)
		logger.Info("Stream finished", "streamErr", streamErr)
		return streamErr
	}),
}

func init() {
	AddPortFlags(StreamCmd)

	RootCmd.AddCommand(StreamCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
	})
}
