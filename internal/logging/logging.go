package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
)

var (
	// Logger receives warnings about invalid or unsupported input. Rendering
	// continues after a warning, the way browsers handle bad content.
	Logger = slog.New(newSwitch(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	// Progress receives one message per rendering step. It is silent by
	// default.
	Progress = slog.New(newSwitch(slog.NewTextHandler(io.Discard, nil)))
)

// Warnf logs a formatted warning on Logger.
func Warnf(format string, args ...any) {
	if Logger.Enabled(context.Background(), slog.LevelWarn) {
		Logger.Warn(fmt.Sprintf(format, args...))
	}
}

// Errorf logs a formatted error on Logger.
func Errorf(format string, args ...any) {
	Logger.Error(fmt.Sprintf(format, args...))
}

// Progressf logs a formatted rendering step on Progress.
func Progressf(format string, args ...any) {
	if Progress.Enabled(context.Background(), slog.LevelInfo) {
		Progress.Info(fmt.Sprintf(format, args...))
	}
}

// switchHandler forwards to a handler that can be replaced at run time, so
// that tests can capture output without every caller holding a new logger.
type switchHandler struct {
	current *atomic.Pointer[slog.Handler]
	derive  []func(slog.Handler) slog.Handler
}

func newSwitch(h slog.Handler) *switchHandler {
	s := &switchHandler{current: new(atomic.Pointer[slog.Handler])}
	s.current.Store(&h)
	return s
}

func (s *switchHandler) resolve() slog.Handler {
	h := *s.current.Load()
	for _, d := range s.derive {
		h = d(h)
	}
	return h
}

func (s *switchHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return s.resolve().Enabled(ctx, l)
}

func (s *switchHandler) Handle(ctx context.Context, r slog.Record) error {
	return s.resolve().Handle(ctx, r)
}

func (s *switchHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return s.with(func(h slog.Handler) slog.Handler { return h.WithAttrs(attrs) })
}

func (s *switchHandler) WithGroup(name string) slog.Handler {
	return s.with(func(h slog.Handler) slog.Handler { return h.WithGroup(name) })
}

func (s *switchHandler) with(d func(slog.Handler) slog.Handler) slog.Handler {
	derive := make([]func(slog.Handler) slog.Handler, len(s.derive)+1)
	copy(derive, s.derive)
	derive[len(s.derive)] = d
	return &switchHandler{current: s.current, derive: derive}
}

// setHandler replaces the handler behind l, which is always Logger or
// Progress, and returns a function that restores the previous one.
func setHandler(l *slog.Logger, h slog.Handler) (restore func()) {
	s := l.Handler().(*switchHandler)
	prev := s.current.Swap(&h)
	return func() { s.current.Store(prev) }
}
