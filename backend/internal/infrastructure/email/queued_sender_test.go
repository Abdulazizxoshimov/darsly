package email

import (
	"context"
	"errors"
	"testing"
)

type fakePub struct{ err error }

func (f fakePub) Publish(context.Context, string, any) error { return f.err }

type recSender struct{ calls int }

func (r *recSender) Send(context.Context, []string, string, string, any) error { return nil }
func (r *recSender) SendRaw(context.Context, []string, string, string) error {
	r.calls++
	return nil
}

func TestQueuedSender_FallsBackToDirectOnPublishError(t *testing.T) {
	fb := &recSender{}
	q := &QueuedSender{mq: fakePub{err: errors.New("broker down")}, fallback: fb}
	if err := q.SendRaw(context.Background(), []string{"a@b.c"}, "s", "b"); err != nil {
		t.Fatal(err)
	}
	if fb.calls != 1 {
		t.Fatalf("fallback chaqirilishi kerak, calls=%d", fb.calls)
	}
}

func TestQueuedSender_NoFallbackWhenPublishOK(t *testing.T) {
	fb := &recSender{}
	q := &QueuedSender{mq: fakePub{}, fallback: fb}
	_ = q.SendRaw(context.Background(), nil, "s", "b")
	if fb.calls != 0 {
		t.Fatal("publish OK bo'lsa fallback chaqirilmasligi kerak")
	}
}

func TestQueuedSender_ErrorWithoutFallback(t *testing.T) {
	q := &QueuedSender{mq: fakePub{err: errors.New("x")}}
	if err := q.SendRaw(context.Background(), nil, "s", "b"); err == nil {
		t.Fatal("fallback yo'q — xato qaytishi kerak")
	}
}
