package worker

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
func (c *WorkerManager) StartWorker(name string, fn func(context.Context) error) {
	errCh := make(chan error, 1)
	go func() {
		ctx, logger := log.MustWithGroup(c.ctx, name)
		logger.Debug("Starting")
		err := fn(ctx)
		logger.Debug("Finished", "err", err)
		errCh <- err
		c.cancelFunc()
	}()
	c.workers = append([]worker{{name: name, errCh: errCh}}, c.workers...)
}

// Cancel cancels the manager's context, causing all workers to exit.
func (c *WorkerManager) Cancel() {
	c.cancelFunc()
}

// Wait blocks until all workers have completed and returns any errors that occurred.
func (c *WorkerManager) Wait(ctx context.Context) (err error) {
	logger := log.MustLogger(ctx)
	for _, worker := range c.workers {
		logger.Debug("Waiting", "worker", worker.name)
		err = errors.Join(err, <-worker.errCh)
	}
	logger.Debug("All workers finished")
	c.workers = nil
	return
}
