package loki

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

// Loki sekin bo'lsa ham Write bloklanmaydi, Stop esa navbatni yetkazadi.
func TestCore_WriteNeverBlocksAndStopFlushes(t *testing.T) {
	var got atomic.Int64
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // sekin Loki
		b, _ := io.ReadAll(r.Body)
		got.Add(int64(strings.Count(string(b), `\"level\"`)))
	}))
	defer srv.Close()

	c := New(srv.URL, "", "", map[string]string{"app": "t"}, zapcore.WarnLevel)
	start := time.Now()
	for i := 0; i < 500; i++ {
		_ = c.Write(zapcore.Entry{Level: zapcore.WarnLevel, Message: "m", Time: time.Now()}, nil)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("Write bloklandi: %v", d)
	}
	close(release)
	c.Stop()
	c.Stop() // idempotent
	if got.Load() != 500 {
		t.Fatalf("Stop hamma yozuvni yetkazishi kerak, got %d", got.Load())
	}
}

func TestCore_DropsWhenQueueFull(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)

	c := New(srv.URL, "", "", nil, zapcore.WarnLevel)
	for i := 0; i < queueSize+maxBatchSize+500; i++ {
		_ = c.Write(zapcore.Entry{Level: zapcore.WarnLevel, Message: "m", Time: time.Now()}, nil)
	}
	if c.Dropped() == 0 {
		t.Fatal("to'lgan navbatda yozuvlar tashlanishi va sanalishi kerak")
	}
}
