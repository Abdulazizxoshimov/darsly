package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/zoom/darsly/internal/pkg/logger"
)

const (
	QueueEmailSend = "email_send"
)

// confirmTimeout — broker Publish'ni tasdiqlashini kutish chegarasi.
const confirmTimeout = 5 * time.Second

// declaredQueues — startda va har reconnect'da e'lon qilinadigan navbatlar.
var declaredQueues = []string{QueueEmailSend}

var errNotConnected = errors.New("rabbitmq: not connected")

// ErrChannelClosed — Consume delivery kanali yopilganda supervisor qayta ulanishi uchun.
var ErrChannelClosed = errors.New("rabbitmq: channel closed")

// Client — RabbitMQ connection/channel, avtomatik qayta ulanish bilan.
// Tarmoq uzilsa (past-internet hududlar uchun kritik) fon-monitor backoff bilan
// qayta ulanadi; Publish/Consume joriy (yangilangan) kanalni ishlatadi.
// prefetch — consumer bir vaqtda nechta ack qilinmagan xabarni oladi (Qos).
const prefetch = 20

type Client struct {
	url  string
	log  logger.Logger
	mu   sync.RWMutex
	conn *amqp.Connection
	ch   *amqp.Channel // consume kanali (Qos bilan)
	// pubCh — Publish uchun ALOHIDA kanal. amqp091 kanali concurrent Publish uchun
	// xavfsiz emas; pubMu bilan seriyalanadi va consume kanalidan ajratilgan.
	pubCh  *amqp.Channel
	pubMu  sync.Mutex
	closed atomic.Bool
	done   chan struct{}
}

// New — RabbitMQ'ga ulanadi va qayta-ulanish monitorini ishga tushiradi.
func New(url string, log logger.Logger) (*Client, error) {
	c := &Client{url: url, log: log, done: make(chan struct{})}
	if err := c.connect(); err != nil {
		return nil, err
	}
	go c.monitor()
	return c, nil
}

func (c *Client) connect() error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("rabbitmq.Dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("rabbitmq.Channel: %w", err)
	}
	// Consumer prefetch — bitta serial worker o'rniga bir vaqtda ≤prefetch xabar
	// ishlanadi (worker o'zi goroutine bilan parallellashtira oladi).
	if err := ch.Qos(prefetch, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("rabbitmq.Qos: %w", err)
	}
	pubCh, err := conn.Channel()
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("rabbitmq.PublishChannel: %w", err)
	}
	// Publisher confirms: broker xabarni qabul qilganini tasdiqlamaguncha Publish
	// muvaffaqiyatli hisoblanmaydi (aks holda uzilgan/to'la broker xabarni jimgina yutardi).
	if err := pubCh.Confirm(false); err != nil {
		pubCh.Close()
		ch.Close()
		conn.Close()
		return fmt.Errorf("rabbitmq.Confirm: %w", err)
	}
	for _, q := range declaredQueues {
		if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
			pubCh.Close()
			ch.Close()
			conn.Close()
			return fmt.Errorf("rabbitmq.QueueDeclare %q: %w", q, err)
		}
	}
	c.mu.Lock()
	c.conn, c.ch, c.pubCh = conn, ch, pubCh
	c.mu.Unlock()
	return nil
}

// monitor connection uzilishini kuzatadi va backoff bilan qayta ulanadi.
func (c *Client) monitor() {
	for {
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()
		if conn == nil {
			return
		}
		closeErr := <-conn.NotifyClose(make(chan *amqp.Error, 1))
		if c.closed.Load() {
			return // ataylab yopildi
		}
		if closeErr != nil {
			c.log.Warn(context.Background(), "rabbitmq connection lost, reconnecting", logger.SafeString("err", closeErr.Error()))
		}
		backoff := time.Second
		for !c.closed.Load() {
			select {
			case <-c.done:
				return
			case <-time.After(backoff):
			}
			if err := c.connect(); err != nil {
				c.log.Warn(context.Background(), "rabbitmq reconnect failed", logger.SafeString("err", err.Error()))
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			c.log.Info(context.Background(), "rabbitmq reconnected")
			break
		}
	}
}

func (c *Client) channel() *amqp.Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ch
}

// Close — ulanishni yopadi va monitorni to'xtatadi.
func (c *Client) Close() {
	c.closed.Store(true)
	close(c.done)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pubCh != nil {
		_ = c.pubCh.Close()
	}
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// DeclareQueue — queue'ni e'lon qiladi (idempotent).
func (c *Client) DeclareQueue(name string) error {
	ch := c.channel()
	if ch == nil {
		return errNotConnected
	}
	_, err := ch.QueueDeclare(name, true, false, false, false, nil)
	return err
}

// Publish — queue'ga message yuboradi (joriy kanal orqali).
func (c *Client) Publish(ctx context.Context, queue string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("rabbitmq.Publish marshal: %w", err)
	}
	// Publish'lar seriyalanadi (bir amqp kanali concurrent Publish uchun xavfsiz emas).
	c.pubMu.Lock()
	defer c.pubMu.Unlock()
	c.mu.RLock()
	ch := c.pubCh
	c.mu.RUnlock()
	if ch == nil {
		return errNotConnected
	}
	conf, err := ch.PublishWithDeferredConfirmWithContext(ctx,
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return err
	}
	// Broker ack'ini kutamiz (pubMu ushlab turilgan — confirm'lar tartibi saqlanadi).
	cctx, cancel := context.WithTimeout(ctx, confirmTimeout)
	defer cancel()
	acked, err := conf.WaitContext(cctx)
	if err != nil {
		return fmt.Errorf("rabbitmq.Publish confirm: %w", err)
	}
	if !acked {
		return errors.New("rabbitmq.Publish: broker nack")
	}
	return nil
}

// Consume — queue'dan xabarlarni o'qiydi (joriy kanal orqali). Kanal yopilsa
// qaytgan delivery kanali ham yopiladi — chaqiruvchi (worker) qayta Consume qiladi.
func (c *Client) Consume(queue string) (<-chan amqp.Delivery, error) {
	ch := c.channel()
	if ch == nil {
		return nil, errNotConnected
	}
	return ch.Consume(queue, "", false, false, false, false, nil)
}
