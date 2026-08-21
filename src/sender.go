package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

// Sender handles sending emails via SMTP.
type Sender struct {
	config *Config
}

// NewSender creates a new email sender with the given config.
func NewSender(config *Config) *Sender {
	return &Sender{config: config}
}

// Send dispatches an email via SMTP.
func (s *Sender) Send(email *Email) error {
	cfg := s.config.SMTP

	// Build the email message with proper MIME headers
	msg := s.buildMessage(email)

	// Connect to the SMTP server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var auth smtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	// For TLS connections, use STARTTLS or direct TLS
	if cfg.UseTLS {
		return s.sendTLS(addr, auth, email, msg)
	}

	// Plain SMTP (no TLS)
	return smtp.SendMail(addr, auth, cfg.From, email.To, []byte(msg))
}

// sendTLS handles TLS/STARTTLS connections.
func (s *Sender) sendTLS(addr string, auth smtp.Auth, email *Email, msg string) error {
	// Try STARTTLS first (port 587/25), fall back to direct TLS (port 465)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("connecting to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.SMTP.Host)
	if err != nil {
		return fmt.Errorf("creating SMTP client: %w", err)
	}
	defer client.Close()

	// Try STARTTLS
	if err = client.StartTLS(&tls.Config{
		ServerName: s.config.SMTP.Host,
	}); err != nil {
		log.Printf("STARTTLS not supported, trying direct TLS: %v", err)
		// Fallback to direct TLS
		tlsConn, err := tls.Dial("tcp", addr, &tls.Config{
			ServerName: s.config.SMTP.Host,
		})
		if err != nil {
			return fmt.Errorf("TLS connection failed: %w", err)
		}
		defer tlsConn.Close()

		client, err = smtp.NewClient(tlsConn, s.config.SMTP.Host)
		if err != nil {
			return fmt.Errorf("creating TLS SMTP client: %w", err)
		}
		defer client.Close()
	}

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err = client.Mail(s.config.SMTP.From); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}

	for _, addr := range email.To {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("SMTP RCPT TO failed for %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	if _, err = w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("writing email data: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("closing data writer: %w", err)
	}

	return client.Quit()
}

// buildMessage constructs a proper MIME email message.
func (s *Sender) buildMessage(email *Email) string {
	var sb strings.Builder

	// Headers
	sb.WriteString(fmt.Sprintf("From: %s\r\n", email.From))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	sb.WriteString("\r\n")

	// Body
	sb.WriteString(email.Body)
	sb.WriteString("\r\n")

	return sb.String()
}
