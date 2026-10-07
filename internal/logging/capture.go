package logging

import (
	"context"
	"log/slog"
	"sync"
)

// Record is a captured log entry.
type Record struct {
	Level   slog.Level
	Message string
}

// Capture collects the entries written to Logger and Progress until Close
// is called. It is meant for tests; captures must not overlap.
type Capture struct {
	mu       sync.Mutex
	records  []Record
	restores []func()
}

// StartCapture redirects Logger and Progress into a new Capture.
func StartCapture() *Capture {
	c := &Capture{}
	h := &captureHandler{c: c}
	c.restores = []func(){setHandler(Logger, h), setHandler(Progress, h)}
	return c
}

// Close stops the capture and restores the previous handlers.
func (c *Capture) Close() {
	for _, r := range c.restores {
		r()
	}
	c.restores = nil
}

// Records returns a copy of every captured entry.
func (c *Capture) Records() []Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Record(nil), c.records...)
}

// Messages returns the messages of the captured entries, in order.
func (c *Capture) Messages() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	msgs := make([]string, len(c.records))
	for i, r := range c.records {
		msgs[i] = r.Message
	}
	return msgs
}

// Warnings returns the messages logged at warning level or above.
func (c *Capture) Warnings() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var msgs []string
	for _, r := range c.records {
		if r.Level >= slog.LevelWarn {
			msgs = append(msgs, r.Message)
		}
	}
	return msgs
}

type captureHandler struct{ c *Capture }

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.c.mu.Lock()
	defer h.c.mu.Unlock()
	h.c.records = append(h.c.records, Record{Level: r.Level, Message: r.Message})
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }
