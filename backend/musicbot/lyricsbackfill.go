package musicbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
)

const maxLyricsTrackMs = 20 * 60 * 1000

type LyricsLookup interface {
	Lookup(ctx context.Context, q SongQuery) (*Lyrics, error)
}

type LyricsWriter interface {
	UpdateLyrics(ctx context.Context, station string, id int, lyrics string) error
}

type BackfillRecord struct {
	CheckedAt time.Time
	Written   bool
}

type BackfillStore interface {
	BackfillRecords(ctx context.Context) (map[string]BackfillRecord, error)
	RecordBackfill(ctx context.Context, key string, written bool) error
}

type BackfillStats struct {
	Enabled   bool      `json:"enabled"`
	LastRun   time.Time `json:"last_run,omitempty"`
	Written   int       `json:"written"`
	NotFound  int       `json:"not_found"`
	Pending   int       `json:"pending"`
	LastError string    `json:"last_error,omitempty"`
}

type LyricsBackfill struct {
	library *Library
	lyrics  LyricsLookup
	writer  LyricsWriter
	store   BackfillStore

	Delay      time.Duration
	PerRun     int
	RetryAfter time.Duration

	mu    sync.Mutex
	stats BackfillStats
}

func NewLyricsBackfill(library *Library, lyrics LyricsLookup, writer LyricsWriter, store BackfillStore) *LyricsBackfill {
	return &LyricsBackfill{
		library:    library,
		lyrics:     lyrics,
		writer:     writer,
		store:      store,
		Delay:      3 * time.Second,
		PerRun:     200,
		RetryAfter: 30 * 24 * time.Hour,
		stats:      BackfillStats{Enabled: true},
	}
}

func (j *LyricsBackfill) Stats() BackfillStats {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.stats
}

func (j *LyricsBackfill) Run(ctx context.Context, interval time.Duration) {
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if !j.library.Ready() {
				timer.Reset(time.Minute)
				continue
			}
			if err := j.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Warn("lyrics backfill stopped", slog.Any("error", err))
			}
			timer.Reset(interval)
		}
	}
}

func backfillKey(t LibraryTrack) string {
	if t.SongID != "" {
		return t.SongID
	}
	return t.Key()
}

func (j *LyricsBackfill) RunOnce(ctx context.Context) error {
	records, err := j.store.BackfillRecords(ctx)
	if err != nil {
		return j.finish(0, 0, 0, err)
	}

	now := time.Now()
	candidates := make([]LibraryTrack, 0)
	for _, t := range j.library.All() {
		if strings.TrimSpace(t.Lyrics) != "" || strings.TrimSpace(t.Title) == "" {
			continue
		}
		if rec, seen := records[backfillKey(t)]; seen && (rec.Written || now.Sub(rec.CheckedAt) < j.RetryAfter) {
			continue
		}
		candidates = append(candidates, t)
	}

	written, notFound := 0, 0
	lookups := 0
	for i, t := range candidates {
		if i >= j.PerRun {
			break
		}
		if t.LengthMs > maxLyricsTrackMs {
			notFound++
			if err := j.store.RecordBackfill(ctx, backfillKey(t), false); err != nil {
				return j.finish(written, notFound, len(candidates)-i-1, err)
			}
			continue
		}
		if lookups > 0 && j.Delay > 0 {
			select {
			case <-ctx.Done():
				return j.finish(written, notFound, len(candidates)-i, ctx.Err())
			case <-time.After(j.Delay):
			}
		}

		lookups++
		found, err := j.lyrics.Lookup(ctx, SongQuery{Title: t.Title, Artist: t.Artist, DurationMs: t.LengthMs})
		text := ""
		if err == nil && found != nil && !found.Instrumental {
			text = lyricsText(found)
		}
		if err != nil && !errors.Is(err, ErrLyricsNotFound) {
			return j.finish(written, notFound, len(candidates)-i, fmt.Errorf("lyrics lookup failed: %w", err))
		}
		if text == "" {
			notFound++
			if err := j.store.RecordBackfill(ctx, backfillKey(t), false); err != nil {
				return j.finish(written, notFound, len(candidates)-i-1, err)
			}
			continue
		}

		if err := j.writer.UpdateLyrics(ctx, t.Station, t.ID, text); err != nil {
			var status *azuracast.StatusError
			if errors.As(err, &status) && (status.Status == http.StatusUnauthorized || status.Status == http.StatusForbidden) {
				return j.finish(written, notFound, len(candidates)-i, errors.New("AzuraCast refused the edit; the API key's user needs the Media permission"))
			}
			slog.Warn("failed to save lyrics to AzuraCast",
				slog.String("station", t.Station), slog.Int("id", t.ID), slog.String("title", t.Title), slog.Any("error", err))
			continue
		}
		j.library.SetLyrics(t.Key(), text)
		written++
		if err := j.store.RecordBackfill(ctx, backfillKey(t), true); err != nil {
			return j.finish(written, notFound, len(candidates)-i-1, err)
		}
	}

	return j.finish(written, notFound, max(len(candidates)-j.PerRun, 0), nil)
}

func (j *LyricsBackfill) finish(written, notFound, pending int, err error) error {
	j.mu.Lock()
	j.stats.LastRun = time.Now()
	j.stats.Written += written
	j.stats.NotFound += notFound
	j.stats.Pending = pending
	j.stats.LastError = ""
	if err != nil {
		j.stats.LastError = err.Error()
	}
	j.mu.Unlock()
	if written > 0 || notFound > 0 {
		slog.Info("lyrics backfill run finished",
			slog.Int("written", written), slog.Int("not_found", notFound), slog.Int("pending", pending))
	}
	return err
}

func lyricsText(l *Lyrics) string {
	if len(l.Synced) > 0 {
		return FormatLRC(l.Synced)
	}
	return strings.TrimSpace(l.Plain)
}

func FormatLRC(lines []LyricLine) string {
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		minutes := line.TimeMs / 60000
		seconds := float64(line.TimeMs%60000) / 1000
		fmt.Fprintf(&b, "[%02d:%05.2f]%s", minutes, seconds, line.Text)
	}
	return b.String()
}

func (l *Library) SetLyrics(key, lyrics string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if i, ok := l.byKey[key]; ok {
		l.tracks[i].Lyrics = lyrics
	}
}

func (d *DB) BackfillRecords(ctx context.Context) (map[string]BackfillRecord, error) {
	rows, err := d.Pool.Query(ctx, "SELECT song_key, checked_at, written FROM lyrics_backfill")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]BackfillRecord)
	for rows.Next() {
		var (
			key string
			rec BackfillRecord
		)
		if err := rows.Scan(&key, &rec.CheckedAt, &rec.Written); err != nil {
			return nil, err
		}
		out[key] = rec
	}
	return out, rows.Err()
}

func (d *DB) RecordBackfill(ctx context.Context, key string, written bool) error {
	_, err := d.Pool.Exec(ctx, `
		INSERT INTO lyrics_backfill (song_key, checked_at, written) VALUES ($1, now(), $2)
		ON CONFLICT (song_key) DO UPDATE SET checked_at = now(), written = EXCLUDED.written`,
		Trim(key, 100), written)
	return err
}
