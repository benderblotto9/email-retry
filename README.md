# Email Retry Agent

A cross-platform CLI tool that watches log files for failed email attempts, parses the email content, and automatically resends them via SMTP.

## Features

- **Cross-platform**: Works on Windows, macOS, and Linux
- **Single binary**: No dependencies needed on target machines
- **File watching**: Automatically detects new emails in the log
- **Email parsing**: Extracts recipients, sender, subject, and HTML body
- **Deduplication**: Tracks sent emails to prevent duplicates
- **Configurable**: SMTP settings, log file path, and parsing patterns

## Installation

### Build from source

```bash
# Requires Go 1.22+
go build -o email-retry .
```

### Download pre-built binaries

Check the releases page for pre-compiled binaries for your platform.

## Usage

### Basic usage

```bash
# Watch a log file and resend emails
./email-retry --log /var/log/mail.log

# Use a config file
./email-retry --config config.yaml

# Dry run (parse only, don't send)
./email-retry --dry-run --log test.log
```

### Command-line flags

- `-config string`: Path to config file (default: `config.yaml`)
- `-log string`: Path to log file (overrides config)
- `-dry-run`: Parse and display emails without sending

## Configuration

Create a `config.yaml` file:

```yaml
# Path to the log file to watch
log_file: "/var/log/mail.log"

# Path to SQLite database for tracking sent emails
state_file: "sent_emails.db"

# SMTP server configuration
smtp:
  host: "smtp.gmail.com"
  port: 587
  username: "your-email@gmail.com"
  password: "your-app-password"
  from: "your-email@gmail.com"
  use_tls: true

# Parser configuration
parser:
  # Pattern that marks the start of an email block
  email_start_pattern: "(?i)^To:\\s+"
  
  # Pattern that marks the end of an email block
  email_end_pattern: "(?i)^---\\s*END\\s*EMAIL\\s*---"
```

## Log Format

The tool looks for email blocks in your log file. Each email should:

1. Start with a line matching the `email_start_pattern` (default: `To:`)
2. Include standard email headers (To, From, Subject)
3. Contain the HTML body
4. End with a line matching the `email_end_pattern` (default: `--- END EMAIL ---`)

### Example log entry

```
2026-08-20 22:05:12 ERROR: Failed to send email - retrying
To: alice@example.com, bob@example.com
From: noreply@myapp.com
Subject: Your Weekly Report is Ready
Content-Type: text/html

<html>
<body>
<h1>Your Weekly Report</h1>
<p>Hi there,</p>
<p>Your report is ready.</p>
</body>
</html>
--- END EMAIL ---
```

## Deduplication

The tool maintains a SQLite database (`sent_emails.db` by default) to track which emails have been sent. Each email is hashed based on its content (recipients, sender, subject, body), and if the same hash is encountered again, the email is skipped.

## Building for other platforms

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o email-retry.exe .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o email-retry .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o email-retry .

# Linux
GOOS=linux GOARCH=amd64 go build -o email-retry .
```

## License

MIT
