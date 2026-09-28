package musicbot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
)

func TestRememberKeepsValuesWithinDiscordLimit(t *testing.T) {
	s := NewSearcher(nil, nil)

	short := lavalink.Track{Encoded: "short", Info: lavalink.TrackInfo{URI: json.Ptr("https://youtu.be/abc")}}
	if v := s.Remember(short); v != "https://youtu.be/abc" {
		t.Fatalf("short uri should be used as-is, got %q", v)
	}

	long := lavalink.Track{Encoded: "long", Info: lavalink.TrackInfo{URI: json.Ptr("https://soundcloud.com/" + strings.Repeat("x", 120))}}
	v := s.Remember(long)
	if len(v) > 100 || !strings.HasPrefix(v, selectionPrefix) {
		t.Fatalf("long uri should map to a short selection value, got %q", v)
	}
	if got, ok := s.selections.Get(v); !ok || got.Encoded != "long" {
		t.Fatal("selection not cached")
	}

	noURI := lavalink.Track{Encoded: "nouri"}
	if v := s.Remember(noURI); !strings.HasPrefix(v, selectionPrefix) {
		t.Fatalf("track without uri should get a selection value, got %q", v)
	}
}

func TestProvidersForAliases(t *testing.T) {
	s := NewSearcher(nil, []string{"youtube", "scsearch:", " spsearch "})
	want := []lavalink.SearchType{"ytsearch", "scsearch", "spsearch"}
	if len(s.providers) != len(want) {
		t.Fatalf("got %v", s.providers)
	}
	for i := range want {
		if s.providers[i] != want[i] {
			t.Fatalf("got %v, want %v", s.providers, want)
		}
	}
	if p, _ := s.providersFor("deezer"); len(p) != 1 || p[0] != "dzsearch" {
		t.Fatalf("got %v", p)
	}
}

func TestTrackHelpersHandleNilPointers(t *testing.T) {
	track := lavalink.Track{Info: lavalink.TrackInfo{Title: "[weird]*title*"}}
	if TrackURL(track) != "" || TrackArtwork(track) != "" {
		t.Fatal("expected empty strings for nil pointers")
	}
	if got := TrackLink(track); got != `\[weird\]\*title\*` {
		t.Fatalf("unexpected link %q", got)
	}
}

func TestFilterSupported(t *testing.T) {
	s := NewSearcher(nil, []string{"spsearch", "scsearch", "dzsearch", "custom"})
	kept, dropped := s.FilterSupported([]string{"soundcloud", "http"})
	if len(kept) != 2 || kept[0] != "scsearch" || kept[1] != "custom" {
		t.Fatalf("kept %v", kept)
	}
	if len(dropped) != 2 {
		t.Fatalf("dropped %v", dropped)
	}

	s = NewSearcher(nil, []string{"spsearch"})
	if kept, _ := s.FilterSupported(nil); len(kept) != 1 {
		t.Fatalf("should never drop every provider, got %v", kept)
	}
}

func TestUnavailableSourcesAreRejected(t *testing.T) {
	s := NewSearcher(nil, []string{"ytsearch", "scsearch"})
	s.FilterSupported([]string{"soundcloud", "http"})

	var unavailable *SourceUnavailableError
	if _, err := s.providersFor("youtube"); !errors.As(err, &unavailable) || unavailable.Source != "youtube" {
		t.Fatalf("expected youtube to be unavailable, got %v", err)
	}
	if _, err := s.Resolve(context.Background(), "https://youtu.be/abc?si=x", ""); !errors.As(err, &unavailable) {
		t.Fatalf("expected youtube url to be rejected, got %v", err)
	}
	if p, err := s.providersFor("soundcloud"); err != nil || len(p) != 1 {
		t.Fatalf("soundcloud should be allowed, got %v %v", p, err)
	}
	if got := sourceOfURL("https://open.spotify.com/track/1"); got != "spotify" {
		t.Fatalf("got %q", got)
	}
}
