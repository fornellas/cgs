package worker_manager

import (
	"context"
	"errors"

	"github.com/fornellas/slogxt/log"
)

type worker struct {
	name  string
	errCh chan error
}

// WorkerManager manages a group of workers and coordinates their execution.
type WorkerManager struct {
	workers    []worker
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewWorkerManager creates a new WorkerManager with the given context.
func NewWorkerManager(ctx context.Context) *WorkerManager {
	ctx, _ = log.MustWithGroup(ctx, "Worker Manager")
	ctx, cancelFunc := context.WithCancel(ctx)
	return &WorkerManager{
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
}

// StartWorker starts a new worker with the given name and function.
// The worker function receives the manager's context.
// When the worker function returns, the manager's context is cancelled (which impacts all workers).
func (wm *WorkerManager) StartWorker(name string, fn func(context.Context) error) {
	ctx, logger := log.MustWithGroupAttrs(wm.ctx, "Worker", "name", name)
	errCh := make(chan error, 1)
	go func() {
		logger.Debug("Starting")
		err := fn(ctx)
		logger.Debug("Finished", "err", err)
		if errors.Is(err, context.Canceled) {
			err = nil
		}
		errCh <- err
		wm.cancelFunc()
	}()
	wm.workers = append([]worker{{name: name, errCh: errCh}}, wm.workers...)
}

// Cancel cancels the manager's context, causing all workers to exit.
func (wm *WorkerManager) Cancel() {
	wm.cancelFunc()
}

// Wait blocks until all workers have completed and returns any errors that occurred.
func (wm *WorkerManager) Wait() (err error) {
	_, logger := log.MustWithGroup(wm.ctx, "Wait")
	for _, worker := range wm.workers {
		wLogger := logger.With("name", worker.name)
		wLogger.Debug("Waiting")
		wErr := <-worker.errCh
		wLogger.Debug("Result", "err", wErr)
		err = errors.Join(err, wErr)
	}
	logger.Debug("All workers finished")
	wm.workers = nil
	return
}
