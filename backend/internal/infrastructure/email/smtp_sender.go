package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"

	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/ports"
	applogger "github.com/rekall/backend/pkg/logger"
)

// SMTPSender delivers transactional emails via SMTP.
//
// Local development: point at Mailpit (host=localhost/mailpit, port=1025, no auth, TLS=false).
// Production: point at any SMTP relay (e.g. SendGrid SMTP API: host=smtp.sendgrid.net, port=587, TLS=true).
type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
	useTLS   bool
	logger   *zap.Logger
}

// NewSMTPSender creates a configured SMTPSender.
func NewSMTPSender(host string, port int, username, password, from string, useTLS bool, log *zap.Logger) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		useTLS:   useTLS,
		logger:   applogger.WithComponent(log, "smtp_sender"),
	}
}

// Send delivers a single email. It satisfies ports.EmailSender.
func (s *SMTPSender) Send(_ context.Context, msg ports.EmailMessage) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n",
		s.from, msg.To, msg.Subject,
	)
	body := []byte(headers + msg.Body)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	var err error
	if s.useTLS {
		err = s.sendTLS(addr, auth, body, msg.To)
	} else {
		err = smtp.SendMail(addr, auth, s.from, []string{msg.To}, body)
	}

	if err != nil {
		s.logger.Error("failed to send email",
			zap.String("to", msg.To),
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
		return fmt.Errorf("smtp_sender: %w", err)
	}

	s.logger.Debug("email sent",
		zap.String("to", msg.To),
		zap.String("subject", msg.Subject),
	)
	return nil
}

// sendTLS dials with an explicit TLS handshake (used for port 465 / SMTPS).
// Most modern relays (SendGrid port 587) use STARTTLS via smtp.SendMail, but
// this path handles explicit TLS for providers that require it.
func (s *SMTPSender) sendTLS(addr string, auth smtp.Auth, body []byte, to string) error {
	tlsCfg := &tls.Config{ServerName: s.host} //nolint:gosec // standard TLS defaults are fine here

	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		// Fall back to STARTTLS via plain dial
		plainConn, dialErr := net.Dial("tcp", addr)
		if dialErr != nil {
			return fmt.Errorf("dial %s: %w", addr, dialErr)
		}
		client, clientErr := smtp.NewClient(plainConn, s.host)
		if clientErr != nil {
			return fmt.Errorf("smtp client: %w", clientErr)
		}
		defer client.Close() //nolint:errcheck
		if err2 := client.StartTLS(tlsCfg); err2 != nil {
			return fmt.Errorf("starttls: %w", err2)
		}
		if auth != nil {
			if err2 := client.Auth(auth); err2 != nil {
				return fmt.Errorf("smtp auth: %w", err2)
			}
		}
		return sendViaClient(client, s.from, to, body)
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp client (tls): %w", err)
	}
	defer client.Close() //nolint:errcheck
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth (tls): %w", err)
		}
	}
	return sendViaClient(client, s.from, to, body)
}

func sendViaClient(c *smtp.Client, from, to string, body []byte) error {
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(body); err != nil {
		return err
	}
	return w.Close()
}
