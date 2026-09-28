package musicbot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/disgoorg/disgolink/v3/lavalink"
)

func TestQueryFromTrackCleansYouTubeTitles(t *testing.T) {
	cases := []struct {
		title, author, source string
		wantTitle, wantArtist string
	}{
		{"Joyner Lucas ft T-Pain - Hate Me (ADHD 2)", "Joyner Lucas", "youtube", "Hate Me (ADHD 2)", "Joyner Lucas"},
		{"The Weeknd - Blinding Lights (Official Video)", "TheWeekndVEVO", "youtube", "Blinding Lights", "The Weeknd"},
		{"Not Like Us", "Kendrick Lamar - Topic", "youtube", "Not Like Us", "Kendrick Lamar"},
		{"Sunset Lover [Official Audio]", "Petit Biscuit", "soundcloud", "Sunset Lover", "Petit Biscuit"},
		{"Young, Wild & Free (feat. Bruno Mars)", "Snoop Dogg", "spotify", "Young, Wild & Free", "Snoop Dogg"},
	}
	for _, c := range cases {
		q := QueryFromTrack(lavalink.Track{Info: lavalink.TrackInfo{Title: c.title, Author: c.author, SourceName: c.source, Length: 200_000}})
		if q.Title != c.wantTitle || q.Artist != c.wantArtist {
			t.Errorf("%q by %q: got title %q artist %q, want %q / %q", c.title, c.author, q.Title, q.Artist, c.wantTitle, c.wantArtist)
		}
	}
}

func TestParseLRC(t *testing.T) {
	lines := ParseLRC("[ar:ignored]\n[00:12.34] first line\n[01:02.5]second\n[02:03] \n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %+v", len(lines), lines)
	}
	if lines[0].TimeMs != 12340 || lines[0].Text != "first line" {
		t.Fatalf("unexpected first line %+v", lines[0])
	}
	if lines[1].TimeMs != 62500 || lines[1].Text != "second" {
		t.Fatalf("unexpected second line %+v", lines[1])
	}
}

func TestLookupFallsBackToSearchAndPicksClosestDuration(t *testing.T) {
	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		switch r.URL.Path {
		case "/api/get":
			http.NotFound(w, r)
		case "/api/search":
			_, _ = w.Write([]byte(`[
				{"trackName":"Song","artistName":"A","duration":300,"plainLyrics":"too long"},
				{"trackName":"Song","artistName":"A","duration":201,"plainLyrics":"plain","syncedLyrics":"[00:01.00] hello"}
			]`))
		}
	}))
	defer srv.Close()

	c := NewLyricsClient(srv.URL)
	lyrics, err := c.Lookup(context.Background(), SongQuery{Title: "Song", Artist: "A", DurationMs: 200_000})
	if err != nil {
		t.Fatal(err)
	}
	if lyrics.Plain != "plain" || len(lyrics.Synced) != 1 {
		t.Fatalf("picked wrong record: %+v", lyrics)
	}

	if _, err := c.Lookup(context.Background(), SongQuery{Title: "Song", Artist: "A", DurationMs: 200_000}); err != nil {
		t.Fatal(err)
	}
	if calls["/api/search"] != 1 {
		t.Fatalf("second lookup should be cached, search called %d times", calls["/api/search"])
	}
}
