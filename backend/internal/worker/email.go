package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zoom/darsly/internal/infrastructure/email"
	"github.com/zoom/darsly/internal/infrastructure/rabbitmq"
	"github.com/zoom/darsly/internal/pkg/logger"
)

// EmailWorker RabbitMQ navbatidan email joblarini o'qib, haqiqiy SMTP orqali yuboradi.
// Bu email yuborishni request oqimidan ajratadi (SMTP kutish latency'ni buzmaydi).
type EmailWorker struct {
	mq     *rabbitmq.Client
	sender email.Sender // to'g'ridan-to'g'ri (SMTP) sender
	log    logger.Logger
}

func NewEmailWorker(mq *rabbitmq.Client, sender email.Sender, log logger.Logger) *EmailWorker {
	return &EmailWorker{mq: mq, sender: sender, log: log}
}

// Run supervisor: RabbitMQ kanali uzilsa (reconnect) backoff bilan qayta Consume qiladi.
func (w *EmailWorker) Run(ctx context.Context) {
	w.log.Info(ctx, "email worker started")
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if err := w.consumeLoop(ctx); err != nil && ctx.Err() == nil {
			w.log.Warn(ctx, "email worker: consume loop ended, retrying", logger.SafeString("err", err.Error()))
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 15*time.Second {
				backoff *= 2
			}
			continue
		}
		return // ctx bekor qilindi
	}
}

// consumeLoop bitta Consume sessiyasini yuritadi; kanal yopilsa (ok=false) qaytadi.
func (w *EmailWorker) consumeLoop(ctx context.Context) error {
	deliveries, err := w.mq.Consume(rabbitmq.QueueEmailSend)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return rabbitmq.ErrChannelClosed // kanal yopildi → supervisor qayta ulaydi
			}
			var job email.EmailJob
			if err := json.Unmarshal(d.Body, &job); err != nil {
				_ = d.Nack(false, false) // yaroqsiz — qayta navbatga solmaymiz
				continue
			}
			if err := w.sender.SendRaw(ctx, job.To, job.Subject, job.Body); err != nil {
				w.log.Warn(ctx, "email worker: send failed", logger.SafeString("err", err.Error()))
				time.Sleep(2 * time.Second) // backoff — SMTP'ni bombalatmaslik uchun
				_ = d.Nack(false, true)     // qayta urinish uchun navbatga qaytaramiz
				continue
			}
			_ = d.Ack(false)
		}
	}
}
