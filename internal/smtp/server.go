package smtp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/emersion/go-smtp"
)

// Config holds SMTP server configuration.
type Config struct {
	ListenAddr  string
	Hostname    string
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string
	ForceTLS    bool
	EnableAuth  bool
	AuthUsers   map[string]string
	RelayMode   bool
}

func (c *Config) Validate() error {
	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0:2525"
	}
	if c.Hostname == "" {
		return errors.New("smtp hostname is required")
	}
	return nil
}

// Server wraps the SMTP server.
type Server struct {
	cfg       *Config
	processor EmailProcessor
	server    *smtp.Server
	ln        net.Listener
	mu        sync.Mutex
	running   bool
	wg        sync.WaitGroup
}

// EmailProcessor processes received emails.
type EmailProcessor interface {
	ProcessEmail(ctx context.Context, from string, rcpt []string, data []byte) error
}

// NewServer creates a new SMTP server.
func NewServer(cfg *Config, processor EmailProcessor) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SMTP config: %w", err)
	}

	s := smtp.NewServer(&backend{processor: processor})
	s.Addr = cfg.ListenAddr
	s.Domain = cfg.Hostname

	s.AllowInsecureAuth = !cfg.ForceTLS

	return &Server{
		cfg:       cfg,
		processor: processor,
		server:    s,
	}, nil
}

// Start starts the SMTP server.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("server already running")
	}

	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	s.ln = ln
	s.running = true

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.server.Serve(ln)
	}()

	return nil
}

// Stop stops the SMTP server.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	if s.ln != nil {
		s.ln.Close()
	}

	s.wg.Wait()

	return nil
}

// IsRunning returns whether the server is running.
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Addr returns the listening address.
func (s *Server) Addr() string {
	return s.cfg.ListenAddr
}
