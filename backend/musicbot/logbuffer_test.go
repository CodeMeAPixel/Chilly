package musicbot

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestLogBufferKeepsNewestFirstAndFilters(t *testing.T) {
	buf := NewLogBuffer(3, slog.LevelInfo)
	logger := slog.New(buf.Handler(slog.NewTextHandler(io.Discard, nil)))

	logger.Debug("ignored")
	logger.Info("one")
	logger.Warn("two", slog.String("guild_id", "1"))
	logger.Info("three")
	logger.Error("four")

	entries := buf.Entries(slog.LevelInfo, 10)
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3 (ring size)", len(entries))
	}
	if entries[0].Message != "four" || entries[1].Message != "three" || entries[2].Message != "two" {
		t.Fatalf("unexpected order: %s, %s, %s", entries[0].Message, entries[1].Message, entries[2].Message)
	}
	if entries[2].Attrs["guild_id"] != "1" {
		t.Errorf("attrs not captured: %v", entries[2].Attrs)
	}

	warn := buf.Entries(slog.LevelWarn, 10)
	if len(warn) != 2 || warn[0].Message != "four" || warn[1].Message != "two" {
		t.Errorf("level filter failed: %+v", warn)
	}
	if limited := buf.Entries(slog.LevelInfo, 1); len(limited) != 1 {
		t.Errorf("limit ignored: %d", len(limited))
	}
}

func TestLogBufferGroupsAndTruncation(t *testing.T) {
	buf := NewLogBuffer(10, slog.LevelInfo)
	logger := slog.New(buf.Handler(slog.NewTextHandler(io.Discard, nil))).
		With(slog.String("name", "client")).
		WithGroup("req")

	logger.Info("long", slog.String("body", strings.Repeat("x", maxLogValueLength+50)), slog.Group("user", slog.String("id", "7")))

	entry := buf.Entries(slog.LevelInfo, 1)[0]
	if entry.Attrs["name"] != "client" {
		t.Errorf("handler attrs missing: %v", entry.Attrs)
	}
	if entry.Attrs["req.user.id"] != "7" {
		t.Errorf("group attrs not flattened: %v", entry.Attrs)
	}
	if body := entry.Attrs["req.body"]; len([]rune(body)) != maxLogValueLength+1 {
		t.Errorf("long value not truncated, got %d runes", len([]rune(body)))
	}
}

func TestLogBufferPassesThroughToNextHandler(t *testing.T) {
	var out strings.Builder
	buf := NewLogBuffer(5, slog.LevelWarn)
	logger := slog.New(buf.Handler(slog.NewTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug})))

	logger.Debug("debug line")
	if !strings.Contains(out.String(), "debug line") {
		t.Error("records below the buffer level must still reach the next handler")
	}
	if len(buf.Entries(slog.LevelDebug, 10)) != 0 {
		t.Error("records below the buffer level must not be stored")
	}
	if !buf.Handler(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})).Enabled(context.Background(), slog.LevelWarn) {
		t.Error("handler should be enabled for levels the buffer records")
	}
}
