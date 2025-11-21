package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

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

func (c *Control) Run(ctx context.Context) (err error) {
	ctx, logger := log.MustWithGroup(ctx, "Control.Run")

	logger.Info("Connecting")
	pushMessageCh, err := c.grbl.Connect(ctx)
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(ctx)

	app := tview.NewApplication()
	app.EnableMouse(true)

	logsPrimitive := NewLogsPrimitive(ctx, app)
	logger = slog.New(log.NewMultiHandler(
		logger.Handler(),
		log.NewTerminalTreeHandler(
			tview.ANSIWriter(logsPrimitive),
			// tview.ANSIWriter(w),
			&log.TerminalHandlerOptions{
				HandlerOptions: slog.HandlerOptions{
					// AddSource: ,
					Level: slog.Level(math.MinInt),
					// ReplaceAttr: ,
				},
				DisableGroupEmoji: true,
				// TimeLayout: ,
				// NoColor: true,
				ForceColor: true,
				// ColorScheme: ,
			},
		),
	))
	ctx = log.WithLogger(ctx, logger)

	var messageProcessors []MessageProcessor

	statusPrimitive := NewStatusPrimitive(ctx, c.grbl, app)
	messageProcessors = append(messageProcessors, statusPrimitive)

	controlPrimitive := NewControlPrimitive(
		ctx,
		c.grbl,
		pushMessageCh,
		app,
		statusPrimitive,
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

	overridesPrimitive := NewOverridesPrimitive(ctx, app, controlPrimitive)
	messageProcessors = append(messageProcessors, overridesPrimitive)

	joggingPrimitive := NewJoggingPrimitive(ctx, app, controlPrimitive)
	messageProcessors = append(messageProcessors, joggingPrimitive)

	rootPrimitive := NewRootPrimitive(
		ctx,
		app,
		statusPrimitive,
		controlPrimitive,
		overridesPrimitive,
		joggingPrimitive,
		logsPrimitive,
	)
	app.SetRoot(rootPrimitive, true)
	messageProcessors = append(messageProcessors, rootPrimitive)

	sendCommandWorkerErrCh := make(chan error, 1)
	go func() {
		ctx, logger := log.MustWithGroup(ctx, "go RunSendCommandWorker")
		defer func() {
			logger.Debug("Stopping app")
			app.Stop()
		}()
		logger.Debug("Starting")
		sendCommandWorkerErrCh <- controlPrimitive.RunSendCommandWorker(ctx)
	}()

	sendRealTimeCommandWorkerErrCh := make(chan error, 1)
	go func() {
		ctx, logger := log.MustWithGroup(ctx, "go RunSendRealTimeCommandWorker")
		defer func() {
			logger.Debug("Stopping app")
			app.Stop()
		}()
		logger.Debug("Starting")
		sendRealTimeCommandWorkerErrCh <- controlPrimitive.RunSendRealTimeCommandWorker(ctx)
	}()

	messageProcessorWorkerErrCh := make(chan error, 1)
	go func() {
		ctx, logger := log.MustWithGroup(ctx, "go messageProcessorWorker")
		defer func() {
			logger.Debug("Stopping app")
			app.Stop()
		}()
		logger.Debug("Starting")
		messageProcessorWorkerErrCh <- c.messageProcessorWorker(
			ctx,
			pushMessageCh,
			controlPrimitive,
			messageProcessors...,
		)
	}()

	statusQueryErrCh := make(chan error, 1)
	go func() {
		ctx, logger := log.MustWithGroup(ctx, "go statusQueryWorker")
		defer func() {
			logger.Debug("Stopping app")
			app.Stop()
		}()
		logger.Debug("Start")
		statusQueryErrCh <- c.statusQueryWorker(ctx)
	}()

	defer func() {
		_, logger := log.MustWithGroup(ctx, "go exit")
		logger.Debug("Cancelling context")
		cancelFn()
		logger.Debug("Waiting for statusQueryErrCh")
		err = errors.Join(err, <-statusQueryErrCh)
		logger.Debug("Waiting for sendCommandWorkerErrCh")
		err = errors.Join(err, <-sendCommandWorkerErrCh)
		logger.Debug("Waiting for sendRealTimeCommandWorkerErrCh")
		err = errors.Join(err, <-sendRealTimeCommandWorkerErrCh)
		logger.Debug("Waiting for messageProcessorWorkerErrCh")
		err = errors.Join(err, <-messageProcessorWorkerErrCh)
		logger.Debug("Disconnecting")
		err = errors.Join(err, c.grbl.Disconnect())
		logger.Debug("Done")
	}()

	return app.Run()
}
