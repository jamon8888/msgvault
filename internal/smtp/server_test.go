package smtp

import (
	"context"
	"sync"
	"testing"
)

type mockProcessor struct {
	mu       sync.Mutex
	received map[string][]byte
}

func (m *mockProcessor) ProcessEmail(ctx context.Context, from string, rcpt []string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.received == nil {
		m.received = make(map[string][]byte)
	}
	m.received[from] = data
	return nil
}

func TestServer_StartStop(t *testing.T) {
	cfg := &Config{
		ListenAddr: "127.0.0.1:19925",
		Hostname:   "test.local",
	}

	server, err := NewServer(cfg, &mockProcessor{})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	if !server.IsRunning() {
		t.Fatal("server should be running")
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("failed to stop server: %v", err)
	}

	if server.IsRunning() {
		t.Fatal("server should be stopped")
	}
}

func TestServer_InvalidConfig(t *testing.T) {
	cfg := &Config{
		ListenAddr: "127.0.0.1:19926",
		Hostname:   "", // Empty hostname should fail
	}

	_, err := NewServer(cfg, &mockProcessor{})
	if err == nil {
		t.Fatal("expected error for empty hostname")
	}
}

func TestServer_PortInUse(t *testing.T) {
	cfg := &Config{
		ListenAddr: "127.0.0.1:19927",
		Hostname:   "test.local",
	}

	server1, err := NewServer(cfg, &mockProcessor{})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server1.Start(); err != nil {
		t.Fatalf("failed to start first server: %v", err)
	}
	defer server1.Stop()

	// Try to start second server on same port
	server2, err := NewServer(cfg, &mockProcessor{})
	if err != nil {
		t.Fatalf("failed to create second server: %v", err)
	}

	if err := server2.Start(); err == nil {
		t.Fatal("expected error when port is in use")
		server2.Stop()
	}
}
