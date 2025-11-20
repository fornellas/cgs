package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type MessageProcessor interface {
	ProcessMessage(message grblMod.Message)
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
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case <-time.After(200 * time.Millisecond):
			if err := c.grbl.SendRealTimeCommand(grblMod.RealTimeCommandStatusReportQuery); err != nil {
				return fmt.Errorf("failed to send periodic status query real-time command: %w", err)
			}
		}
	}
}

func (c *Control) messageProcessorWorker(
	ctx context.Context,
	pushMessageCh chan grblMod.Message,
	messageProcessors ...MessageProcessor,
) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case message, ok := <-pushMessageCh:
			if !ok {
				return fmt.Errorf("push message channel closed")
			}
			for _, messageProcessor := range messageProcessors {
				messageProcessor.ProcessMessage(message)
			}
		}
	}
}

func (c *Control) Run(ctx context.Context) (err error) {
	logger := log.MustLogger(ctx)

	logger.Info("Connecting")
	pushMessageCh, err := c.grbl.Connect(ctx)
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(ctx)

	app := tview.NewApplication()
	app.EnableMouse(true)

	controlPrimitive := NewControlPrimitive(
		app,
		c.grbl,
		!c.options.DisplayGcodeParserStateComms,
		!c.options.DisplayGcodeParamStateComms,
		!c.options.DisplayStatusComms,
	)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlX {
			controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset)
			return nil
		}
		return event
	})

	overridesPrimitive := NewOverridesPrimitive(app, controlPrimitive)

	joggingPrimitive := NewJoggingPrimitive(controlPrimitive)

	rootPrimitive := NewRootPrimitive(
		app,
		c.grbl,
		controlPrimitive,
		overridesPrimitive,
		joggingPrimitive,
	)
	app.SetRoot(rootPrimitive, true)

	logger = slog.New(NewViewLogHandler(
		logger.Handler(),
		controlPrimitive.GetLogsTextView(),
	))
	ctx = log.WithLogger(ctx, logger)

	sendCommandWorkerErrCh := make(chan error, 1)
	go func() {
		defer cancelFn()
		defer app.Stop()
		sendCommandWorkerErrCh <- controlPrimitive.RunSendCommandWorker(ctx)
	}()

	sendRealTimeCommandWorkerErrCh := make(chan error, 1)
	go func() {
		defer cancelFn()
		defer app.Stop()
		sendRealTimeCommandWorkerErrCh <- controlPrimitive.RunSendRealTimeCommandWorker(ctx)
	}()

	pushMessageErrCh := make(chan error, 1)
	go func() {
		defer cancelFn()
		defer app.Stop()
		// Sending $G enables tracking of G-Code parsing state
		controlPrimitive.QueueCommand("$G")
		// Sending $G enables tracking of G-Code parameters
		controlPrimitive.QueueCommand("$#")
		pushMessageErrCh <- c.messageProcessorWorker(
			ctx,
			pushMessageCh,
			controlPrimitive,
			overridesPrimitive,
			joggingPrimitive,
		)
	}()

	statusQueryErrCh := make(chan error, 1)
	go func() {
		defer cancelFn()
		defer app.Stop()
		statusQueryErrCh <- c.statusQueryWorker(ctx)
	}()

	defer func() {
		cancelFn()
		err = errors.Join(err, <-sendCommandWorkerErrCh)
		err = errors.Join(err, <-sendRealTimeCommandWorkerErrCh)
		err = errors.Join(err, <-pushMessageErrCh)
		err = errors.Join(err, <-statusQueryErrCh)
		err = errors.Join(err, c.grbl.Disconnect())
	}()

	return app.Run()
}
