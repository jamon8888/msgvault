package smtp

import (
	"context"
	"io"

	"github.com/emersion/go-smtp"
)

// backend implements the smtp.Backend interface.
type backend struct {
	processor EmailProcessor
}

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &session{
		processor: b.processor,
		ctx:       context.Background(),
	}, nil
}

// session implements smtp.Session.
type session struct {
	processor EmailProcessor
	ctx       context.Context
	from      string
	to        []string
}

func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *session) Data(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return s.processor.ProcessEmail(s.ctx, s.from, s.to, data)
}

func (s *session) Reset() {
	s.from = ""
	s.to = nil
}

func (s *session) Logout() error {
	return nil
}
