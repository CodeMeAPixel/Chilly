package musicbot

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
)

type fakeLyrics map[string]*Lyrics

func (f fakeLyrics) Lookup(_ context.Context, q SongQuery) (*Lyrics, error) {
	if l, ok := f[q.Title]; ok {
		return l, nil
	}
	return nil, ErrLyricsNotFound
}

type fakeWriter struct {
	writes map[string]string
	err    error
}

func (f *fakeWriter) UpdateLyrics(_ context.Context, station string, id int, lyrics string) error {
	if f.err != nil {
		return f.err
	}
	f.writes[LibraryTrack{Station: station, ID: id}.Key()] = lyrics
	return nil
}

type memoryStore map[string]BackfillRecord

func (m memoryStore) BackfillRecords(context.Context) (map[string]BackfillRecord, error) {
	out := make(map[string]BackfillRecord, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out, nil
}

func (m memoryStore) RecordBackfill(_ context.Context, key string, written bool) error {
	m[key] = BackfillRecord{CheckedAt: time.Now(), Written: written}
	return nil
}

func newTestBackfill(t *testing.T, lyrics fakeLyrics, writer *fakeWriter, store memoryStore) (*LyricsBackfill, *Library) {
	t.Helper()
	lib := testLibrary(t)
	job := NewLyricsBackfill(lib, lyrics, writer, store)
	job.Delay = 0
	return job, lib
}

func TestBackfillWritesFoundLyricsAndRemembersMisses(t *testing.T) {
	writer := &fakeWriter{writes: map[string]string{}}
	store := memoryStore{}
	job, lib := newTestBackfill(t, fakeLyrics{
		"Sunset Lover":  {Synced: []LyricLine{{TimeMs: 1500, Text: "first"}, {TimeMs: 65250, Text: "second"}}},
		"Problems":      {Plain: "plain words"},
		"Midnight City": {Instrumental: true},
	}, writer, store)

	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := writer.writes["lib:chill:1"]; got != "[00:01.50]first\n[01:05.25]second" {
		t.Errorf("synced lyrics should be written as LRC, got %q", got)
	}
	if got := writer.writes["lib:chill:2"]; got != "plain words" {
		t.Errorf("plain lyrics not written, got %q", got)
	}
	if _, wrote := writer.writes["lib:chill:3"]; wrote {
		t.Error("instrumental songs must not get lyrics")
	}
	if track, _ := lib.Get("lib:chill:1"); track.Lyrics == "" {
		t.Error("the in-memory library should pick up written lyrics immediately")
	}
	stats := job.Stats()
	if stats.Written != 2 || stats.NotFound != 3 || stats.LastError != "" {
		t.Errorf("unexpected stats %+v", stats)
	}
	if !store["a"].Written || store["c"].Written {
		t.Errorf("store should record written and missing songs: %+v", store)
	}

	writer.writes = map[string]string{}
	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(writer.writes) != 0 {
		t.Errorf("a second run must not redo checked songs, wrote %v", writer.writes)
	}
}

func TestBackfillRetriesOldMisses(t *testing.T) {
	writer := &fakeWriter{writes: map[string]string{}}
	store := memoryStore{"b": {CheckedAt: time.Now().Add(-40 * 24 * time.Hour)}, "c": {CheckedAt: time.Now()}}
	job, _ := newTestBackfill(t, fakeLyrics{"Problems": {Plain: "now available"}, "Midnight City": {Plain: "skip"}}, writer, store)

	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if writer.writes["lib:chill:2"] != "now available" {
		t.Error("misses older than the retry window should be retried")
	}
	if _, wrote := writer.writes["lib:chill:3"]; wrote {
		t.Error("recent misses must be skipped")
	}
}

func TestBackfillStopsWhenAzuraCastRefuses(t *testing.T) {
	writer := &fakeWriter{writes: map[string]string{}, err: &azuracast.StatusError{Status: 403, Body: "forbidden"}}
	job, _ := newTestBackfill(t, fakeLyrics{"Sunset Lover": {Plain: "words"}, "Problems": {Plain: "more"}}, writer, memoryStore{})

	err := job.RunOnce(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Media permission") {
		t.Fatalf("expected a permission error, got %v", err)
	}
	if job.Stats().LastError == "" {
		t.Error("the error should be visible in stats")
	}
}

func TestBackfillRespectsPerRunLimitAndLookupErrors(t *testing.T) {
	writer := &fakeWriter{writes: map[string]string{}}
	job, _ := newTestBackfill(t, fakeLyrics{}, writer, memoryStore{})
	job.PerRun = 2
	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if stats := job.Stats(); stats.NotFound != 2 || stats.Pending != 3 {
		t.Errorf("per-run limit ignored: %+v", stats)
	}

	failing := NewLyricsBackfill(testLibrary(t), lookupFunc(func() error { return errors.New("lrclib down") }), writer, memoryStore{})
	failing.Delay = 0
	if err := failing.RunOnce(context.Background()); err == nil || !strings.Contains(err.Error(), "lrclib down") {
		t.Errorf("lookup outages should stop the run, got %v", err)
	}
}

type lookupFunc func() error

func (f lookupFunc) Lookup(context.Context, SongQuery) (*Lyrics, error) {
	return nil, f()
}

type countingLyrics struct {
	calls int
}

func (c *countingLyrics) Lookup(context.Context, SongQuery) (*Lyrics, error) {
	c.calls++
	return nil, ErrLyricsNotFound
}

func TestBackfillSkipsLongMixesWithoutLookup(t *testing.T) {
	lib := NewLibrary(fakeFetcher{"mix": {
		{ID: 1, SongID: "long", Title: "Two Hour Mix", Artist: "DJ", Length: 7200},
		{ID: 2, SongID: "song", Title: "Normal Song", Artist: "Band", Length: 210},
	}}, func() []string { return []string{"mix"} })
	if err := lib.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	lookup := &countingLyrics{}
	store := memoryStore{}
	job := NewLyricsBackfill(lib, lookup, &fakeWriter{writes: map[string]string{}}, store)
	job.Delay = 0

	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if lookup.calls != 1 {
		t.Errorf("only the normal song should be looked up, got %d lookups", lookup.calls)
	}
	if _, recorded := store["long"]; !recorded {
		t.Error("long mixes should be recorded so they aren't retried every run")
	}
	if stats := job.Stats(); stats.NotFound != 2 {
		t.Errorf("unexpected stats %+v", stats)
	}
}
