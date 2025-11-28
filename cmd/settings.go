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

		logger.Info("Requesting settings")
		for _, fn := range []func(context.Context) error{
			grbl.SendGrblCommandViewGrblSettings,
			grbl.SendGrblCommandViewStartupBlocks,
			grbl.SendGrblCommandViewBuildInfo,
			grbl.SendGrblCommandViewGcodeParameters,
		} {
			sendCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
			err = fn(sendCtx)
			cancel()
			if err != nil {
				return err
			}
		}

		readCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		fmt.Fprintln(output, "$H")
		for {
			select {
			case msg, ok := <-pushMessageCh:
				if !ok {
					return fmt.Errorf("push message channel closed unexpectedly")
				}

				if settingPushMessage, ok := msg.(*grblMod.SettingPushMessage); ok {
					if _, err := fmt.Fprintln(output, settingPushMessage.String()); err != nil {
						return err
					}
				}

				if startupLineExecutionPushMessage, ok := msg.(*grblMod.StartupLineExecutionPushMessage); ok {
					if _, err := fmt.Fprintln(output, startupLineExecutionPushMessage.String()); err != nil {
						return err
					}
				}

				if versionPushMessage, ok := msg.(*grblMod.VersionPushMessage); ok {
					if _, err := fmt.Fprintf(output, "$I=%s\n", versionPushMessage.Info); err != nil {
						return err
					}
				}

				if gcodeParamPushMessage, ok := msg.(*grblMod.GcodeParamPushMessage); ok {
					for coordinates, gcode := range map[*grblMod.Coordinates]string{
						// Coordinate system 1 (G54)
						gcodeParamPushMessage.GcodeParameters.CoordinateSystem1: "G10 L2 P1",
						// Coordinate system 2 (G55)
						gcodeParamPushMessage.GcodeParameters.CoordinateSystem2: "G10 L2 P2",
						// Coordinate system 3 (G56)
						gcodeParamPushMessage.GcodeParameters.CoordinateSystem3: "G10 L2 P3",
						// Coordinate system 4 (G57)
						gcodeParamPushMessage.GcodeParameters.CoordinateSystem4: "G10 L2 P4",
						// Coordinate system 5 (G58)
						gcodeParamPushMessage.GcodeParameters.CoordinateSystem5: "G10 L2 P5",
						// Coordinate system 6 (G59)
						gcodeParamPushMessage.GcodeParameters.CoordinateSystem6: "G10 L2 P6",
					} {
						if coordinates == nil {
							continue
						}
						if _, err := fmt.Fprintf(output, "%s X%.4f Y%.4f Z%.4f", gcode, coordinates.X, coordinates.Y, coordinates.Z); err != nil {
							return err
						}
						if coordinates.A != nil {
							if _, err := fmt.Fprintf(output, " A%.4f", *coordinates.A); err != nil {
								return err
							}
						}
						if _, err := fmt.Fprintln(output); err != nil {
							return err
						}
					}
					// Primary Pre-Defined Position (G28)
					if gcodeParamPushMessage.GcodeParameters.PrimaryPreDefinedPosition != nil {
						if _, err := fmt.Fprintf(
							output, "G0 G53 X%.4f Y%.4f Z%.4f",
							gcodeParamPushMessage.GcodeParameters.PrimaryPreDefinedPosition.X,
							gcodeParamPushMessage.GcodeParameters.PrimaryPreDefinedPosition.Y,
							gcodeParamPushMessage.GcodeParameters.PrimaryPreDefinedPosition.Z,
						); err != nil {
							return err
						}
						if gcodeParamPushMessage.GcodeParameters.PrimaryPreDefinedPosition.A != nil {
							if _, err := fmt.Fprintf(output, " A%.4f", *gcodeParamPushMessage.GcodeParameters.PrimaryPreDefinedPosition.A); err != nil {
								return err
							}
						}
						if _, err := fmt.Fprintln(output); err != nil {
							return err
						}
						if _, err := fmt.Fprintf(output, "G28.1\n"); err != nil {
							return err
						}
					}
					// Secondary Pre-Defined Position (G30)
					if gcodeParamPushMessage.GcodeParameters.SecondaryPreDefinedPosition != nil {
						if _, err := fmt.Fprintf(
							output, "G0 G53 X%.4f Y%.4f Z%.4f",
							gcodeParamPushMessage.GcodeParameters.SecondaryPreDefinedPosition.X,
							gcodeParamPushMessage.GcodeParameters.SecondaryPreDefinedPosition.Y,
							gcodeParamPushMessage.GcodeParameters.SecondaryPreDefinedPosition.Z,
						); err != nil {
							return err
						}
						if gcodeParamPushMessage.GcodeParameters.SecondaryPreDefinedPosition.A != nil {
							if _, err := fmt.Fprintf(output, " A%.4f", *gcodeParamPushMessage.GcodeParameters.SecondaryPreDefinedPosition.A); err != nil {
								return err
							}
						}
						if _, err := fmt.Fprintln(output); err != nil {
							return err
						}
						if _, err := fmt.Fprintf(output, "G30.1\n"); err != nil {
							return err
						}
					}
				}
			case <-readCtx.Done():
				return ctx.Err()
			default:
				return nil
			}
		}
	}),
}

func init() {
	AddPortFlags(SettingsSaveCmd)

	SettingsCmd.AddCommand(SettingsSaveCmd)
	RootCmd.AddCommand(SettingsCmd)
}
