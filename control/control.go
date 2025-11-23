package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type worker struct {
	name  string
	errCh chan error
}

type MessageProcessor interface {
	ProcessMessage(ctx context.Context, message grblMod.Message)
}

type ControlOptions struct {
	DisplayStatusComms           bool
	DisplayGcodeParserStateComms bool
	DisplayGcodeParamStateComms  bool
}

type Control struct {
	grbl    *grblMod.Grbl
	options *ControlOptions
	workers []worker
}

func NewControl(grbl *grblMod.Grbl, options *ControlOptions) *Control {
	if options == nil {
		options = &ControlOptions{}
	}
	return &Control{
		grbl:    grbl,
		options: options,
	}
}

func (c *Control) statusQueryWorker(ctx context.Context) error {
	logger := log.MustLogger(ctx).WithGroup("statusQueryWorker")
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			logger.Debug("Exiting", "err", err)
			return err
		case <-time.After(200 * time.Millisecond):
			if err := c.grbl.SendRealTimeCommand(grblMod.RealTimeCommandStatusReportQuery); err != nil {
				err := fmt.Errorf("failed to send periodic status query real-time command: %w", err)
				logger.Debug("Exiting", "err", err)
				return err
			}
		}
	}
}

func (c *Control) messageProcessorWorker(
	ctx context.Context,
	pushMessageCh chan grblMod.Message,
	messageProcessors ...MessageProcessor,
) error {
	logger := log.MustLogger(ctx).WithGroup("messageProcessorWorker")

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			logger.Debug("Exiting", "err", err)
			return err
		case message, ok := <-pushMessageCh:
			if !ok {
				err := fmt.Errorf("push message channel closed")
				logger.Debug("Exiting", "err", err)
				return err
			}

			msgLogger := logger.WithGroup("Message").With("message", message, "type", reflect.TypeOf(message))
			msgLogger.Debug("Received")

			if _, ok := message.(*grblMod.MessagePushAlarm); ok {
				// Grbl can generate an alarm push message, but then stop answering to real time
				// commands for status report query. This means that, there are effectively two sources
				// to look for alarm state.
				// We generate this virtual status report push message here, to simplify the rest of the
				// codebase, that only need to look for alarm state in a sigle place.
				pushMessageCh <- &grblMod.MessagePushStatusReport{
					Message: "(virtual push message: status report: Alarm)",
					MachineState: grblMod.StatusReportMachineState{
						State: "Alarm",
					},
				}
			}

			for _, messageProcessor := range messageProcessors {
				msgLogger.Debug("Processor", "type", reflect.TypeOf(messageProcessor))
				messageProcessor.ProcessMessage(ctx, message)
			}
			msgLogger.Debug("Done")
		}
	}
}
func (c *Control) startWorker(
	ctx context.Context, exitFn func(ctx context.Context, stop bool),
	name string, fn func(context.Context) error,
) {
	_, logger := log.MustWithGroupAttrs(ctx, "Worker", "name", name)
	errCh := make(chan error, 1)
	go func() {
		logger.Debug("Starting")
		errCh <- fn(ctx)
		logger.Debug("Stopped")
		exitFn(ctx, true)
	}()
	c.workers = append([]worker{{name: name, errCh: errCh}}, c.workers...)
}

func (c *Control) waitForWorkers(ctx context.Context) (err error) {
	_, logger := log.MustWithGroupAttrs(ctx, "waitForWorkers", "total", len(c.workers))
	for i, worker := range c.workers {
		logger.Debug("Waiting for worker", "name", worker.name, "number", i+1)
		err = errors.Join(err, <-worker.errCh)
	}
	logger.Debug("All workers stopped")
	c.workers = nil
	return
}

func (c *Control) Run(ctx context.Context) (err error) {
	// Application
	app := tview.NewApplication()
	app.EnableMouse(true)

	// Context & Logging
	consoleCtx, consoleLogger := log.MustWithGroup(ctx, "Control")
	logsPrimitive := NewLogsPrimitive(app)
	appLogger := slog.New(&EnabledOverrideHandler{
		Handler: log.NewTerminalTreeHandler(
			tview.ANSIWriter(logsPrimitive),
			&log.TerminalHandlerOptions{
				// tview.TextView does not handle emojis correctly: drawing is corrupted.
				DisableGroupEmoji: true,
				ForceColor:        true,
			},
		),
		EnabledHandler: consoleLogger.Handler(),
	})
	appCtx := log.WithLogger(consoleCtx, appLogger)
	appCtx, cancel := context.WithCancel(appCtx)
	var exitOnce sync.Once
	exitFn := func(ctx context.Context, stop bool) {
		exitOnce.Do(func() {
			cancel()
			logger := log.MustLogger(ctx)
			err = errors.Join(err, c.waitForWorkers(ctx))
			logger.Info("Disconnecting")
			err = errors.Join(err, c.grbl.Disconnect(ctx))
			if stop {
				app.Stop()
			}
		})
	}

	// Grbl
	consoleLogger.Info("Connecting to Grbl")
	pushMessageCh, err := c.grbl.Connect(consoleCtx)
	if err != nil {
		cancel()
		return err
	}

	// Message Processors
	var messageProcessors []MessageProcessor

	// StatusPrimitive
	statusPrimitive := NewStatusPrimitive(appCtx, c.grbl, app)
	messageProcessors = append(messageProcessors, statusPrimitive)

	// ControlPrimitive
	controlPrimitive := NewControlPrimitive(
		appCtx, c.grbl, pushMessageCh,
		app, statusPrimitive,
		!c.options.DisplayGcodeParserStateComms,
		!c.options.DisplayGcodeParamStateComms,
		!c.options.DisplayStatusComms,
	)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		appCtx, logger := log.MustWithGroup(appCtx, "Application.InputCapture")
		logger.Debug("Called", "event", event)
		if event.Key() == tcell.KeyCtrlX {
			logger.Debug("QueueRealTimeCommand SoftReset")
			controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset)
			return nil
		}
		if event.Key() == tcell.KeyCtrlC {
			go func() {
				logger.Info("Exiting")
				exitFn(appCtx, true)
			}()
			return nil
		}
		return event
	})
	messageProcessors = append(messageProcessors, controlPrimitive)

	// JoggingPrimitive
	joggingPrimitive := NewJoggingPrimitive(appCtx, app, controlPrimitive)
	messageProcessors = append(messageProcessors, joggingPrimitive)

	// OverridesPrimitive
	overridesPrimitive := NewOverridesPrimitive(appCtx, app, controlPrimitive)
	messageProcessors = append(messageProcessors, overridesPrimitive)

	// settingsPrimitive
	settingsPrimitive := NewSettingsPrimitive(appCtx, app, controlPrimitive)
	messageProcessors = append(messageProcessors, settingsPrimitive)

	// RootPrimitive
	rootPrimitive := NewRootPrimitive(
		appCtx, app,
		statusPrimitive,
		controlPrimitive,
		joggingPrimitive,
		overridesPrimitive,
		settingsPrimitive,
		logsPrimitive,
	)
	app.SetRoot(rootPrimitive, true)
	messageProcessors = append(messageProcessors, rootPrimitive)

	// Workers
	c.startWorker(
		appCtx, exitFn,
		"Control.messageProcessorWorker",
		func(ctx context.Context) error {
			return c.messageProcessorWorker(ctx, pushMessageCh, messageProcessors...)
		},
	)
	c.startWorker(
		appCtx, exitFn,
		"ControlPrimitive.RunSendCommandWorker",
		controlPrimitive.RunSendCommandWorker,
	)
	c.startWorker(
		appCtx, exitFn,
		"ControlPrimitive.RunSendRealTimeCommandWorker",
		controlPrimitive.RunSendRealTimeCommandWorker,
	)
	c.startWorker(appCtx, exitFn, "Control.statusQueryWorker", c.statusQueryWorker)

	if runErr := app.Run(); runErr != nil {
		consoleLogger.Error("Application failed", "err", err)
		err = errors.Join(err, runErr)
		exitFn(consoleCtx, false)
	}
	return
}
