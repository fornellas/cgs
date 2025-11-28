package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/spf13/cobra"

	grblMod "github.com/fornellas/cgs/grbl"
)

var SettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage Grbl settings.",
	Args:  cobra.NoArgs,
}

var SettingsSaveCmd = &cobra.Command{
	Use:   "save [path]",
	Short: "Read Grbl settings and output to stdout or save to file.",
	Args:  cobra.MaximumNArgs(1),
	Run: GetRunFn(func(cmd *cobra.Command, args []string) (err error) {
		var path string
		if len(args) > 0 {
			path = args[0]
		}

		ctx, logger := log.MustWithAttrs(
			cmd.Context(),
			"port-name", portName,
			"address", address,
			"timeout", timeout,
			"path", path,
		)
		cmd.SetContext(ctx)

		// Determine output writer
		var output io.Writer
		var file *os.File
		if path != "" {
			var err error
			file, err = os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer func() { err = errors.Join(err, file.Close()) }()
			output = file
		} else {
			output = cmd.OutOrStdout()
		}

		// Get port connection function
		openPortFn, err := GetOpenPortFn()
		if err != nil {
			return err
		}

		// Connect to Grbl
		grbl := grblMod.NewGrbl(openPortFn)
		pushMessageCh, err := grbl.Connect(ctx)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, grbl.Disconnect(ctx)) }()

		// Send command to read settings
		logger.Info("Requesting Grbl Settings")
		sendCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		err = grbl.SendGrblCommandViewGrblSettings(sendCtx)
		cancel()
		if err != nil {
			return err
		}

		// Read settings with timeout
		readCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		settingCount := 0
		for {
			select {
			case msg, ok := <-pushMessageCh:
				if !ok {
					return fmt.Errorf("push message channel closed unexpectedly")
				}

				if settingMsg, ok := msg.(*grblMod.SettingPushMessage); ok {
					if _, err := fmt.Fprintln(output, settingMsg.String()); err != nil {
						return err
					}
					settingCount++
				}
			case <-readCtx.Done():
				return ctx.Err()
			default:
				return
			}
		}
	}),
}

func init() {
	AddPortFlags(SettingsSaveCmd)

	SettingsCmd.AddCommand(SettingsSaveCmd)
	RootCmd.AddCommand(SettingsCmd)
}
