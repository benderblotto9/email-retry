package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Command-line flags
	configPath := flag.String("config", "", "path to config file (default: config.yaml)")
	logFile := flag.String("log", "", "path to log file (overrides config)")
	dryRun := flag.Bool("dry-run", false, "parse and display emails without sending")
	flag.Parse()

	// Load configuration
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Override log file if specified
	if *logFile != "" {
		cfg.LogFile = *logFile
	}

	// Validate config
	if cfg.LogFile == "" {
		log.Fatal("log file not specified (use -log flag or set log_file in config)")
	}

	// Check log file exists
	if _, err := os.Stat(cfg.LogFile); os.IsNotExist(err) {
		log.Fatalf("log file does not exist: %s", cfg.LogFile)
	}

	// Initialize parser
	parser, err := NewParser(cfg.Parser.EmailStartPattern, cfg.Parser.EmailEndPattern)
	if err != nil {
		log.Fatalf("failed to create parser: %v", err)
	}

	// Initialize sender
	sender := NewSender(cfg)

	// Initialize store
	store, err := NewStore(cfg.StateFile)
	if err != nil {
		log.Fatalf("failed to open state database: %v", err)
	}
	defer store.Close()

	// Create watcher
	watcher := NewWatcher(cfg, parser, sender, store)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("shutting down...")
		watcher.Stop()
		os.Exit(0)
	}()

	// Dry run mode: just show what's in the log
	if *dryRun {
		log.Println("dry run mode - parsing log file without sending")
		content, err := os.ReadFile(cfg.LogFile)
		if err != nil {
			log.Fatalf("failed to read log file: %v", err)
		}

		emails := parser.FindEmails(string(content))
		if len(emails) == 0 {
			log.Println("no emails found in log file")
			return
		}

		for i, email := range emails {
			hash := email.Hash()
			alreadySent := store.HasSent(hash)
			status := "pending"
			if alreadySent {
				status = "already sent"
			}

			log.Printf("email %d [%s]:", i+1, status)
			log.Printf("  To: %v", email.To)
			log.Printf("  From: %s", email.From)
			log.Printf("  Subject: %s", email.Subject)
			log.Printf("  Hash: %s", hash)
			log.Println()
		}
		return
	}

	// Start watching
	log.Printf("email retry agent starting...")
	if err := watcher.Start(); err != nil {
		log.Fatalf("watcher failed: %v", err)
	}
}
