package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/zoom/darsly/internal/pkg/config"
	"github.com/zoom/darsly/internal/pkg/logger"
)

type Sender interface {
	Send(ctx context.Context, to []string, subject, templateName string, data any) error
	SendRaw(ctx context.Context, to []string, subject, body string) error
}

// sharedTmpl is the package-level compiled template, loaded once.
var sharedTmpl *template.Template

// Render executes a named template and returns the HTML body.
func Render(templateName string, data any) (string, error) {
	if sharedTmpl == nil {
		return "", fmt.Errorf("email: templates not initialized")
	}
	var buf bytes.Buffer
	if err := sharedTmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("email: template %q: %w", templateName, err)
	}
	return buf.String(), nil
}

type emailSender struct {
	cfg config.EmailConfig
	log logger.Logger
}

type nopSender struct{}

func (nopSender) Send(_ context.Context, _ []string, _, _ string, _ any) error { return nil }
func (nopSender) SendRaw(_ context.Context, _ []string, _, _ string) error     { return nil }

// NewNopSender returns a Sender that silently discards all emails.
// Used when EMAIL_ENABLED=false.
func NewNopSender() Sender { return nopSender{} }

func New(cfg config.EmailConfig, log logger.Logger) (Sender, error) {
	if sharedTmpl == nil {
		tmpl, err := loadTemplates()
		if err != nil {
			return nil, fmt.Errorf("email: load templates: %w", err)
		}
		sharedTmpl = tmpl
	}
	return &emailSender{cfg: cfg, log: log}, nil
}

func (s *emailSender) Send(ctx context.Context, to []string, subject, templateName string, data any) error {
	body, err := Render(templateName, data)
	if err != nil {
		return err
	}
	return s.SendRaw(ctx, to, subject, body)
}

func (s *emailSender) SendRaw(ctx context.Context, to []string, subject, body string) error {
	authUser := s.cfg.Username
	if authUser == "" || !strings.Contains(authUser, "@") {
		authUser = s.cfg.From
	}
	auth := smtp.PlainAuth("", authUser, s.cfg.Password, s.cfg.Host)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	displayName := s.cfg.Username
	if strings.Contains(displayName, "@") || displayName == "" {
		displayName = "Jonly"
	}
	fromHeader := fmt.Sprintf("%s <%s>", displayName, s.cfg.From)

	msg := []byte(
		"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"From: " + fromHeader + "\r\n" +
			"To: " + strings.Join(to, ", ") + "\r\n" +
			"Subject: " + subject + "\r\n\r\n" +
			body,
	)

	if err := sendMailTimeout(ctx, addr, s.cfg.Host, auth, s.cfg.From, to, msg); err != nil {
		s.log.Error(ctx, "email send failed",
			logger.String("to", strings.Join(to, ",")),
			logger.Error(err),
		)
		return fmt.Errorf("email: send: %w", err)
	}
	return nil
}

// loadTemplates parses all built-in HTML email templates.
// Each template is self-contained to avoid shared "content" block conflicts.
func loadTemplates() (*template.Template, error) {
	tmpl := template.New("").Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	})

	const inviteEmail = `{{define "invite"}}<!DOCTYPE html>` +
		`<html><body style="font-family:Arial,sans-serif;max-width:600px;margin:auto;padding:20px">` +
		`<h2>Taklifnoma</h2>` +
		`<p>Siz <strong>{{.ProjectName}}</strong> loyihasiga taklif qilindingiz.</p>` +
		`<p><a href="{{.InviteURL}}" style="background:#0052cc;color:#fff;padding:10px 20px;border-radius:4px;text-decoration:none">Qabul qilish</a></p>` +
		`<p style="color:#999;font-size:12px">Havola {{.ExpiresIn}} muddatgacha amal qiladi.</p>` +
		`<p style="color:#999;font-size:12px;margin-top:40px">Jonly · noreply@jonly.uz</p></body></html>{{end}}`

	const resetEmail = `{{define "password_reset"}}<!DOCTYPE html>` +
		`<html><body style="font-family:Arial,sans-serif;max-width:600px;margin:auto;padding:20px">` +
		`<h2>Parolni tiklash</h2>` +
		`<p>Parolingizni tiklash uchun quyidagi havolaga bosing:</p>` +
		`<p><a href="{{.ResetURL}}" style="background:#0052cc;color:#fff;padding:10px 20px;border-radius:4px;text-decoration:none">Parolni tiklash</a></p>` +
		`<p style="color:#999;font-size:12px">Agar siz so'rov yubormasangiz, bu xatni e'tiborsiz qoldiring.</p>` +
		`<p style="color:#999;font-size:12px;margin-top:40px">Jonly · noreply@jonly.uz</p></body></html>{{end}}`

	const notifyEmail = `{{define "notification"}}<!DOCTYPE html>` +
		`<html><body style="font-family:Arial,sans-serif;max-width:600px;margin:auto;padding:20px">` +
		`<h2>{{.Title}}</h2>` +
		`<p>{{.Body | safeHTML}}</p>` +
		`{{if .ActionURL}}<p><a href="{{.ActionURL}}" style="background:#0052cc;color:#fff;padding:10px 20px;border-radius:4px;text-decoration:none">Ko'rish</a></p>{{end}}` +
		`<p style="color:#999;font-size:12px;margin-top:40px">Jonly · noreply@jonly.uz</p></body></html>{{end}}`

	for _, t := range []string{inviteEmail, resetEmail, notifyEmail} {
		if _, err := tmpl.Parse(t); err != nil {
			return nil, err
		}
	}
	return tmpl, nil
}

// SMTP vaqt chegaralari (var — testda qisqartiriladi).
var (
	smtpDialTimeout = 10 * time.Second // TCP ulanish
	smtpTotalBudget = 30 * time.Second // butun sessiya (handshake + DATA)
)

// sendMailTimeout — smtp.SendMail bilan bir xil oqim (STARTTLS, AUTH, MAIL/RCPT/DATA),
// lekin ulanish va butun sessiya vaqt bilan chegaralangan. smtp.SendMail timeout'siz:
// javob bermayotgan SMTP server email worker'ini abadiy osib qo'yardi.
func sendMailTimeout(ctx context.Context, addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	d := net.Dialer{Timeout: smtpDialTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(smtpTotalBudget)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	_ = conn.SetDeadline(deadline)

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
