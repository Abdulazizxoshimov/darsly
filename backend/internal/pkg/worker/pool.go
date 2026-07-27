package worker

import (
	"context"
	"sync"
)

// Pool manages a fixed set of goroutines that process submitted tasks.
type Pool struct {
	queue chan func()
	wg    sync.WaitGroup
	once  sync.Once
}

// New creates a worker pool with the given number of goroutines and queue buffer.
// Sensible defaults: workers=10, bufSize=workers*100.
func New(workers, bufSize int) *Pool {
	if workers <= 0 {
		workers = 10
	}
	if bufSize <= 0 {
		bufSize = workers * 100
	}
	p := &Pool{queue: make(chan func(), bufSize)}
	for range workers {
		p.wg.Add(1)
		go p.run()
	}
	return p
}

func (p *Pool) run() {
	defer p.wg.Done()
	for task := range p.queue {
		if task != nil {
			task()
		}
	}
}

// Submit enqueues a task. Returns false if the queue is full (task is dropped).
func (p *Pool) Submit(task func()) bool {
	select {
	case p.queue <- task:
		return true
	default:
		return false
	}
}

// SubmitCtx enqueues a context-aware task. The context is detached from the
// caller's cancellation so the task continues even after the HTTP request ends.
// Returns false if ctx is already canceled or the queue is full.
func (p *Pool) SubmitCtx(ctx context.Context, task func(context.Context)) bool {
	if ctx.Err() != nil {
		return false
	}
	detached := context.WithoutCancel(ctx)
	return p.Submit(func() { task(detached) })
}

// Stop closes the queue and waits for all in-flight and queued tasks to finish.
func (p *Pool) Stop() {
	p.once.Do(func() { close(p.queue) })
	p.wg.Wait()
}

// QueueSize returns the number of tasks currently waiting in the queue.
func (p *Pool) QueueSize() int { return len(p.queue) }

// Workers returns the number of goroutines in the pool.
func (p *Pool) Workers() int { return cap(p.queue) / 100 }
