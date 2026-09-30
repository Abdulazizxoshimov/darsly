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
				w.log.Warn(ctx, "email worker: send failed",
					logger.Int("attempt", job.Attempt), logger.SafeString("err", err.Error()))
				retry, next := nextAttempt(job, maxEmailAttempts)
				if !retry {
					// Poison xabar: cheksiz requeue-loop o'rniga tashlaymiz (DLQ yo'q —
					// navbat argumentlarini o'zgartirish mavjud durable navbatni buzardi).
					w.log.Error(ctx, "email worker: max attempts reached, dropping message",
						logger.Int("attempts", job.Attempt))
					_ = d.Nack(false, false)
					continue
				}
				time.Sleep(2 * time.Second) // backoff — SMTP'ni bombalatmaslik uchun
				// Attempt oshirilgan nusxani qayta publish qilib, eskisini ack qilamiz
				// (Nack(requeue) body'ni o'zgartira olmaydi → hisoblagich oshmasdi).
				if perr := w.mq.Publish(ctx, rabbitmq.QueueEmailSend, next); perr != nil {
					_ = d.Nack(false, true) // publish bo'lmadi — xabar yo'qolmasin
					continue
				}
				_ = d.Ack(false)
				continue
			}
			_ = d.Ack(false)
		}
	}
}

// maxEmailAttempts — bitta email uchun jami urinishlar (birinchisi ham hisobda).
const maxEmailAttempts = 5

// nextAttempt keyingi urinish kerakmi va Attempt oshirilgan job'ni qaytaradi.
// Attempt 0/1 dan boshlanadi (QueuedSender 1 yozadi); chegaraga yetgach retry=false.
func nextAttempt(job email.EmailJob, max int) (retry bool, next email.EmailJob) {
	if job.Attempt < 1 {
		job.Attempt = 1
	}
	if job.Attempt >= max {
		return false, job
	}
	job.Attempt++
	return true, job
}
