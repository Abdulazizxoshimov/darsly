package worker

import (
	"context"
	"sync/atomic"
	"time"
)

// Job is a periodic task executed by a BackgroundWorker.
type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context)
}

// BackgroundWorker runs a set of periodic jobs until stopped.
type BackgroundWorker struct {
	name      string
	jobs      []Job
	stopChan  chan struct{}
	doneChan  chan struct{}
	isRunning atomic.Int32
}

func NewBackgroundWorker(name string, jobs ...Job) *BackgroundWorker {
	return &BackgroundWorker{
		name:     name,
		jobs:     jobs,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Start runs all jobs in separate goroutines. Non-blocking — call in a goroutine.
func (w *BackgroundWorker) Start(ctx context.Context) {
	if !w.isRunning.CompareAndSwap(0, 1) {
		return
	}
	defer func() {
		w.isRunning.Store(0)
		close(w.doneChan)
	}()

	if len(w.jobs) == 0 {
		<-w.stopChan
		return
	}

	tickers := make([]*time.Ticker, len(w.jobs))
	for i, job := range w.jobs {
		tickers[i] = time.NewTicker(job.Interval)
		defer tickers[i].Stop()
	}

	// build a single select via channel fan-in
	type tick struct{ idx int }
	merged := make(chan tick, len(w.jobs))
	for i, t := range tickers {
		go func(idx int, ticker *time.Ticker) {
			for {
				select {
				case <-ticker.C:
					merged <- tick{idx}
				case <-w.stopChan:
					return
				case <-ctx.Done():
					return
				}
			}
		}(i, t)
	}

	for {
		select {
		case t := <-merged:
			job := w.jobs[t.idx]
			jobCtx, cancel := context.WithTimeout(ctx, job.Interval)
			job.Run(jobCtx)
			cancel()

		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop signals the worker to stop and waits until it exits (max 10s).
func (w *BackgroundWorker) Stop(ctx context.Context) {
	if w.isRunning.Load() == 0 {
		return
	}
	close(w.stopChan)
	select {
	case <-w.doneChan:
	case <-time.After(10 * time.Second):
	}
}

func (w *BackgroundWorker) IsRunning() bool { return w.isRunning.Load() == 1 }
