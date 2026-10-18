package logging

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// Silent is a level above every entry the pipeline writes.
const Silent = slog.LevelError + 4

// SetOutput sends the entries of Logger at level and above to w, one line
// each, as "WARNING: message". It returns a function that restores the
// previous output.
func SetOutput(w io.Writer, level slog.Level) (restore func()) {
	return setHandler(Logger, &lineHandler{out: &lockedWriter{w: w}, level: level})
}

// SetProgressOutput sends the rendering steps of Progress to w, and nothing
// when w is nil. It returns a function that restores the previous output.
func SetProgressOutput(w io.Writer) (restore func()) {
	if w == nil {
		return setHandler(Progress, &lineHandler{out: &lockedWriter{w: io.Discard}, level: Silent})
	}
	return setHandler(Progress, &lineHandler{out: &lockedWriter{w: w}, level: slog.LevelInfo})
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) write(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = io.WriteString(l.w, s)
}

// lineHandler writes each entry as a line made of its level and message,
// followed by its attributes.
type lineHandler struct {
	out   *lockedWriter
	level slog.Level
	attrs []slog.Attr
}

func (h *lineHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.level }

func (h *lineHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(levelName(r.Level))
	b.WriteString(": ")
	b.WriteString(r.Message)
	write := func(a slog.Attr) bool {
		b.WriteByte(' ')
		b.WriteString(a.String())
		return true
	}
	for _, a := range h.attrs {
		write(a)
	}
	r.Attrs(write)
	b.WriteByte('\n')
	h.out.write(b.String())
	return nil
}

func (h *lineHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &c
}

func (h *lineHandler) WithGroup(string) slog.Handler { return h }
