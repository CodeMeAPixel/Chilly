package musicbot

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
)

const (
	mediaDownloadTimeout = 5 * time.Minute
	mediaCacheExt        = ".media"
	mediaPartialExt      = ".part"
)

var ErrMediaTooLarge = errors.New("file is larger than the media cache")

type MediaFetcher interface {
	PlayFile(ctx context.Context, station string, id int, rangeHeader string) (*http.Response, error)
}

type MediaCache struct {
	dir      string
	maxBytes int64
	fetcher  MediaFetcher

	mu       sync.Mutex
	inflight map[string]*mediaDownload
}

type mediaDownload struct {
	done chan struct{}
	err  error
}

func NewMediaCache(dir string, maxBytes int64, fetcher MediaFetcher) (*MediaCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	partials, _ := filepath.Glob(filepath.Join(dir, "*"+mediaPartialExt))
	for _, p := range partials {
		_ = os.Remove(p)
	}
	return &MediaCache{dir: dir, maxBytes: maxBytes, fetcher: fetcher, inflight: map[string]*mediaDownload{}}, nil
}

func (c *MediaCache) Dir() string {
	return c.dir
}

func (c *MediaCache) MaxBytes() int64 {
	return c.maxBytes
}

func (c *MediaCache) Open(ctx context.Context, station string, id int) (*os.File, error) {
	f, _, err := c.open(ctx, station, id)
	return f, err
}

func (c *MediaCache) Warm(ctx context.Context, station string, id int) (bool, error) {
	f, hit, err := c.open(ctx, station, id)
	if err != nil {
		return false, err
	}
	return hit, f.Close()
}

func (c *MediaCache) open(ctx context.Context, station string, id int) (*os.File, bool, error) {
	path := c.path(station, id)
	if f, err := os.Open(path); err == nil {
		now := time.Now()
		_ = os.Chtimes(path, now, now)
		return f, true, nil
	}
	if err := c.download(ctx, station, id, path); err != nil {
		return nil, false, err
	}
	f, err := os.Open(path)
	return f, false, err
}

func (c *MediaCache) path(station string, id int) string {
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, station)
	return filepath.Join(c.dir, safe+"-"+strconv.Itoa(id)+mediaCacheExt)
}

func (c *MediaCache) download(ctx context.Context, station string, id int, path string) error {
	c.mu.Lock()
	d, ok := c.inflight[path]
	if !ok {
		d = &mediaDownload{done: make(chan struct{})}
		c.inflight[path] = d
		go func() {
			d.err = c.fetch(station, id, path)
			if d.err == nil {
				c.evict(path)
			}
			c.mu.Lock()
			delete(c.inflight, path)
			c.mu.Unlock()
			close(d.done)
		}()
	}
	c.mu.Unlock()

	select {
	case <-d.done:
		return d.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *MediaCache) fetch(station string, id int, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), mediaDownloadTimeout)
	defer cancel()
	resp, err := c.fetcher.PlayFile(ctx, station, id, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return &azuracast.StatusError{Status: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	if c.maxBytes > 0 && resp.ContentLength > c.maxBytes {
		return ErrMediaTooLarge
	}

	tmp, err := os.CreateTemp(c.dir, "download-*"+mediaPartialExt)
	if err != nil {
		return err
	}
	n, err := io.Copy(tmp, resp.Body)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil && resp.ContentLength > 0 && n != resp.ContentLength {
		err = io.ErrUnexpectedEOF
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (c *MediaCache) evict(keep string) {
	if c.maxBytes <= 0 {
		return
	}
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	type cachedFile struct {
		path string
		size int64
		used time.Time
	}
	var files []cachedFile
	var total int64
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != mediaCacheExt {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, cachedFile{path: filepath.Join(c.dir, e.Name()), size: info.Size(), used: info.ModTime()})
		total += info.Size()
	}
	if total <= c.maxBytes {
		return
	}
	slices.SortFunc(files, func(a, b cachedFile) int { return a.used.Compare(b.used) })
	for _, f := range files {
		if total <= c.maxBytes {
			return
		}
		if f.path == keep {
			continue
		}
		if os.Remove(f.path) == nil {
			total -= f.size
		}
	}
}
