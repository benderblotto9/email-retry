package main

import (
	"strings"
	"testing"
)

// ============================================================
// Parser Tests
// ============================================================

func TestNewParser(t *testing.T) {
	// Valid patterns
	p, err := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)
	if err != nil {
		t.Fatalf("NewParser() returned error: %v", err)
	}
	if p == nil {
		t.Fatal("NewParser() returned nil parser")
	}

	// Invalid start pattern
	_, err = NewParser(`[invalid`, `(?i)^END$`)
	if err == nil {
		t.Fatal("NewParser() should return error for invalid start pattern")
	}

	// Invalid end pattern
	_, err = NewParser(`(?i)^To:\s+`, `[invalid`)
	if err == nil {
		t.Fatal("NewParser() should return error for invalid end pattern")
	}
}

func TestParseAddressList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single address",
			input:    "alice@example.com",
			expected: []string{"alice@example.com"},
		},
		{
			name:     "multiple addresses",
			input:    "alice@example.com, bob@example.com, charlie@example.com",
			expected: []string{"alice@example.com", "bob@example.com", "charlie@example.com"},
		},
		{
			name:     "addresses with angle brackets",
			input:    "<alice@example.com>, <bob@example.com>",
			expected: []string{"alice@example.com", "bob@example.com"},
		},
		{
			name:     "mixed bracket styles",
			input:    "<alice@example.com>, bob@example.com",
			expected: []string{"alice@example.com", "bob@example.com"},
		},
		{
			name:     "extra whitespace",
			input:    "  alice@example.com  ,   bob@example.com  ",
			expected: []string{"alice@example.com", "bob@example.com"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "just commas",
			input:    ",,,",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAddressList(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("parseAddressList(%q) returned %d addresses, want %d: %v",
					tt.input, len(result), len(tt.expected), result)
			}
			for i, addr := range result {
				if addr != tt.expected[i] {
					t.Errorf("parseAddressList(%q)[%d] = %q, want %q",
						tt.input, i, addr, tt.expected[i])
				}
			}
		})
	}
}

func TestFindEmails_SingleEmail(t *testing.T) {
	p, _ := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)

	logContent := `2026-08-20 22:00:00 INFO: System started
To: alice@example.com
From: sender@example.com
Subject: Test Email
Content-Type: text/html

<html>
<body>
<h1>Hello!</h1>
<p>This is a test email.</p>
</body>
</html>
--- END EMAIL ---
2026-08-20 22:00:01 INFO: Done`

	emails := p.FindEmails(logContent)
	if len(emails) != 1 {
		t.Fatalf("FindEmails() returned %d emails, want 1", len(emails))
	}

	email := emails[0]
	if len(email.To) != 1 || email.To[0] != "alice@example.com" {
		t.Errorf("To = %v, want [alice@example.com]", email.To)
	}
	if email.From != "sender@example.com" {
		t.Errorf("From = %q, want %q", email.From, "sender@example.com")
	}
	if email.Subject != "Test Email" {
		t.Errorf("Subject = %q, want %q", email.Subject, "Test Email")
	}
	if email.Body == "" {
		t.Error("Body is empty")
	}
}

func TestFindEmails_MultipleEmails(t *testing.T) {
	p, _ := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)

	logContent := `To: alice@example.com, bob@example.com
From: noreply@myapp.com
Subject: Weekly Report

<html><body>Report content</body></html>
--- END EMAIL ---
Some log noise in between
To: dave@company.org
From: alerts@myapp.com
Subject: System Alert

<html><body>Alert content</body></html>
--- END EMAIL ---`

	emails := p.FindEmails(logContent)
	if len(emails) != 2 {
		t.Fatalf("FindEmails() returned %d emails, want 2", len(emails))
	}

	// First email
	if emails[0].Subject != "Weekly Report" {
		t.Errorf("email 1 Subject = %q, want %q", emails[0].Subject, "Weekly Report")
	}
	if len(emails[0].To) != 2 {
		t.Errorf("email 1 has %d recipients, want 2", len(emails[0].To))
	}

	// Second email
	if emails[1].Subject != "System Alert" {
		t.Errorf("email 2 Subject = %q, want %q", emails[1].Subject, "System Alert")
	}
}

func TestFindEmails_NoEmails(t *testing.T) {
	p, _ := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)

	logContent := `2026-08-20 22:00:00 INFO: System started
2026-08-20 22:01:00 INFO: Everything is fine
2026-08-20 22:02:00 WARN: Something weird`

	emails := p.FindEmails(logContent)
	if len(emails) != 0 {
		t.Fatalf("FindEmails() returned %d emails, want 0", len(emails))
	}
}

func TestFindEmails_UnterminatedEmail(t *testing.T) {
	p, _ := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)

	logContent := `To: alice@example.com
From: sender@example.com
Subject: Incomplete

<html><body>This email was never terminated</body></html>`

	emails := p.FindEmails(logContent)
	if len(emails) != 1 {
		t.Fatalf("FindEmails() returned %d emails for unterminated block, want 1", len(emails))
	}
	if emails[0].Subject != "Incomplete" {
		t.Errorf("Subject = %q, want %q", emails[0].Subject, "Incomplete")
	}
}

func TestFindEmails_CaseInsensitive(t *testing.T) {
	p, _ := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)

	logContent := `to: alice@example.com
FROM: sender@example.com
SUBJECT: Case Test

<html><body>test</body></html>
--- END EMAIL ---`

	emails := p.FindEmails(logContent)
	if len(emails) != 1 {
		t.Fatalf("FindEmails() returned %d emails, want 1", len(emails))
	}
	if emails[0].To[0] != "alice@example.com" {
		t.Errorf("To = %v, want [alice@example.com]", emails[0].To)
	}
}

func TestFindEmails_MissingFromAndSubject(t *testing.T) {
	p, _ := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)

	// Email with only To header — should still be parsed (To is the only required field)
	logContent := `To: alice@example.com

<html><body>No from or subject</body></html>
--- END EMAIL ---`

	emails := p.FindEmails(logContent)
	if len(emails) != 1 {
		t.Fatalf("FindEmails() returned %d emails, want 1", len(emails))
	}
	if emails[0].From != "" {
		t.Errorf("From = %q, want empty", emails[0].From)
	}
	if emails[0].Subject != "" {
		t.Errorf("Subject = %q, want empty", emails[0].Subject)
	}
}

func TestFindEmails_EmptyInput(t *testing.T) {
	p, _ := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)

	emails := p.FindEmails("")
	if len(emails) != 0 {
		t.Fatalf("FindEmails(\"\") returned %d emails, want 0", len(emails))
	}
}

func TestFindEmails_LogNoiseBeforeAndAfter(t *testing.T) {
	p, _ := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)

	logContent := `[2026-08-20 10:00:00] INFO Starting up
[2026-08-20 10:00:01] DEBUG Config loaded
[2026-08-20 10:05:33] ERROR Failed to deliver email
To: user@test.com
From: system@test.com
Subject: Alert!
Content-Type: text/html

<h1>Alert</h1>
<p>Something happened.</p>
--- END EMAIL ---
[2026-08-20 10:05:34] INFO Queued for retry
[2026-08-20 10:10:00] DEBUG Heartbeat`

	emails := p.FindEmails(logContent)
	if len(emails) != 1 {
		t.Fatalf("FindEmails() returned %d emails, want 1", len(emails))
	}
	if emails[0].To[0] != "user@test.com" {
		t.Errorf("To = %v, want [user@test.com]", emails[0].To)
	}
	if emails[0].Subject != "Alert!" {
		t.Errorf("Subject = %q, want %q", emails[0].Subject, "Alert!")
	}
}

func TestFindEmails_BodyPreservesHTML(t *testing.T) {
	p, _ := NewParser(`(?i)^To:\s+`, `(?i)^---\s*END\s*EMAIL\s*---`)

	logContent := `To: alice@example.com
From: sender@example.com
Subject: HTML Test

<html>
<head><title>Test</title></head>
<body>
<h1 style="color:red">Big Red Heading</h1>
<table border="1">
<tr><th>Name</th><th>Value</th></tr>
<tr><td>A</td><td>1</td></tr>
</table>
<p>Paragraph with <strong>bold</strong> and <em>italic</em>.</p>
</body>
</html>
--- END EMAIL ---`

	emails := p.FindEmails(logContent)
	if len(emails) != 1 {
		t.Fatalf("FindEmails() returned %d emails, want 1", len(emails))
	}

	body := emails[0].Body
	if body == "" {
		t.Fatal("Body is empty")
	}
	// Verify HTML content is preserved
	for _, want := range []string{"<html>", "<table", "<strong>bold</strong>", "<em>italic</em>"} {
		if !strings.Contains(body, want) {
			t.Errorf("Body missing expected HTML content %q", want)
		}
	}
}
