package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the email retry agent.
type Config struct {
	// LogFile is the path to the log file to watch.
	LogFile string `yaml:"log_file"`

	// StateFile is the path to the SQLite database for tracking sent emails.
	StateFile string `yaml:"state_file"`

	// SMTP configuration
	SMTP struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		From     string `yaml:"from"`
		UseTLS   bool   `yaml:"use_tls"`
	} `yaml:"smtp"`

	// Parser configuration
	Parser struct {
		// EmailStartPattern is a regex pattern that marks the start of an email in the log.
		EmailStartPattern string `yaml:"email_start_pattern"`
		// EmailEndPattern is a regex pattern that marks the end of an email in the log.
		EmailEndPattern string `yaml:"email_end_pattern"`
	} `yaml:"parser"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		LogFile:   "mail.log",
		StateFile: "sent_emails.db",
		SMTP: struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
			From     string `yaml:"from"`
			UseTLS   bool   `yaml:"use_tls"`
		}{
			Host:   "localhost",
			Port:   25,
			From:   "retry-agent@example.com",
			UseTLS: false,
		},
		Parser: struct {
			EmailStartPattern string `yaml:"email_start_pattern"`
			EmailEndPattern   string `yaml:"email_end_pattern"`
		}{
			EmailStartPattern: `(?i)^To:\s+`,
			EmailEndPattern:   `(?i)^---\s*END\s*EMAIL\s*---`,
		},
	}
}

// LoadConfig reads configuration from a YAML file, falling back to defaults.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		// Try default locations
		for _, p := range []string{"config.yaml", "email-retry.yaml", "/etc/email-retry/config.yaml"} {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
		log.Printf("loaded config from %s", path)
	} else {
		log.Printf("no config file found, using defaults")
	}

	return cfg, nil
}

// Email represents a parsed email from the log.
type Email struct {
	To      []string
	From    string
	Subject string
	Body    string // HTML body
	Headers map[string]string
}

// Hash returns a SHA256 hash of the email content for deduplication.
func (e *Email) Hash() string {
	h := sha256.New()
	for _, to := range e.To {
		h.Write([]byte(to))
	}
	h.Write([]byte(e.From))
	h.Write([]byte(e.Subject))
	h.Write([]byte(e.Body))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Store tracks which emails have been sent to prevent duplicates.
type Store struct {
	db *sql.DB
}

// NewStore opens or creates the state database.
func NewStore(path string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating state directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sent_emails (
			hash TEXT PRIMARY KEY,
			to_addresses TEXT,
			from_address TEXT,
			subject TEXT,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("creating table: %w", err)
	}

	return &Store{db: db}, nil
}

// HasSent returns true if an email with this hash has already been sent.
func (s *Store) HasSent(hash string) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM sent_emails WHERE hash = ?", hash).Scan(&count)
	return count > 0
}

// MarkSent records that an email was sent.
func (s *Store) MarkSent(e *Email, hash string) error {
	_, err := s.db.Exec(
		"INSERT INTO sent_emails (hash, to_addresses, from_address, subject) VALUES (?, ?, ?, ?)",
		hash,
		joinStrings(e.To, ", "),
		e.From,
		e.Subject,
	)
	return err
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
