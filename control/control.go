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

	logger.Info("NewAppManager")
	c.AppManager = NewAppManager()
	defer func() { c.AppManager = nil }()

	logger.Info("NewCommandDispatcher")
	commandDispatcher := NewCommandDispatcher(
		c.grbl,
		c.AppManager,
		!c.options.DisplayGcodeParserStateComms,
		!c.options.DisplayGcodeParamStateComms,
		!c.options.DisplayStatusComms,
	)
	c.AppManager.CommandDispatcher = commandDispatcher

	logger.Info("Log")
	logger = slog.New(NewViewLogHandler(
		logger.Handler(),
		c.AppManager.PushMessagesLogsTextView,
	))
	ctx = log.WithLogger(ctx, logger)

	sendCommandWorkerErrCh := make(chan error, 1)
	go func() {
		defer cancelFn()
		// defer c.AppManager.App.Stop()
		e := commandDispatcher.RunSendCommandWorker(ctx)
		logger.Info("RunSendCommandWorker", "err", e)
		sendCommandWorkerErrCh <- e
	}()

	sendRealTimeCommandWorkerErrCh := make(chan error, 1)
	go func() {
		defer cancelFn()
		// defer c.AppManager.App.Stop()
		e := commandDispatcher.RunSendRealTimeCommandWorker(ctx)
		logger.Info("RunSendRealTimeCommandWorker", "err", e)
		sendRealTimeCommandWorkerErrCh <- e
	}()

	pushMessageErrCh := make(chan error, 1)
	go func() {
		defer cancelFn()
		// defer c.AppManager.App.Stop()
		// Sending $G enables tracking of G-Code parsing state
		commandDispatcher.QueueCommand("$G")
		// Sending $G enables tracking of G-Code parameters
		commandDispatcher.QueueCommand("$#")
		messageProcessor := NewMessageProcessor(
			c.grbl,
			c.AppManager,
			pushMessageCh,
			commandDispatcher,
			!c.options.DisplayGcodeParserStateComms,
			!c.options.DisplayGcodeParamStateComms,
			!c.options.DisplayStatusComms,
		)
		e := messageProcessor.Run(ctx)
		logger.Info("messageProcessor.Run", "err", e)
		pushMessageErrCh <- e
	}()

	statusQueryErrCh := make(chan error, 1)
	go func() {
		defer cancelFn()
		// defer c.AppManager.App.Stop()
		statusQueryErrCh <- c.statusQueryWorker(ctx)
	}()

	defer func() {
		cancelFn()
		err = errors.Join(err, <-sendCommandWorkerErrCh)
		err = errors.Join(err, <-sendRealTimeCommandWorkerErrCh)
		err = errors.Join(err, <-pushMessageErrCh)
		err = errors.Join(err, <-statusQueryErrCh)
		logger.Info("Disconnecting")
		err = errors.Join(err, c.grbl.Disconnect())
	}()
	logger.Info("App.Run")
	return c.AppManager.App.Run()
}
