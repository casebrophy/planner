package smtpbus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-smtp"

	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/foundation/logger"
)

// Config holds SMTP server configuration.
type Config struct {
	Addr   string
	Domain string
}

// Server wraps the go-smtp server and connects it to the ingestion pipeline.
type Server struct {
	log       *logger.Logger
	ingestBus *ingestbus.Business
	smtpSrv   *smtp.Server
	domain    string
}

// NewServer creates a new SMTP server wired to the ingestion pipeline.
func NewServer(log *logger.Logger, ingestBus *ingestbus.Business, cfg Config) *Server {
	s := &Server{
		log:       log,
		ingestBus: ingestBus,
		domain:    cfg.Domain,
	}

	smtpSrv := smtp.NewServer(s)
	smtpSrv.Addr = cfg.Addr
	smtpSrv.Domain = cfg.Domain
	smtpSrv.ReadTimeout = 30 * time.Second
	smtpSrv.WriteTimeout = 30 * time.Second
	smtpSrv.MaxMessageBytes = 10 * 1024 * 1024 // 10MB
	smtpSrv.MaxRecipients = 5
	smtpSrv.AllowInsecureAuth = true

	s.smtpSrv = smtpSrv

	return s
}

// ListenAndServe starts the SMTP server.
func (s *Server) ListenAndServe() error {
	s.log.Info(context.Background(), "smtp", "status", "starting", "addr", s.smtpSrv.Addr)
	return s.smtpSrv.ListenAndServe()
}

// Close gracefully shuts down the SMTP server.
func (s *Server) Close() error {
	return s.smtpSrv.Close()
}

// NewSession implements smtp.Backend.
func (s *Server) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &session{
		log:       s.log,
		ingestBus: s.ingestBus,
		domain:    s.domain,
	}, nil
}

// session implements smtp.Session for a single SMTP transaction.
type session struct {
	log       *logger.Logger
	ingestBus *ingestbus.Business
	domain    string
	from      string
	to        string
}

// AuthPlain implements smtp.Session. We accept all auth for now (single-user system).
func (s *session) AuthPlain(username, password string) error {
	return nil
}

// Mail implements smtp.Session — called with the sender address.
func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

// Rcpt implements smtp.Session — called with the recipient address.
func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	// Validate domain if configured
	if s.domain != "" && s.domain != "localhost" {
		parts := strings.SplitN(to, "@", 2)
		if len(parts) == 2 && parts[1] != s.domain {
			return fmt.Errorf("recipient domain mismatch: got %s, want %s", parts[1], s.domain)
		}
	}
	s.to = to
	return nil
}

// Data implements smtp.Session — called with the email body.
func (s *session) Data(r io.Reader) error {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return fmt.Errorf("read data: %w", err)
	}

	rawContent := buf.String()

	ctx := context.Background()
	s.log.Info(ctx, "smtp", "msg", "received email", "from", s.from, "to", s.to, "size", len(rawContent))

	if err := s.ingestBus.ProcessEmail(ctx, rawContent); err != nil {
		s.log.Error(ctx, "smtp", "msg", "process email failed", "error", err)
		// Still accept the email (it's stored as failed raw_input)
	}

	return nil
}

// Reset implements smtp.Session.
func (s *session) Reset() {
	s.from = ""
	s.to = ""
}

// Logout implements smtp.Session.
func (s *session) Logout() error {
	return nil
}
