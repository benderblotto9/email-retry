package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ============================================================
// Email Hash Tests
// ============================================================

func TestEmailHash_Deterministic(t *testing.T) {
	e := &Email{
		To:      []string{"alice@example.com"},
		From:    "sender@example.com",
		Subject: "Test",
		Body:    "<html><body>hello</body></html>",
	}

	h1 := e.Hash()
	h2 := e.Hash()

	if h1 != h2 {
		t.Errorf("Hash() is not deterministic: %q != %q", h1, h2)
	}
}

func TestEmailHash_DifferentEmails(t *testing.T) {
	e1 := &Email{
		To:      []string{"alice@example.com"},
		From:    "sender@example.com",
		Subject: "Subject A",
		Body:    "body A",
	}
	e2 := &Email{
		To:      []string{"alice@example.com"},
		From:    "sender@example.com",
		Subject: "Subject B",
		Body:    "body A",
	}

	if e1.Hash() == e2.Hash() {
		t.Error("different emails should produce different hashes")
	}
}

func TestEmailHash_DifferentRecipients(t *testing.T) {
	e1 := &Email{
		To:      []string{"alice@example.com"},
		From:    "sender@example.com",
		Subject: "Test",
		Body:    "body",
	}
	e2 := &Email{
		To:      []string{"bob@example.com"},
		From:    "sender@example.com",
		Subject: "Test",
		Body:    "body",
	}

	if e1.Hash() == e2.Hash() {
		t.Error("emails with different recipients should produce different hashes")
	}
}

func TestEmailHash_MultipleRecipients(t *testing.T) {
	e1 := &Email{
		To:      []string{"alice@example.com", "bob@example.com"},
		From:    "sender@example.com",
		Subject: "Test",
		Body:    "body",
	}
	e2 := &Email{
		To:      []string{"bob@example.com", "alice@example.com"},
		From:    "sender@example.com",
		Subject: "Test",
		Body:    "body",
	}

	// Different order should produce different hashes (simple concatenation)
	if e1.Hash() == e2.Hash() {
		t.Error("different recipient order should produce different hashes")
	}
}

// ============================================================
// DefaultConfig Tests
// ============================================================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LogFile != "mail.log" {
		t.Errorf("LogFile = %q, want %q", cfg.LogFile, "mail.log")
	}
	if cfg.StateFile != "sent_emails.db" {
		t.Errorf("StateFile = %q, want %q", cfg.StateFile, "sent_emails.db")
	}
	if cfg.SMTP.Host != "localhost" {
		t.Errorf("SMTP.Host = %q, want %q", cfg.SMTP.Host, "localhost")
	}
	if cfg.SMTP.Port != 25 {
		t.Errorf("SMTP.Port = %d, want 25", cfg.SMTP.Port)
	}
	if cfg.SMTP.From != "retry-agent@example.com" {
		t.Errorf("SMTP.From = %q, want %q", cfg.SMTP.From, "retry-agent@example.com")
	}
	if cfg.SMTP.UseTLS != false {
		t.Error("SMTP.UseTLS should default to false")
	}
	if cfg.Parser.EmailStartPattern == "" {
		t.Error("Parser.EmailStartPattern should not be empty")
	}
	if cfg.Parser.EmailEndPattern == "" {
		t.Error("Parser.EmailEndPattern should not be empty")
	}
}

// ============================================================
// LoadConfig Tests
// ============================================================

func TestLoadConfig_ValidFile(t *testing.T) {
	content := `
log_file: "/var/log/custom.log"
state_file: "/tmp/test.db"
smtp:
  host: "smtp.gmail.com"
  port: 587
  username: "user@gmail.com"
  password: "secret123"
  from: "user@gmail.com"
  use_tls: true
parser:
  email_start_pattern: "(?i)^To:\\s+"
  email_end_pattern: "(?i)^---\\s*END\\s*EMAIL\\s*---"
`
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	if cfg.LogFile != "/var/log/custom.log" {
		t.Errorf("LogFile = %q, want %q", cfg.LogFile, "/var/log/custom.log")
	}
	if cfg.SMTP.Host != "smtp.gmail.com" {
		t.Errorf("SMTP.Host = %q, want %q", cfg.SMTP.Host, "smtp.gmail.com")
	}
	if cfg.SMTP.Port != 587 {
		t.Errorf("SMTP.Port = %d, want 587", cfg.SMTP.Port)
	}
	if cfg.SMTP.Username != "user@gmail.com" {
		t.Errorf("SMTP.Username = %q, want %q", cfg.SMTP.Username, "user@gmail.com")
	}
	if cfg.SMTP.Password != "secret123" {
		t.Errorf("SMTP.Password = %q, want %q", cfg.SMTP.Password, "secret123")
	}
	if cfg.SMTP.UseTLS != true {
		t.Error("SMTP.UseTLS should be true")
	}
}

func TestLoadConfig_NonexistentFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("LoadConfig() should return error for nonexistent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(cfgPath, []byte("{{{{invalid yaml"), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("LoadConfig() should return error for invalid YAML")
	}
}

func TestLoadConfig_EmptyPath_UsesDefaults(t *testing.T) {
	// Run from a temp dir where no config.yaml exists so defaults are used
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig(\"\") error: %v", err)
	}
	// Should have defaults
	if cfg.LogFile != "mail.log" {
		t.Errorf("LogFile = %q, want %q (default)", cfg.LogFile, "mail.log")
	}
}

// ============================================================
// joinStrings Tests
// ============================================================

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		strs     []string
		sep      string
		expected string
	}{
		{"single element", []string{"a"}, ",", "a"},
		{"multiple elements", []string{"a", "b", "c"}, ", ", "a, b, c"},
		{"empty slice", []string{}, ",", ""},
		{"nil slice", nil, ",", ""},
		{"pipe separator", []string{"x", "y"}, "|", "x|y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.strs, tt.sep)
			if result != tt.expected {
				t.Errorf("joinStrings(%v, %q) = %q, want %q",
					tt.strs, tt.sep, result, tt.expected)
			}
		})
	}
}

// ============================================================
// Store Tests
// ============================================================

func TestStore_HasSent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	defer store.Close()

	email := &Email{
		To:      []string{"alice@example.com"},
		From:    "sender@example.com",
		Subject: "Test",
		Body:    "body",
	}
	hash := email.Hash()

	// Should not be sent yet
	if store.HasSent(hash) {
		t.Error("HasSent() returned true for new email")
	}

	// Mark as sent
	if err := store.MarkSent(email, hash); err != nil {
		t.Fatalf("MarkSent() error: %v", err)
	}

	// Now should be sent
	if !store.HasSent(hash) {
		t.Error("HasSent() returned false after MarkSent()")
	}

	// Different hash should not be sent
	if store.HasSent("nonexistent-hash") {
		t.Error("HasSent() returned true for nonexistent hash")
	}
}

func TestStore_MarkSent_DuplicateHash(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	defer store.Close()

	email := &Email{
		To:      []string{"alice@example.com"},
		From:    "sender@example.com",
		Subject: "Test",
		Body:    "body",
	}
	hash := email.Hash()

	// First insert should succeed
	if err := store.MarkSent(email, hash); err != nil {
		t.Fatalf("MarkSent() first call error: %v", err)
	}

	// Duplicate insert should fail (primary key constraint)
	if err := store.MarkSent(email, hash); err == nil {
		t.Error("MarkSent() should return error for duplicate hash")
	}
}

func TestStore_MultipleEmails(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	defer store.Close()

	emails := []*Email{
		{To: []string{"a@test.com"}, From: "s1@test.com", Subject: "A", Body: "a"},
		{To: []string{"b@test.com"}, From: "s2@test.com", Subject: "B", Body: "b"},
		{To: []string{"c@test.com"}, From: "s3@test.com", Subject: "C", Body: "c"},
	}

	for _, e := range emails {
		if err := store.MarkSent(e, e.Hash()); err != nil {
			t.Fatalf("MarkSent() error for %q: %v", e.Subject, err)
		}
	}

	for _, e := range emails {
		if !store.HasSent(e.Hash()) {
			t.Errorf("HasSent() = false for %q", e.Subject)
		}
	}
}

func TestStore_NewStore_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("NewStore() should create the database file")
	}
}
