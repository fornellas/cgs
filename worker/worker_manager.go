package worker

import (
	"context"
	"errors"
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
		errCh <- fn(c.ctx)
		c.cancelFunc()
	}()
	c.workers = append([]worker{{name: name, errCh: errCh}}, c.workers...)
}

// Wait blocks until all workers have completed and returns any errors that occurred.
func (c *WorkerManager) Wait() (err error) {
	for _, worker := range c.workers {
		err = errors.Join(err, <-worker.errCh)
	}
	c.workers = nil
	return
}
