package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"

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

		openPortFn, err := GetOpenPortFn()
		if err != nil {
			return err
		}

		grbl := grblMod.NewGrbl(openPortFn)

		logger.Info("Connecting")
		pushMessageCh, err := grbl.Connect(ctx)
		if err != nil {
			return err
		}
		pushMessageDoneCh := make(chan struct{})
		defer func() {
			err = errors.Join(err, grbl.Disconnect(ctx))
			<-pushMessageDoneCh
		}()

		go func() {
			defer func() { pushMessageDoneCh <- struct{}{} }()
			for {
				select {
				case <-ctx.Done():
					return
				case message, ok := <-pushMessageCh:
					if !ok {
						return
					}
					fmt.Printf("< %s\n", message.String())
				}
			}
		}()

		for {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("$ ")
			text, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			text = text[:len(text)-1]
			if text == "q" {
				return nil
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
