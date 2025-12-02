package worker_manager

import (
	"context"

	"github.com/fornellas/slogxt/log"
)

type workerType struct {
	name       string
	fn         func(context.Context) error
	cancelFunc context.CancelFunc
	errCh      chan error
}

// WorkerManager manages a group of workers and coordinates their execution.
type WorkerManager struct {
	workers []workerType
}

// NewWorkerManager creates a new WorkerManager with the given context.
func NewWorkerManager() *WorkerManager {
	return &WorkerManager{}
}

func (wm *WorkerManager) AddWorker(name string, fn func(context.Context) error) {
	wm.workers = append([]workerType{{name: name, fn: fn}}, wm.workers...)
}

func (wm *WorkerManager) Start(ctx context.Context) {
	ctx, logger := log.MustWithGroup(ctx, "Worker Manager > Workers")
	logger.Debug("Starting workers")
	for i := range wm.workers {
		workerCtx, workerLogger := log.MustWithGroup(ctx, wm.workers[i].name)
		workerCtx, wm.workers[i].cancelFunc = context.WithCancel(workerCtx)
		wm.workers[i].errCh = make(chan error, 1)
		go func() {
			workerLogger.Debug("Starting")
			wm.workers[i].errCh <- wm.workers[i].fn(workerCtx)
			workerLogger.Debug("Finished")
			wm.Cancel(workerCtx)
		}()
	}
	logger.Debug("All workers started")
}

func (wm *WorkerManager) Cancel(ctx context.Context) {
	logger := log.MustLogger(ctx).WithGroup("Worker Manager > Cancel")
	if len(wm.workers) == 0 {
		return
	}
	worker := wm.workers[0]
	logger = logger.With("name", worker.name)
	logger.Debug("Cancelling")
	worker.cancelFunc()
}

func (wm *WorkerManager) Wait(ctx context.Context) map[string]error {
	logger := log.MustLogger(ctx).WithGroup("Worker Manager > Wait")
	logger.Debug("Waiting for all workers")
	errMap := map[string]error{}
	for i, worker := range wm.workers {
		workerLogger := logger.WithGroup(worker.name)
		if i > 0 {
			workerLogger.Debug("Cancelling")
			worker.cancelFunc()
		}
		workerLogger.Debug("Waiting")
		errMap[worker.name] = <-worker.errCh
	}
	wm.workers = nil
	logger.Debug("All workers returned")
	return errMap
}
