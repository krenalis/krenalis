package cmd

import (
	"context"
	"log/slog"
	"sync"
	"testing"
)

// Test_httpLogger_Write checks that httpLogger.Write logs expected messages and
// skips unwanted ones.
func Test_httpLogger_Write(t *testing.T) {

	handler := &captureHandler{}
	orig := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(orig)

	l := &httpLogger{}

	tests := []struct {
		name    string
		input   []byte
		wantMsg string
	}{
		{"empty", []byte{}, ""},
		{"tls without newline", append(tlsHandshakeMsg, []byte("1.2.3.4:1: EOF")...), ""},
		{"tls with newline", append(append([]byte{}, tlsHandshakeMsg...), []byte("1.2.3.4:1: EOF\n")...), ""},
		{"trim newline", []byte("hello world\n"), "hello world"},
		{"plain", []byte("simple log"), "simple log"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler.reset()
			n, err := l.Write(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if n != len(tt.input) {
				t.Fatalf("expected %d bytes written, got %d", len(tt.input), n)
			}
			got := handler.Messages()
			if tt.wantMsg == "" {
				if len(got) != 0 {
					t.Errorf("expected no log message, got %v", got)
				}
			} else {
				if len(got) != 1 {
					t.Errorf("expected one log message, got %v", got)
					return
				}
				if got[0] != tt.wantMsg {
					t.Errorf("expected %q, got %q", tt.wantMsg, got[0])
				}
			}
		})
	}

}

// captureHandler is a slog.Handler that records log messages and their levels.
type captureHandler struct {
	mu   sync.Mutex
	msgs []slog.Record
}

func (h *captureHandler) Enabled(ctx context.Context, level slog.Level) bool { return true }

func (h *captureHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Make a copy of the record, since slog.Record may be reused internally.
	rec := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	h.msgs = append(h.msgs, rec)
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(name string) slog.Handler       { return h }

func (h *captureHandler) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.msgs = h.msgs[:0]
}

// Messages returns a copy of the recorded messages.
func (h *captureHandler) Messages() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	msgs := make([]string, len(h.msgs))
	for i, rec := range h.msgs {
		msgs[i] = rec.Message
	}
	return msgs
}
