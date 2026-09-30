package email

import (
	"context"
	"fmt"

	"github.com/zoom/darsly/internal/infrastructure/rabbitmq"
)

// EmailJob is the payload published to QueueEmailSend for async delivery.
type EmailJob struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Attempt int      `json:"attempt"`
}

// QueuedSender wraps a real Sender but enqueues emails via RabbitMQ for
// async delivery with retry instead of sending inline.
type QueuedSender struct {
	mq       publisher
	fallback Sender // navbatga qo'yib bo'lmasa to'g'ridan-to'g'ri yuboruvchi (nil bo'lishi mumkin)
}

// publisher — *rabbitmq.Client'ning QueuedSender ishlatadigan qismi (test uchun ajratilgan).
type publisher interface {
	Publish(ctx context.Context, queue string, payload any) error
}

// NewQueuedSender returns a Sender that enqueues jobs via RabbitMQ.
// Call email.New() first so that sharedTmpl is initialized.
//
// fallback — broker uzilgan/nack qilgan paytda email YO'QOLMASLIGI uchun to'g'ridan-to'g'ri
// (SMTP) yuboruvchi. nil bo'lsa Publish xatosi qaytariladi.
func NewQueuedSender(mq *rabbitmq.Client, fallback Sender) Sender {
	return &QueuedSender{mq: mq, fallback: fallback}
}

// Send renders the named template and enqueues the resulting HTML body.
func (q *QueuedSender) Send(ctx context.Context, to []string, subject, templateName string, data any) error {
	body, err := Render(templateName, data)
	if err != nil {
		return err
	}
	return q.SendRaw(ctx, to, subject, body)
}

// SendRaw enqueues a pre-built HTML body to RabbitMQ.
func (q *QueuedSender) SendRaw(ctx context.Context, to []string, subject, body string) error {
	job := EmailJob{To: to, Subject: subject, Body: body, Attempt: 1}
	err := q.mq.Publish(ctx, rabbitmq.QueueEmailSend, job)
	if err != nil && q.fallback != nil {
		// Navbat mavjud emas — xatni yo'qotmasdan to'g'ridan-to'g'ri yuboramiz.
		if ferr := q.fallback.SendRaw(ctx, to, subject, body); ferr != nil {
			return fmt.Errorf("email: queue failed (%v), direct send failed: %w", err, ferr)
		}
		return nil
	}
	return err
}
