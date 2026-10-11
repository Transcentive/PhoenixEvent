// Package email sends PhoenixEvent's messages: email confirmation, password
// reset and invitation (spec sections 8 and 8A). The application depends on
// the Sender interface only, so the delivery service can change without
// touching the code that writes the messages.
package email

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Message is one plain-text email to one person.
type Message struct {
	To      string
	Subject string
	Text    string
}

// Sender delivers a message.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// LogSender writes each message to the server log in place of sending it. It
// is the adapter for a machine with no SMTP server configured, such as a
// development Mac, where the link in the message is read from the log.
type LogSender struct {
	Logger *log.Logger
}

func (s LogSender) Send(_ context.Context, m Message) error {
	if err := m.validate(); err != nil {
		return err
	}
	logger := s.Logger
	if logger == nil {
		logger = log.Default()
	}
	logger.Printf("email (not sent, no SMTP server configured)\n  to: %s\n  subject: %s\n%s", m.To, m.Subject, indent(m.Text))
	return nil
}

// SMTPSender sends through an SMTP server, which for PhoenixEvent is
// Rackspace (spec 2A.2). The connection is upgraded with STARTTLS whenever the
// server offers it. An empty Username sends without signing in, which suits a
// local mail catcher.
type SMTPSender struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s SMTPSender) Send(_ context.Context, m Message) error {
	if err := m.validate(); err != nil {
		return err
	}
	from, err := mail.ParseAddress(s.From)
	if err != nil {
		return fmt.Errorf("EMAIL_FROM is not an email address: %w", err)
	}

	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	if err := smtp.SendMail(addr, auth, from.Address, []string{m.To}, compose(s.From, m, time.Now())); err != nil {
		return fmt.Errorf("failed to send email to %s: %w", m.To, err)
	}
	return nil
}

func (m Message) validate() error {
	if _, err := mail.ParseAddress(m.To); err != nil {
		return fmt.Errorf("recipient %q is not an email address: %w", m.To, err)
	}
	// A line break in a header would let the text after it be read as a new
	// header, so a subject carrying one is refused outright.
	if strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("subject contains a line break")
	}
	if m.Subject == "" {
		return errors.New("subject is empty")
	}
	return nil
}

func compose(from string, m Message, now time.Time) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + m.To + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", m.Subject) + "\r\n")
	b.WriteString("Date: " + now.Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(strings.ReplaceAll(m.Text, "\r\n", "\n"), "\n", "\r\n"))
	b.WriteString("\r\n")
	return []byte(b.String())
}

func indent(text string) string {
	return "  " + strings.ReplaceAll(strings.TrimRight(text, "\n"), "\n", "\n  ")
}
