package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	grblMod "github.com/fornellas/cgs/grbl"
)

var ShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open Grbl serial connection and provide a shell prompt to send commands.",
	Args:  cobra.NoArgs,
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		ctx, logger := log.MustWithAttrs(
			cmd.Context(),
			"port-name", portName,
			"address", address,
		)
		cmd.SetContext(ctx)

		logger.Info("Running")

		openPortFn, err := GetOpenPortFn()
		if err != nil {
			return err
		}

		grbl := grblMod.NewGrbl(openPortFn)

		pushMessageCh, err := grbl.Connect(ctx)
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Join(err, grbl.Disconnect(ctx))
		}()

		time.Sleep(time.Second)

		go func() {
			select {
			case <-ctx.Done():
				logger.Info("Receiver: context done", "err", ctx.Err())
				return
			case message, ok := <-pushMessageCh:
				if !ok {
					logger.Info("Receiver: push messages channel closed")
					return
				}
				fmt.Printf("< %s\n", message.String())
			}
		}()

		for {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("$ ")
			text, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			fmt.Printf("> %s", text)
			message, err := grbl.SendCommand(ctx, text)
			if err != nil {
				return fmt.Errorf("failed to send command: %w", err)
			}
			fmt.Printf("< %s\n", message.String())
		}
	}),
}

func init() {
	AddPortFlags(ShellCmd)

	RootCmd.AddCommand(ShellCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
	})
}
