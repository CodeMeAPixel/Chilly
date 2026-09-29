package musicbot

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const maxLogValueLength = 4000

type LogEntry struct {
	Time    time.Time         `json:"time"`
	Level   string            `json:"level"`
	Message string            `json:"message"`
	Attrs   map[string]string `json:"attrs,omitempty"`
}

type LogBuffer struct {
	mu      sync.Mutex
	entries []LogEntry
	next    int
	full    bool
	min     slog.Level
}

func NewLogBuffer(size int, minLevel slog.Level) *LogBuffer {
	return &LogBuffer{entries: make([]LogEntry, size), min: minLevel}
}

func (b *LogBuffer) add(entry LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries[b.next] = entry
	b.next = (b.next + 1) % len(b.entries)
	if b.next == 0 {
		b.full = true
	}
}

func (b *LogBuffer) Entries(minLevel slog.Level, limit int) []LogEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	count := b.next
	if b.full {
		count = len(b.entries)
	}
	out := make([]LogEntry, 0, min(limit, count))
	for i := 0; i < count && len(out) < limit; i++ {
		idx := (b.next - 1 - i + len(b.entries)) % len(b.entries)
		entry := b.entries[idx]
		var level slog.Level
		if err := level.UnmarshalText([]byte(entry.Level)); err == nil && level < minLevel {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func (b *LogBuffer) Handler(next slog.Handler) slog.Handler {
	return &logBufferHandler{buf: b, next: next}
}

type logBufferHandler struct {
	buf    *LogBuffer
	next   slog.Handler
	attrs  []slog.Attr
	prefix string
}

func (h *logBufferHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.buf.min || h.next.Enabled(ctx, level)
}

func (h *logBufferHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level >= h.buf.min {
		attrs := make(map[string]string, len(h.attrs)+record.NumAttrs())
		for _, a := range h.attrs {
			flattenAttr(attrs, "", a)
		}
		record.Attrs(func(a slog.Attr) bool {
			flattenAttr(attrs, h.prefix, a)
			return true
		})
		h.buf.add(LogEntry{Time: record.Time.UTC(), Level: record.Level.String(), Message: record.Message, Attrs: attrs})
	}
	if h.next.Enabled(ctx, record.Level) {
		return h.next.Handle(ctx, record)
	}
	return nil
}

func (h *logBufferHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	prefixed := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	prefixed = append(prefixed, h.attrs...)
	for _, a := range attrs {
		a.Key = h.prefix + a.Key
		prefixed = append(prefixed, a)
	}
	return &logBufferHandler{buf: h.buf, next: h.next.WithAttrs(attrs), attrs: prefixed, prefix: h.prefix}
}

func (h *logBufferHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &logBufferHandler{buf: h.buf, next: h.next.WithGroup(name), attrs: h.attrs, prefix: h.prefix + name + "."}
}

func flattenAttr(out map[string]string, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Value.Kind() == slog.KindGroup {
		for _, sub := range a.Value.Group() {
			flattenAttr(out, prefix+a.Key+".", sub)
		}
		return
	}
	if a.Key == "" {
		return
	}
	value := a.Value.String()
	if len(value) > maxLogValueLength {
		value = value[:maxLogValueLength] + "…"
	}
	out[prefix+strings.TrimSpace(a.Key)] = value
}
