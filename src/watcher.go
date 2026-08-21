package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a log file for changes and processes new content.
type Watcher struct {
	config    *Config
	parser    *Parser
	sender    *Sender
	store     *Store
	file      *os.File
	fileSize  int64
}

// NewWatcher creates a new file watcher with the given dependencies.
func NewWatcher(config *Config, parser *Parser, sender *Sender, store *Store) *Watcher {
	return &Watcher{
		config: config,
		parser: parser,
		sender: sender,
		store:  store,
	}
}

// Start begins watching the log file for changes.
func (w *Watcher) Start() error {
	// Open the log file
	file, err := os.Open(w.config.LogFile)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	w.file = file

	// Get current file size (we only want new content)
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("stat log file: %w", err)
	}
	w.fileSize = stat.Size()

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		file.Close()
		return fmt.Errorf("creating file watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the directory containing the log file (for log rotation)
	if err := watcher.Add(w.config.LogFile); err != nil {
		file.Close()
		return fmt.Errorf("watching log file: %w", err)
	}

	log.Printf("watching %s (starting at byte %d)", w.config.LogFile, w.fileSize)

	// Process existing content if any
	w.processNewContent()

	// Main event loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				w.processNewContent()
			}
			// Handle log rotation (file recreated)
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("log file recreated, reopening")
				w.file.Close()
				w.file, err = os.Open(w.config.LogFile)
				if err != nil {
					log.Printf("error reopening log file: %v", err)
					continue
				}
				w.fileSize = 0
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)
		case <-time.After(100 * time.Millisecond):
			// Small delay to prevent busy-waiting
		}
	}
}

// processNewContent reads and processes any new content in the log file.
func (w *Watcher) processNewContent() {
	// Seek to where we left off
	if _, err := w.file.Seek(w.fileSize, io.SeekStart); err != nil {
		log.Printf("error seeking: %v", err)
		return
	}

	// Read new content
	buf := make([]byte, 32*1024) // 32KB buffer
	var newContent []string

	for {
		n, err := w.file.Read(buf)
		if n > 0 {
			newContent = append(newContent, string(buf[:n]))
			w.fileSize += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("error reading: %v", err)
			break
		}
	}

	if len(newContent) == 0 {
		return
	}

	// Parse and process emails
	content := ""
	for _, c := range newContent {
		content += c
	}

	emails := w.parser.FindEmails(content)
	for _, email := range emails {
		w.processEmail(email)
	}
}

// processEmail handles a single parsed email.
func (w *Watcher) processEmail(email *Email) {
	hash := email.Hash()

	// Check if we've already sent this email
	if w.store.HasSent(hash) {
		log.Printf("already sent email to %s (hash: %s), skipping", email.To, hash[:12])
		return
	}

	// Send the email
	log.Printf("sending email to %s (subject: %s)", email.To, email.Subject)
	if err := w.sender.Send(email); err != nil {
		log.Printf("ERROR sending email: %v", err)
		return
	}

	// Mark as sent
	if err := w.store.MarkSent(email, hash); err != nil {
		log.Printf("ERROR marking email as sent: %v", err)
		return
	}

	log.Printf("successfully sent email to %s (hash: %s)", email.To, hash[:12])
}

// Stop cleans up resources.
func (w *Watcher) Stop() {
	if w.file != nil {
		w.file.Close()
	}
}
