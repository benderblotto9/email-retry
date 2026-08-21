package main

import (
	"bufio"
	"regexp"
	"strings"
)

// Parser extracts email data from log content.
type Parser struct {
	startPattern *regexp.Regexp
	endPattern   *regexp.Regexp
}

// NewParser creates a parser with the configured patterns.
func NewParser(startPattern, endPattern string) (*Parser, error) {
	start, err := regexp.Compile(startPattern)
	if err != nil {
		return nil, err
	}
	end, err := regexp.Compile(endPattern)
	if err != nil {
		return nil, err
	}
	return &Parser{
		startPattern: start,
		endPattern:   end,
	}, nil
}

// FindEmails scans the given content and returns all parsed emails found.
func (p *Parser) FindEmails(content string) []*Email {
	var emails []*Email
	scanner := bufio.NewScanner(strings.NewReader(content))

	var inEmail bool
	var emailLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if !inEmail {
			if p.startPattern.MatchString(line) {
				inEmail = true
				emailLines = []string{line}
			}
			continue
		}

		emailLines = append(emailLines, line)

		if p.endPattern.MatchString(line) {
			inEmail = false
			if email := p.parseEmail(emailLines); email != nil {
				emails = append(emails, email)
			}
			emailLines = nil
		}
	}

	// Handle case where email wasn't terminated
	if inEmail && len(emailLines) > 0 {
		if email := p.parseEmail(emailLines); email != nil {
			emails = append(emails, email)
		}
	}

	return emails
}

// parseEmail extracts email fields from a block of lines.
func (p *Parser) parseEmail(lines []string) *Email {
	if len(lines) == 0 {
		return nil
	}

	email := &Email{
		Headers: make(map[string]string),
	}

	var bodyStart int
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToLower(trimmed), "to:") {
			email.To = parseAddressList(trimmed[3:])
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "from:") {
			email.From = strings.TrimSpace(trimmed[5:])
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "subject:") {
			email.Subject = strings.TrimSpace(trimmed[8:])
			continue
		}

		// First non-header line marks body start
		if email.To != nil && bodyStart == 0 && trimmed != "" {
			bodyStart = i
			break
		}
	}

	if bodyStart > 0 {
		email.Body = strings.Join(lines[bodyStart:], "\n")
	}

	// Validate we have minimum required fields
	if len(email.To) == 0 {
		return nil
	}

	return email
}

// parseAddressList parses a comma-separated list of email addresses.
func parseAddressList(s string) []string {
	var addrs []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			// Strip angle brackets if present: <user@example.com> -> user@example.com
			trimmed = strings.Trim(trimmed, "<>")
			addrs = append(addrs, trimmed)
		}
	}
	return addrs
}
