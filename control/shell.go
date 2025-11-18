package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fornellas/slogxt/log"

	grblMod "github.com/fornellas/cgs/grbl"
)

type ControlOptions struct {
	DisplayStatusComms           bool
	DisplayGcodeParserStateComms bool
	DisplayGcodeParamStateComms  bool
}

type Control struct {
	grbl       *grblMod.Grbl
	options    *ControlOptions
	AppManager *AppManager
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

func (c *Control) Run(ctx context.Context) (err error) {
	logger := log.MustLogger(ctx)
	logger.Info("Connecting")

	pushMessageCh, err := c.grbl.Connect(ctx)
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(ctx)

	sendCommandCh := make(chan string, 10)
	sendCommandWorkerErrCh := make(chan error, 1)

	sendRealTimeCommandCh := make(chan grblMod.RealTimeCommand, 10)
	sendRealTimeCommandWorkerErrCh := make(chan error, 1)

	pushMessageErrCh := make(chan error, 1)

	statusQueryErrCh := make(chan error, 1)

	c.AppManager = NewAppManager(sendCommandCh, sendRealTimeCommandCh)
	defer func() { c.AppManager = nil }()

	logger = slog.New(NewViewLogHandler(
		logger.Handler(),
		c.AppManager.PushMessagesLogsTextView,
	))
	ctx = log.WithLogger(ctx, logger)

	commandDispatcher := NewCommandDispatcher(
		c.grbl,
		c.AppManager,
		!c.options.DisplayGcodeParserStateComms,
		!c.options.DisplayGcodeParamStateComms,
		!c.options.DisplayStatusComms,
	)
	go func() {
		defer cancelFn()
		defer c.AppManager.App.Stop()
		sendCommandWorkerErrCh <- commandDispatcher.RunSendCommandWorker(ctx, sendCommandCh)
	}()
	go func() {
		defer cancelFn()
		defer c.AppManager.App.Stop()
		sendRealTimeCommandWorkerErrCh <- commandDispatcher.RunSendRealTimeCommandWorker(
			ctx, sendRealTimeCommandCh,
		)
	}()
	go func() {
		defer cancelFn()
		defer c.AppManager.App.Stop()
		// Sending $G enables tracking of G-Code parsing state
		commandDispatcher.SendCommand(ctx, "$G")
		// Sending $G enables tracking of G-Code parameters
		commandDispatcher.SendCommand(ctx, "$#")
		messageProcessor := NewMessageProcessor(
			c.grbl,
			c.AppManager,
			pushMessageCh,
			commandDispatcher.SendCommand,
			!c.options.DisplayGcodeParserStateComms,
			!c.options.DisplayGcodeParamStateComms,
			!c.options.DisplayStatusComms,
		)
		pushMessageErrCh <- messageProcessor.Run(ctx)
	}()
	go func() {
		defer cancelFn()
		defer c.AppManager.App.Stop()
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
	return c.AppManager.App.Run()
}
