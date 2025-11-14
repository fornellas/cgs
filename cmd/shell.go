package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/fornellas/slogxt/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"

	grblMod "github.com/fornellas/cgs/grbl"
)

func sendRealTimeCommand(ctx context.Context, g *grblMod.Grbl, cmd grblMod.RealTimeCommand, realTimeTextView *tview.TextView) error {
	if err := g.SendRealTimeCommand(ctx, cmd); err != nil {
		return err
	}
	fmt.Fprintf(realTimeTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(cmd.String()))
	return nil
}

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

		app := tview.NewApplication()
		app.EnableMouse(true)

		commandsTextView := tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(true).
			SetWrap(true)
		commandsTextView.SetBorder(true).SetTitle("Commands")
		commandsTextView.SetChangedFunc(func() {
			commandsTextView.ScrollToEnd()
			app.Draw()
		})

		realTimeTextView := tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(true).
			SetWrap(true)
		realTimeTextView.SetBorder(true).SetTitle("Real-Time")
		realTimeTextView.SetChangedFunc(func() {
			realTimeTextView.ScrollToEnd()
			app.Draw()
		})

		feedbackTextView := tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(true).
			SetWrap(true)
		feedbackTextView.SetTitle("Feedback Message")
		feedbackTextView.SetChangedFunc(func() {
			feedbackTextView.ScrollToEnd()
			app.Draw()
		})

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

					var color = tcell.ColorGreen
					var detailsFn func()
					if _, ok := message.(*grblMod.MessagePushWelcome); ok {
						color = tcell.ColorYellow
						detailsFn = func() {
							fmt.Fprintf(realTimeTextView, "[%s]Soft-Reset detected[-]\n", color)
						}
						feedbackTextView.SetText("")
					}
					if messagePushAlarm, ok := message.(*grblMod.MessagePushAlarm); ok {
						color = tcell.ColorRed
						detailsFn = func() {
							fmt.Fprintf(realTimeTextView, "[%s]%s[-]\n", color, tview.Escape(messagePushAlarm.Error().Error()))
						}
					}
					if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
						switch messagePushStatusReport.MachineState.State {
						case "Idle":
							color = tcell.ColorGreen
						case "Run":
							color = tcell.ColorLightCyan
						case "Hold":
							color = tcell.ColorYellow
						case "Jog":
							color = tcell.ColorBlue
						case "Alarm":
							color = tcell.ColorRed
						case "Door":
							color = tcell.ColorOrange
						case "Check":
							color = tcell.ColorBlue
						case "Home":
							color = tcell.ColorLime
						case "Sleep":
							color = tcell.ColorSilver
						}
					}
					if messagePushFeedback, ok := message.(*grblMod.MessagePushFeedback); ok {
						feedbackTextView.SetText(messagePushFeedback.Text())
					}
					text := message.String()
					if len(text) == 0 {
						fmt.Fprintf(realTimeTextView, "\n\n")
					} else {
						fmt.Fprintf(realTimeTextView, "[%s]%s[-]\n", color, tview.Escape(text))
					}
					if detailsFn != nil {
						detailsFn()
					}
				}
			}
		}()

		commandInputField := tview.NewInputField().
			SetLabel("Command: ")
		commandInputField.SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyEscape:
				commandInputField.SetText("")
			case tcell.KeyEnter:
				command := commandInputField.GetText()
				if command != "" {
					fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(command))

					message, err := grbl.SendCommand(ctx, command)
					if err != nil {
						fmt.Fprintf(commandsTextView, "[%s]Failed to send: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
					} else {
						messageResponse := message.(*grblMod.MessageResponse)
						if messageResponse.Error() == nil {
							fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorGreen, tview.Escape(messageResponse.String()))
						} else {
							fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.String()))
							fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.Error().Error()))
						}
					}
					commandInputField.SetText("")
				}
			}
		})

		app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyCtrlX {
				if err := sendRealTimeCommand(ctx, grbl, grblMod.RealTimeCommandSoftReset, realTimeTextView); err != nil {
					fmt.Fprintf(realTimeTextView, "[%s]Failed to send soft reset: %s[-]\n", tcell.ColorRed, err)
				}
				return nil
			}
			return event
		})

		rootFlex := tview.NewFlex().
			AddItem(
				tview.NewFlex().SetDirection(tview.FlexRow).
					AddItem(
						tview.NewFlex().
							AddItem(
								tview.NewFlex().SetDirection(tview.FlexRow).
									AddItem(realTimeTextView, 0, 1, false).
									AddItem(commandsTextView, 0, 1, false),
								0, 3, false,
							).
							AddItem(tview.NewBox().SetBorder(true).SetTitle("G-Code Parser"), 0, 1, false).
							AddItem(tview.NewBox().SetBorder(true).SetTitle("Status"), 14, 0, false),
						0, 1, false,
					).
					AddItem(feedbackTextView, 1, 0, false).
					AddItem(commandInputField, 1, 0, false),
				0, 1, false,
			)

		return app.SetRoot(rootFlex, true).SetFocus(commandInputField).Run()
	}),
}

func init() {
	AddPortFlags(ShellCmd)

	RootCmd.AddCommand(ShellCmd)

	resetFlagsFns = append(resetFlagsFns, func() {
	})
}
