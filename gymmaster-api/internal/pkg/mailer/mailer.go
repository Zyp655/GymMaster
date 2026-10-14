// Package mailer gửi email qua SMTP (gomail). Dev dùng Mailpit (docker compose, giao diện http://localhost:8025).
//
//	m := mailer.New(cfg.Mail)
//	html, _ := mailer.Render(mailer.TplSubscriptionExpiring, mailer.ExpiringData{...})
//	m.SendAsync(mailer.Message{To: []string{"a@b.vn"}, Subject: "...", HTML: html})
//
// SendAsync gửi nền, thử lại tối đa 3 lần (2s, 4s); lỗi chỉ ghi log – không làm hỏng nghiệp vụ.
package mailer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gopkg.in/gomail.v2"

	"gymmaster-api/internal/config"
)

type Message struct {
	To      []string
	Subject string
	HTML    string
}

type Mailer interface {
	// Send gửi đồng bộ (một lần, không retry).
	Send(ctx context.Context, msg Message) error
	// SendAsync gửi nền có retry nhẹ; lỗi cuối cùng chỉ được log.
	SendAsync(msg Message)
	// Wait chờ các email đang gửi nền (dùng khi tắt server / trong test).
	Wait()
}

type smtpMailer struct {
	cfg     config.MailCfg
	retries int
	backoff time.Duration
	wg      sync.WaitGroup
	sem     chan struct{} // giới hạn số kết nối SMTP đồng thời
}

func New(cfg config.MailCfg) Mailer {
	return &smtpMailer{cfg: cfg, retries: 3, backoff: 2 * time.Second, sem: make(chan struct{}, 4)}
}

var ErrNoRecipient = errors.New("mailer: thiếu người nhận")

func (m *smtpMailer) Send(ctx context.Context, msg Message) error {
	to := make([]string, 0, len(msg.To))
	for _, t := range msg.To {
		if t = strings.TrimSpace(t); t != "" {
			to = append(to, t)
		}
	}
	if len(to) == 0 {
		return ErrNoRecipient
	}
	gm := gomail.NewMessage(gomail.SetCharset("UTF-8"))
	gm.SetHeader("From", m.cfg.From)
	gm.SetHeader("To", to...)
	gm.SetHeader("Subject", msg.Subject)
	gm.SetBody("text/html", msg.HTML)

	d := gomail.NewDialer(m.cfg.Host, m.cfg.Port, m.cfg.User, m.cfg.Pass)
	// Mailpit/SMTP nội bộ không có TLS: gomail chỉ STARTTLS khi server hỗ trợ.
	done := make(chan error, 1)
	go func() { done <- d.DialAndSend(gm) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *smtpMailer) SendAsync(msg Message) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.sem <- struct{}{}
		defer func() { <-m.sem }()
		ctx := context.Background()
		var err error
		wait := m.backoff
		for attempt := 1; attempt <= m.retries; attempt++ {
			sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			err = m.Send(sendCtx, msg)
			cancel()
			if err == nil || errors.Is(err, ErrNoRecipient) {
				break
			}
			if attempt < m.retries {
				time.Sleep(wait)
				wait *= 2
			}
		}
		if err != nil {
			g.Log().Warningf(ctx, "mailer: gửi email %q tới %v thất bại: %v", msg.Subject, msg.To, err)
			return
		}
		g.Log().Debugf(ctx, "mailer: đã gửi %q tới %v", msg.Subject, msg.To)
	}()
}

func (m *smtpMailer) Wait() { m.wg.Wait() }

var (
	defaultOnce sync.Once
	defaultM    Mailer
)

// Default trả mailer dùng chung của tiến trình (tạo một lần từ cfg đầu tiên).
func Default(cfg config.MailCfg) Mailer {
	defaultOnce.Do(func() { defaultM = New(cfg) })
	return defaultM
}
