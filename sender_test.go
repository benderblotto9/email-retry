package main

import (
	"strings"
	"testing"
)

// ============================================================
// Sender Tests
// ============================================================

func TestBuildMessage(t *testing.T) {
	cfg := &Config{}
	sender := NewSender(cfg)

	email := &Email{
		To:      []string{"alice@example.com", "bob@example.com"},
		From:    "sender@example.com",
		Subject: "Test Subject",
		Body:    "<html><body>Hello!</body></html>",
	}

	msg := sender.buildMessage(email)

	// Verify headers
	if !strings.Contains(msg, "From: sender@example.com\r\n") {
		t.Error("message missing From header")
	}
	if !strings.Contains(msg, "To: alice@example.com, bob@example.com\r\n") {
		t.Error("message missing or wrong To header")
	}
	if !strings.Contains(msg, "Subject: Test Subject\r\n") {
		t.Error("message missing Subject header")
	}
	if !strings.Contains(msg, "MIME-Version: 1.0\r\n") {
		t.Error("message missing MIME-Version header")
	}
	if !strings.Contains(msg, "Content-Type: text/html; charset=UTF-8\r\n") {
		t.Error("message missing Content-Type header")
	}

	// Verify body
	if !strings.Contains(msg, "<html><body>Hello!</body></html>") {
		t.Error("message missing email body")
	}

	// Verify proper line endings
	if !strings.Contains(msg, "\r\n\r\n") {
		t.Error("message missing blank line separator between headers and body")
	}
}

func TestBuildMessage_SingleRecipient(t *testing.T) {
	cfg := &Config{}
	sender := NewSender(cfg)

	email := &Email{
		To:      []string{"only@example.com"},
		From:    "me@example.com",
		Subject: "One Person",
		Body:    "plain text body",
	}

	msg := sender.buildMessage(email)

	if !strings.Contains(msg, "To: only@example.com\r\n") {
		t.Error("single recipient To header incorrect")
	}
}

func TestBuildMessage_SpecialCharacters(t *testing.T) {
	cfg := &Config{}
	sender := NewSender(cfg)

	email := &Email{
		To:      []string{"user@example.com"},
		From:    "system@example.com",
		Subject: "Subject with \"quotes\" & <brackets>",
		Body:    "<html><body>&amp; entity</body></html>",
	}

	msg := sender.buildMessage(email)

	if !strings.Contains(msg, "Subject: Subject with \"quotes\" & <brackets>\r\n") {
		t.Error("Subject with special characters not preserved")
	}
	if !strings.Contains(msg, "&amp; entity") {
		t.Error("HTML entity in body not preserved")
	}
}
