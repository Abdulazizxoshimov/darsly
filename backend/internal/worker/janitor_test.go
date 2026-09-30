package worker

import (
	"context"
	"testing"
	"time"

	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/testutil"
)

type fakeJanitor struct {
	repository.JanitorRepository
	wrCalls, notifCalls int
	batches             []int64
}

func (f *fakeJanitor) DeleteDecidedWaitingRequests(_ context.Context, _ time.Time, _ int) (int64, error) {
	f.wrCalls++
	if len(f.batches) > 0 {
		n := f.batches[0]
		f.batches = f.batches[1:]
		return n, nil
	}
	return 0, nil
}

func (f *fakeJanitor) DeleteOldReadNotifications(_ context.Context, _ time.Time, _ int) (int64, error) {
	f.notifCalls++
	return 0, nil
}

func TestJanitorTick_DrainsBatches(t *testing.T) {
	f := &fakeJanitor{batches: []int64{janitorBatch, janitorBatch, 10}}
	w := NewJanitorWorker(testutil.NewFakeAuthRepo(), f, testutil.NewLogger())
	w.tick(context.Background())
	if f.wrCalls != 3 {
		t.Fatalf("waiting batches = %d, want 3", f.wrCalls)
	}
	if f.notifCalls != 1 {
		t.Fatalf("notif calls = %d, want 1", f.notifCalls)
	}
}
