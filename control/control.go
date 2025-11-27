package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
	"github.com/fornellas/cgs/worker"
)

type PushMessageProcessor interface {
	ProcessPushMessage(context.Context, grblMod.PushMessage)
}

type ControlOptions struct {
	DisplayStatusComms bool
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
				err := fmt.Errorf("failed to send periodic status query real-time command: %w", err)
				return err
			}
		}
	}
}

func (c *Control) pushMessageProcessorWorker(
	ctx context.Context,
	pushMessageCh chan grblMod.PushMessage,
	pushMessageProcessors ...PushMessageProcessor,
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

			if _, ok := message.(*grblMod.AlarmPushMessage); ok {
				// Grbl can generate an alarm push message, but then stop answering to real time
				// commands for status report query. This means that, there are effectively two sources
				// to look for alarm state.
				// We generate this virtual status report push message here, to simplify the rest of the
				// codebase, that only need to look for alarm state in a sigle place.
				pushMessageCh <- &grblMod.StatusReportPushMessage{
					Message: "(virtual push message: status report: Alarm)",
					MachineState: grblMod.MachineState{
						State: "Alarm",
					},
				}
			}

			for _, pushMessageProcessor := range pushMessageProcessors {
				pushMessageProcessor.ProcessPushMessage(ctx, message)
			}
		}
	}
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

	// Grbl
	consoleLogger.Info("Connecting to Grbl")
	pushMessageCh, err := c.grbl.Connect(consoleCtx)
	if err != nil {
		return err
	}

	// WorkerManager
	workerManager := worker.NewWorkerManager(appCtx)

	// Message Processors
	var pushMessageProcessors []PushMessageProcessor

	// StatusPrimitive
	statusPrimitive := NewStatusPrimitive(appCtx, c.grbl, app)
	pushMessageProcessors = append(pushMessageProcessors, statusPrimitive)

	// ControlPrimitive
	controlPrimitive := NewControlPrimitive(
		appCtx, c.grbl, pushMessageCh,
		app, statusPrimitive,
		!c.options.DisplayStatusComms,
	)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		_, logger := log.MustWithGroup(appCtx, "Application.InputCapture")
		logger.Debug("Called", "event", event)
		if event.Key() == tcell.KeyCtrlX {
			logger.Debug("QueueRealTimeCommand SoftReset")
			controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset)
			return nil
		}
		if event.Key() == tcell.KeyCtrlC {
			go func() { app.Stop() }()
			return nil
		}
		return event
	})
	pushMessageProcessors = append(pushMessageProcessors, controlPrimitive)

	// JoggingPrimitive
	joggingPrimitive := NewJoggingPrimitive(appCtx, app, controlPrimitive)
	pushMessageProcessors = append(pushMessageProcessors, joggingPrimitive)

	// probePrimitive
	probePrimitive := NewProbePrimitive(appCtx, app, controlPrimitive)
	pushMessageProcessors = append(pushMessageProcessors, probePrimitive)

	// OverridesPrimitive
	overridesPrimitive := NewOverridesPrimitive(appCtx, app, controlPrimitive)
	pushMessageProcessors = append(pushMessageProcessors, overridesPrimitive)

	// StreamPrimitive
	heightMapPrimitive := NewHeightMapPrimitive(appCtx, app, controlPrimitive)
	pushMessageProcessors = append(pushMessageProcessors, heightMapPrimitive)
	streamPrimitive := NewStreamPrimitive(appCtx, app, controlPrimitive, heightMapPrimitive)
	pushMessageProcessors = append(pushMessageProcessors, streamPrimitive)

	// settingsPrimitive
	settingsPrimitive := NewSettingsPrimitive(appCtx, app, controlPrimitive)
	pushMessageProcessors = append(pushMessageProcessors, settingsPrimitive)

	// RootPrimitive
	rootPrimitive := NewRootPrimitive(
		appCtx, app,
		statusPrimitive,
		controlPrimitive,
		joggingPrimitive,
		probePrimitive,
		overridesPrimitive,
		streamPrimitive,
		settingsPrimitive,
		logsPrimitive,
	)
	app.SetRoot(rootPrimitive, true)
	pushMessageProcessors = append(pushMessageProcessors, rootPrimitive)

	// Workers
	workerManager.StartWorker(
		"Control.messageProcessorWorker",
		func(ctx context.Context) error {
			return c.pushMessageProcessorWorker(ctx, pushMessageCh, pushMessageProcessors...)
		},
	)
	workerManager.StartWorker(
		"ControlPrimitive.RunSendCommandWorker",
		controlPrimitive.RunSendCommandWorker,
	)
	workerManager.StartWorker(
		"ControlPrimitive.RunSendRealTimeCommandWorker",
		controlPrimitive.RunSendRealTimeCommandWorker,
	)
	workerManager.StartWorker(
		"Control.statusQueryWorker",
		c.statusQueryWorker,
	)

	// Exit
	var exitMu sync.Mutex
	exitMu.Lock()
	defer func() { exitMu.Lock() }()
	defer func() {
		logger := log.MustLogger(appCtx)
		logger.Info("Stopping all workers")
		workerManager.Cancel()
	}()
	go func() {
		logger := log.MustLogger(appCtx)
		err = errors.Join(err, workerManager.Wait(appCtx))
		logger.Info("Disconnecting")
		err = errors.Join(err, c.grbl.Disconnect(appCtx))
		logger.Info("Stopping App")
		app.Stop()
		exitMu.Unlock()
	}()

	if runErr := app.Run(); runErr != nil {
		consoleLogger.Error("Application failed", "err", runErr)
		err = errors.Join(err, runErr)
	}
	return
}
