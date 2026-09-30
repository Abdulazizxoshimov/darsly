package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap/zapcore"
)

const (
	maxBatchSize = 100
	flushPeriod  = 2 * time.Second
)

// ── Loki push API types ───────────────────────────────────────────────────────

type pushPayload struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"` // [nanosecond_ts_string, log_line]
}

type entry struct {
	ts   int64
	line string
}

// ── transport: batching + HTTP delivery ──────────────────────────────────────

// queueSize — log yozuvlari navbati sig'imi. To'lsa yangi yozuvlar TASHLANADI
// (Loki sekin/o'chiq bo'lganda handlerlar bloklanmasin — log yo'qolishi request
// stall'idan yaxshiroq).
const queueSize = 2048

type transport struct {
	url      string
	labels   map[string]string
	user     string
	password string
	client   *http.Client

	// in — yozuvlar kanali: add() bloklanmaydi, yagona sender goroutine o'qiydi va
	// HTTP yuboradi. Batch faqat shu goroutine'ga tegishli (qulf kerak emas).
	in      chan entry
	dropped atomic.Int64
	ticker  *time.Ticker
	done    chan struct{}
	stopped chan struct{}
	once    sync.Once
}

func newTransport(lokiURL, user, password string, labels map[string]string) *transport {
	t := &transport{
		url:      lokiURL + "/loki/api/v1/push",
		labels:   labels,
		user:     user,
		password: password,
		client:   &http.Client{Timeout: 5 * time.Second},
		in:       make(chan entry, queueSize),
		ticker:   time.NewTicker(flushPeriod),
		done:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}
	go t.loop()
	return t
}

// add — yozuvni navbatga QO'YADI va darhol qaytadi. Navbat to'la bo'lsa yozuv
// tashlanadi va hisoblagich oshadi (Dropped()).
func (t *transport) add(ts int64, line string) {
	select {
	case t.in <- entry{ts: ts, line: line}:
	default:
		t.dropped.Add(1)
	}
}

// Dropped — navbat to'lganligi sababli tashlangan yozuvlar soni.
func (t *transport) Dropped() int64 { return t.dropped.Load() }

func (t *transport) loop() {
	defer close(t.stopped)
	var batch []entry
	flush := func() {
		if len(batch) > 0 {
			t.send(batch)
			batch = nil
		}
	}
	for {
		select {
		case e := <-t.in:
			batch = append(batch, e)
			if len(batch) >= maxBatchSize {
				flush()
			}
		case <-t.ticker.C:
			flush()
			if n := t.dropped.Swap(0); n > 0 {
				fmt.Fprintf(os.Stderr, "[loki] queue full: %d log entries dropped\n", n)
			}
		case <-t.done:
			// Shutdown: navbatda qolganlarni yig'ib, oxirgi marta yuboramiz.
			for {
				select {
				case e := <-t.in:
					batch = append(batch, e)
					if len(batch) >= maxBatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

func (t *transport) send(batch []entry) {
	values := make([][2]string, len(batch))
	for i, e := range batch {
		values[i] = [2]string{strconv.FormatInt(e.ts, 10), e.line}
	}

	payload := pushPayload{
		Streams: []lokiStream{{Stream: t.labels, Values: values}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, t.url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if t.user != "" {
		req.SetBasicAuth(t.user, t.password)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[loki] push failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "[loki] push error %d: %s\n", resp.StatusCode, string(body))
	}
}

// stop navbatni tugatib, oxirgi flush tugaguncha kutadi (idempotent).
func (t *transport) stop() {
	t.once.Do(func() {
		t.ticker.Stop()
		close(t.done)
	})
	<-t.stopped
}

// ── Core: zapcore.Core implementation ────────────────────────────────────────

var encCfg = zapcore.EncoderConfig{
	TimeKey:        "ts",
	LevelKey:       "level",
	MessageKey:     "msg",
	CallerKey:      "caller",
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
	EncodeDuration: zapcore.StringDurationEncoder,
}

// Core ships structured JSON log lines to Loki via its HTTP push API.
// It batches entries and flushes every 2 seconds or when batch reaches 100 entries.
type Core struct {
	zapcore.LevelEnabler
	tr  *transport
	enc zapcore.Encoder
}

// New creates a Loki Core. Call Stop() on shutdown to flush remaining entries.
// user and password are used for Basic Auth (required by Grafana Cloud; leave empty for self-hosted).
func New(lokiURL, user, password string, labels map[string]string, minLevel zapcore.Level) *Core {
	return &Core{
		LevelEnabler: minLevel,
		tr:           newTransport(lokiURL, user, password, labels),
		enc:          zapcore.NewJSONEncoder(encCfg),
	}
}

func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := &Core{
		LevelEnabler: c.LevelEnabler,
		tr:           c.tr, // shared transport
		enc:          c.enc.Clone(),
	}
	for _, f := range fields {
		f.AddTo(clone.enc)
	}
	return clone
}

func (c *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *Core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}
	c.tr.add(ent.Time.UnixNano(), buf.String())
	buf.Free()
	return nil
}

// Sync — no-op: yuborish fon goroutine'ida; oxirgi flush Stop() da bajariladi.
// (Sync'da sinxron HTTP qilish chaqiruvchini Loki timeout'iga bog'lab qo'yardi.)
func (c *Core) Sync() error { return nil }

// Dropped — navbat to'lib tashlangan yozuvlar soni (diagnostika).
func (c *Core) Dropped() int64 { return c.tr.Dropped() }

// Stop flushes remaining entries and stops the background goroutine.
func (c *Core) Stop() {
	c.tr.stop()
}
