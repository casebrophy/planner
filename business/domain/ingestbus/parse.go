package ingestbus

import (
	"fmt"
	"io"
	"mime"
	"strings"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

// ParsedEmail holds the parsed components of an email message.
type ParsedEmail struct {
	MessageID   string
	FromAddress string
	FromName    string
	ToAddress   string
	Subject     string
	BodyText    string
	BodyHTML    string
}

// parseEmail parses a raw RFC 5322 email message into structured components.
func parseEmail(rawContent string) (ParsedEmail, error) {
	r := strings.NewReader(rawContent)

	mr, err := mail.CreateReader(r)
	if err != nil {
		return ParsedEmail{}, fmt.Errorf("create mail reader: %w", err)
	}
	defer mr.Close()

	header := mr.Header

	var parsed ParsedEmail

	// Message-ID
	parsed.MessageID, _ = header.MessageID()

	// From
	fromAddrs, err := header.AddressList("From")
	if err == nil && len(fromAddrs) > 0 {
		parsed.FromAddress = fromAddrs[0].Address
		parsed.FromName = fromAddrs[0].Name
	}

	// To
	toAddrs, err := header.AddressList("To")
	if err == nil && len(toAddrs) > 0 {
		parsed.ToAddress = toAddrs[0].Address
	}

	// Subject
	parsed.Subject, _ = header.Subject()

	// Body parts
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed parts
			continue
		}

		ct, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))

		body, err := io.ReadAll(p.Body)
		if err != nil {
			continue
		}

		switch ct {
		case "text/plain", "":
			if parsed.BodyText == "" {
				parsed.BodyText = string(body)
			}
		case "text/html":
			if parsed.BodyHTML == "" {
				parsed.BodyHTML = string(body)
			}
		}
	}

	if parsed.BodyText == "" && parsed.BodyHTML != "" {
		parsed.BodyText = parsed.BodyHTML
	}

	return parsed, nil
}

// parseEmailEntity parses a MIME entity from go-message into structured components.
func parseEmailEntity(entity *message.Entity) (ParsedEmail, error) {
	mr := mail.NewReader(entity)
	defer mr.Close()

	header := mr.Header

	var parsed ParsedEmail

	parsed.MessageID, _ = header.MessageID()

	fromAddrs, err := header.AddressList("From")
	if err == nil && len(fromAddrs) > 0 {
		parsed.FromAddress = fromAddrs[0].Address
		parsed.FromName = fromAddrs[0].Name
	}

	toAddrs, err := header.AddressList("To")
	if err == nil && len(toAddrs) > 0 {
		parsed.ToAddress = toAddrs[0].Address
	}

	parsed.Subject, _ = header.Subject()

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		ct, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))

		body, err := io.ReadAll(p.Body)
		if err != nil {
			continue
		}

		switch ct {
		case "text/plain", "":
			if parsed.BodyText == "" {
				parsed.BodyText = string(body)
			}
		case "text/html":
			if parsed.BodyHTML == "" {
				parsed.BodyHTML = string(body)
			}
		}
	}

	if parsed.BodyText == "" && parsed.BodyHTML != "" {
		parsed.BodyText = parsed.BodyHTML
	}

	return parsed, nil
}
