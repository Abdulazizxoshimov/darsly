package email

import (
	"context"

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
	mq *rabbitmq.Client
}

// NewQueuedSender returns a Sender that enqueues jobs via RabbitMQ.
// Call email.New() first so that sharedTmpl is initialized.
func NewQueuedSender(mq *rabbitmq.Client) Sender {
	return &QueuedSender{mq: mq}
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
	return q.mq.Publish(ctx, rabbitmq.QueueEmailSend, job)
}
