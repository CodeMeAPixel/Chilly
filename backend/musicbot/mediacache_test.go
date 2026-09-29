package musicbot

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
)

type fakeMediaFetcher struct {
	files map[int]string
	calls atomic.Int32
	delay time.Duration
}

func (f *fakeMediaFetcher) PlayFile(_ context.Context, _ string, id int, _ string) (*http.Response, error) {
	f.calls.Add(1)
	time.Sleep(f.delay)
	body, ok := f.files[id]
	if !ok {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("missing"))}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, ContentLength: int64(len(body)), Body: io.NopCloser(strings.NewReader(body))}, nil
}

func TestMediaCacheDownloadsOnceAndServesFromDisk(t *testing.T) {
	fetcher := &fakeMediaFetcher{files: map[int]string{1: "song-one"}, delay: 20 * time.Millisecond}
	cache, err := NewMediaCache(t.TempDir(), 1<<20, fetcher)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, err := cache.Open(context.Background(), "chilly_radio", 1)
			if err != nil {
				t.Error(err)
				return
			}
			defer f.Close()
			if b, _ := io.ReadAll(f); string(b) != "song-one" {
				t.Errorf("got %q", b)
			}
		}()
	}
	wg.Wait()

	hit, err := cache.Warm(context.Background(), "chilly_radio", 1)
	if err != nil || !hit {
		t.Fatalf("warm after download: hit=%v err=%v", hit, err)
	}
	if n := fetcher.calls.Load(); n != 1 {
		t.Errorf("expected one download, got %d", n)
	}
}

func TestMediaCacheReportsMissingFiles(t *testing.T) {
	cache, err := NewMediaCache(t.TempDir(), 1<<20, &fakeMediaFetcher{files: map[int]string{}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = cache.Open(context.Background(), "chilly_radio", 9)
	var status *azuracast.StatusError
	if !errors.As(err, &status) || status.Status != http.StatusNotFound {
		t.Fatalf("got %v", err)
	}
}

func TestMediaCacheEvictsLeastRecentlyUsed(t *testing.T) {
	dir := t.TempDir()
	fetcher := &fakeMediaFetcher{files: map[int]string{1: "aaaaaaaaaa", 2: "bbbbbbbbbb", 3: "cccccccccc"}}
	cache, err := NewMediaCache(dir, 25, fetcher)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, id := range []int{1, 2} {
		if _, err := cache.Warm(ctx, "s", id); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-time.Hour)
	_ = os.Chtimes(cache.path("s", 1), old, old)
	if _, err := cache.Warm(ctx, "s", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cache.path("s", 1)); !os.IsNotExist(err) {
		t.Errorf("least recently used file should be evicted, stat err = %v", err)
	}
	for _, id := range []int{2, 3} {
		if _, err := os.Stat(cache.path("s", id)); err != nil {
			t.Errorf("file %d should be kept: %v", id, err)
		}
	}
}

func TestMediaCacheRejectsOversizedFilesAndCleansPartials(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "download-1"+mediaPartialExt)
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cache, err := NewMediaCache(dir, 4, &fakeMediaFetcher{files: map[int]string{1: "too large"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale partial download should be removed")
	}
	if _, err := cache.Open(context.Background(), "s", 1); !errors.Is(err, ErrMediaTooLarge) {
		t.Errorf("got %v", err)
	}
}
