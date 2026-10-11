package email

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"
)

func TestLogSenderWritesTheMessage(t *testing.T) {
	var out bytes.Buffer
	sender := LogSender{Logger: log.New(&out, "", 0)}

	err := sender.Send(context.Background(), Message{
		To:      "dana@example.org",
		Subject: "Confirm your email",
		Text:    "Open this link:\nhttps://example.org/confirm",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"dana@example.org", "Confirm your email", "https://example.org/confirm"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("log output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestMessagesAreRefused(t *testing.T) {
	tests := []struct {
		name string
		m    Message
	}{
		{"recipient is not an address", Message{To: "dana", Subject: "Hello", Text: "x"}},
		{"subject carries a line break", Message{To: "dana@example.org", Subject: "Hello\r\nBcc: other@example.org", Text: "x"}},
		{"subject is empty", Message{To: "dana@example.org", Text: "x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := (LogSender{Logger: log.New(&bytes.Buffer{}, "", 0)}).Send(context.Background(), tt.m); err == nil {
				t.Error("expected the message to be refused")
			}
			// The SMTP adapter refuses before it connects anywhere.
			if err := (SMTPSender{Host: "localhost", Port: 1, From: "events@example.org"}).Send(context.Background(), tt.m); err == nil {
				t.Error("expected the SMTP adapter to refuse the message")
			}
		})
	}
}

func TestCompose(t *testing.T) {
	when := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	got := string(compose("PhoenixEvent <events@example.org>", Message{
		To:      "dana@example.org",
		Subject: "Réinitialiser",
		Text:    "Line one\nLine two",
	}, when))

	for _, want := range []string{
		"From: PhoenixEvent <events@example.org>\r\n",
		"To: dana@example.org\r\n",
		"Subject: =?utf-8?q?R=C3=A9initialiser?=\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
		"\r\n\r\nLine one\r\nLine two\r\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("composed message lacks %q:\n%s", want, got)
		}
	}
}
