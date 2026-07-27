package worker

import "context"

// Dispatcher routes fire-and-forget async jobs to the worker pool.
// It replaces a message queue for in-process background tasks such as
// sending emails, creating notifications, and writing audit logs.
type Dispatcher struct {
	pool *Pool
}

func NewDispatcher(workers int) *Dispatcher {
	return &Dispatcher{pool: New(workers, workers*200)}
}

// Dispatch enqueues an async task. The caller's context is detached so the
// task outlives the originating HTTP request. Returns false if the queue is full.
func (d *Dispatcher) Dispatch(ctx context.Context, fn func(ctx context.Context)) bool {
	return d.pool.SubmitCtx(ctx, fn)
}

// Stop drains the queue and shuts down all workers gracefully.
func (d *Dispatcher) Stop() { d.pool.Stop() }

// QueueSize returns the number of pending tasks.
func (d *Dispatcher) QueueSize() int { return d.pool.QueueSize() }
