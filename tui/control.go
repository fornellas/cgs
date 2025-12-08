package tui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
	"github.com/fornellas/cgs/worker_manager"
)

type TuiOptions struct {
	DisplayStatusComms bool
	AppLogger          *slog.Logger
}

type Tui struct {
	grbl    *grblMod.Grbl
	options *TuiOptions
}

func NewTui(grbl *grblMod.Grbl, options *TuiOptions) *Tui {
	if options == nil {
		options = &TuiOptions{}
	}
	return &Tui{
		grbl:    grbl,
		options: options,
	}
}

func (t *Tui) statusQueryWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			if err := t.grbl.SendRealTimeCommand(grblMod.RealTimeCommandStatusReportQuery); err != nil {
				err := fmt.Errorf("failed to send periodic status query real-time command: %w", err)
				return err
			}
		}
	}
}

func (t *Tui) Run(ctx context.Context) (err error) {
	// Application
	app := tview.NewApplication()
	app.EnableMouse(true)

	// Context & Logging
	consoleCtx, consoleLogger := log.MustWithGroup(ctx, "Control")
	logsPrimitive := NewLogsPrimitive(app)
	appHandler := NewEnabledOverrideHandler(
		log.NewTerminalTreeHandler(
			tview.ANSIWriter(logsPrimitive),
			&log.TerminalHandlerOptions{
				// tview.TextView does not handle emojis correctly: drawing is corrupted.
				DisableGroupEmoji: true,
				ForceColor:        true,
			},
		),
		consoleLogger.Handler(),
	)
	appHandlers := []slog.Handler{
		appHandler,
	}
	if t.options.AppLogger != nil {
		appHandlers = append(appHandlers, t.options.AppLogger.Handler())
	}
	appLogger := slog.New(log.NewMultiHandler(appHandlers...))
	appCtx := log.WithLogger(consoleCtx, appLogger)

	// Grbl
	grblPushMessageCh, err := t.grbl.Connect(consoleCtx)
	if err != nil {
		return err
	}

	// WorkerManager
	workerManager := worker_manager.NewWorkerManager()

	subscriberChSize := 50

	// Push Message Broker
	pushMessageBroker := NewPushMessageBroker()
	workerManager.AddWorker("PushMessageBroker", func(ctx context.Context) error {
		return pushMessageBroker.Worker(ctx, grblPushMessageCh)
	})

	// StateTracker
	stateTracker := NewStateTracker()
	workerManager.AddWorker("StateTracker", func(ctx context.Context) error {
		return stateTracker.Worker(
			ctx, pushMessageBroker.Subscribe("StateTracker", subscriberChSize),
		)
	})

	// StatusPrimitive
	statusPrimitive := NewStatusPrimitive(appCtx, t.grbl, app)
	workerManager.AddWorker("StatusPrimitive", func(ctx context.Context) error {
		return statusPrimitive.Worker(
			ctx,
			pushMessageBroker.Subscribe("StatusPrimitive", subscriberChSize),
			stateTracker.Subscribe("StatusPrimitive", subscriberChSize),
		)
	})

	// ControlPrimitive
	controlPrimitive := NewControlPrimitive(
		appCtx, t.grbl, app, stateTracker,
		!t.options.DisplayStatusComms,
	)
	workerManager.AddWorker("ControlPrimitive.Worker", func(ctx context.Context) error {
		return controlPrimitive.Worker(
			ctx,
			pushMessageBroker.Subscribe("ControlPrimitive", subscriberChSize),
			stateTracker.Subscribe("ControlPrimitive", subscriberChSize),
		)
	})

	// JoggingPrimitive
	joggingPrimitive := NewJoggingPrimitive(appCtx, app, controlPrimitive)
	workerManager.AddWorker("JoggingPrimitive", func(ctx context.Context) error {
		return joggingPrimitive.Worker(
			ctx,
			pushMessageBroker.Subscribe("JoggingPrimitive", subscriberChSize),
			stateTracker.Subscribe("JoggingPrimitive", subscriberChSize),
		)
	})

	// ProbePrimitive
	probePrimitive := NewProbePrimitive(appCtx, app, controlPrimitive)
	workerManager.AddWorker("ProbePrimitive", func(ctx context.Context) error {
		return probePrimitive.Worker(
			ctx,
			pushMessageBroker.Subscribe("ProbePrimitive", subscriberChSize),
			stateTracker.Subscribe("ProbePrimitive", subscriberChSize),
		)
	})

	// OverridesPrimitive
	overridesPrimitive := NewOverridesPrimitive(appCtx, app, controlPrimitive)
	workerManager.AddWorker("OverridesPrimitive", func(ctx context.Context) error {
		return overridesPrimitive.Worker(
			ctx,
			stateTracker.Subscribe("OverridesPrimitive", subscriberChSize),
		)
	})

	// StreamPrimitive
	heightMapPrimitive := NewHeightMapPrimitive(appCtx, app, controlPrimitive)
	workerManager.AddWorker("HeightMapPrimitive", func(ctx context.Context) error {
		return heightMapPrimitive.Worker(
			ctx,
			stateTracker.Subscribe("HeightMapPrimitive", subscriberChSize),
		)
	})
	streamPrimitive := NewStreamPrimitive(appCtx, app, controlPrimitive, heightMapPrimitive)
	workerManager.AddWorker("StreamPrimitive", func(ctx context.Context) error {
		return streamPrimitive.Worker(
			ctx,
			stateTracker.Subscribe("StreamPrimitive", subscriberChSize),
		)
	})

	// SettingsPrimitive
	settingsPrimitive := NewSettingsPrimitive(appCtx, app, controlPrimitive)
	workerManager.AddWorker("SettingsPrimitive", func(ctx context.Context) error {
		return settingsPrimitive.Worker(
			ctx,
			pushMessageBroker.Subscribe("SettingsPrimitive", subscriberChSize),
			stateTracker.Subscribe("SettingsPrimitive", subscriberChSize),
		)
	})

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
	workerManager.AddWorker("RootPrimitive", func(ctx context.Context) error {
		return rootPrimitive.Worker(
			ctx,
			pushMessageBroker.Subscribe("RootPrimitive", subscriberChSize),
			stateTracker.Subscribe("RootPrimitive", subscriberChSize),
		)
	})
	app.SetRoot(rootPrimitive, true)

	// Status Query
	workerManager.AddWorker("Control.statusQueryWorker", t.statusQueryWorker)

	// Start
	workerManager.Start(appCtx)

	// App Input
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlX {
			controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset)
			return nil
		}
		if event.Key() == tcell.KeyCtrlC {
			appLogger.Info("Exiting")
			workerManager.Cancel(appCtx)
			return nil
		}
		return event
	})

	// Exit
	var exitMu sync.Mutex
	exitMu.Lock()
	go func() {
		logger := log.MustLogger(appCtx)
		for name, workerErr := range workerManager.Wait(appCtx) {
			if errors.Is(workerErr, context.Canceled) {
				workerErr = nil
			}
			if workerErr != nil {
				workerErr = fmt.Errorf("%s: %w", name, workerErr)
			}
			err = errors.Join(err, workerErr)
		}
		logger.Info("Disconnecting")
		err = errors.Join(err, t.grbl.Disconnect(appCtx))
		logger.Info("Stopping App")
		app.Stop()
		exitMu.Unlock()
	}()
	defer func() { exitMu.Lock() }()
	defer func() {
		logger := log.MustLogger(appCtx)

		if r := recover(); r != nil {
			logger.Debug("Panic", "recovered", r, "stack", string(debug.Stack()))
		}

		// After Application.Run returns, any pending or future calls to Application.QueueUpdate
		// will block indefinitely.
		// This hack here spins the app again using a simulated screen, which enables any pending
		// Application.QueueUpdate to be processed, unblocking them, so that workers can properly
		// shutdown.
		app.SetScreen(tcell.NewSimulationScreen("UTF-8"))
		go func() {
			logger.Debug("Restarting app with simulated screen to support workers shutdown")
			logger.Debug("Simulated screen app returned", "err", app.Run())
		}()

		logger.Info("Stopping all workers")
		workerManager.Cancel(appCtx)
	}()

	if runErr := app.Run(); runErr != nil {
		consoleLogger.Error("Application failed", "err", runErr)
		err = errors.Join(err, runErr)
	}
	return
}
