package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
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
	controlPrimitive *ControlPrimitive,
	messageProcessors ...MessageProcessor,
) error {
	logger := log.MustLogger(ctx).WithGroup("messageProcessorWorker")

	logger.Debug("Sending G-Code commands")
	// Sending $G enables tracking of G-Code parsing state
	controlPrimitive.QueueCommand("$G")
	// Sending $G enables tracking of G-Code parameters
	controlPrimitive.QueueCommand("$#")

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
					Message: "(virtual status report: Alarm)",
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
	ctx context.Context, app *tview.Application,
	name string, fn func(context.Context) error,
) {
	_, logger := log.MustWithGroupAttrs(ctx, "Worker", "name", name)
	errCh := make(chan error, 1)
	go func() {
		defer func() {
			app.Stop()
			logger.Debug("Stopped")
		}()
		logger.Debug("Starting")
		errCh <- fn(ctx)
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

	// Application Logging
	logsPrimitive := NewLogsPrimitive(app)
	ctx, logger := log.MustWithGroup(ctx, "Control")
	originalHandler := logger.Handler()
	logsPrimitiveHandler := &EnabledOverrideHandler{
		Handler: log.NewTerminalTreeHandler(
			tview.ANSIWriter(logsPrimitive),
			&log.TerminalHandlerOptions{
				DisableGroupEmoji: true,
				ForceColor:        true,
			},
		),
		EnabledHandler: originalHandler,
	}
	primitiveLogger := slog.New(logsPrimitiveHandler)
	// From this point onwards:
	// - While app.Run is active, logger from ctx must be used.
	// - Otherwise, originalContext logger must be used.
	originalContext := ctx
	ctx = log.WithLogger(originalContext, primitiveLogger)

	// Grbl
	logger.Info("Connecting to Grbl")
	pushMessageCh, err := c.grbl.Connect(ctx)
	if err != nil {
		return err
	}
	logger.Info("Connected to Grbl")

	// Context Cancellation
	ctx, cancelFn := context.WithCancel(ctx)

	// Message Processors
	var messageProcessors []MessageProcessor

	// StatusPrimitive
	statusPrimitive := NewStatusPrimitive(ctx, c.grbl, app)
	messageProcessors = append(messageProcessors, statusPrimitive)

	// ControlPrimitive
	controlPrimitive := NewControlPrimitive(
		ctx,
		c.grbl, pushMessageCh,
		app, statusPrimitive,
		!c.options.DisplayGcodeParserStateComms,
		!c.options.DisplayGcodeParamStateComms,
		!c.options.DisplayStatusComms,
	)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		_, logger := log.MustWithGroup(ctx, "app.InputCapture")
		logger.Debug("Called", "event", event)
		if event.Key() == tcell.KeyCtrlX {
			logger.Debug("QueueRealTimeCommand SoftReset")
			controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset)
			return nil
		}
		return event
	})
	messageProcessors = append(messageProcessors, controlPrimitive)

	// OverridesPrimitive
	overridesPrimitive := NewOverridesPrimitive(ctx, app, controlPrimitive)
	messageProcessors = append(messageProcessors, overridesPrimitive)

	// JoggingPrimitive
	joggingPrimitive := NewJoggingPrimitive(ctx, app, controlPrimitive)
	messageProcessors = append(messageProcessors, joggingPrimitive)

	// RootPrimitive
	rootPrimitive := NewRootPrimitive(
		ctx, app,
		statusPrimitive,
		controlPrimitive,
		overridesPrimitive,
		joggingPrimitive,
		logsPrimitive,
	)
	app.SetRoot(rootPrimitive, true)
	messageProcessors = append(messageProcessors, rootPrimitive)

	// Workers
	c.startWorker(
		ctx, app,
		"Control.messageProcessorWorker",
		func(ctx context.Context) error {
			return c.messageProcessorWorker(
				ctx, pushMessageCh, controlPrimitive, messageProcessors...,
			)
		},
	)
	c.startWorker(
		ctx, app,
		"ControlPrimitive.RunSendCommandWorker",
		controlPrimitive.RunSendCommandWorker,
	)
	c.startWorker(
		ctx, app,
		"ControlPrimitive.RunSendRealTimeCommandWorker",
		controlPrimitive.RunSendRealTimeCommandWorker,
	)
	c.startWorker(ctx, app, "Control.statusQueryWorker", c.statusQueryWorker)

	// Defer
	defer func() {
		logger := logger.WithGroup("Exit")
		logger.Debug("Cancelling context")
		cancelFn()
		// There's a bug in tview. When Application.Run returns, any pending or future calls to
		// Application.QueueUpdate will block indefinitely (it should fail these calls).
		// Any worker that calls Application.QueueUpdate may get stuck on it, and not be able to
		// process the context cancellation.
		// This hack here spins the app again using a simulated screen, which enables any pending
		// Application.QueueUpdate to be processed, unblocking them, so the worker can return from
		// context cancellation.
		app.SetScreen(tcell.NewSimulationScreen("UTF-8"))
		errCh := make(chan error)
		go func() {
			errCh <- errors.Join(err, app.Run())
		}()
		err = errors.Join(err, c.waitForWorkers(originalContext))
		app.Stop()
		err = errors.Join(err, <-errCh)
		logger.Info("Disconnecting")
		err = errors.Join(err, c.grbl.Disconnect())
		logger.Debug("Done")
	}()

	return app.Run()
}
