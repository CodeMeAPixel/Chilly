package musicbot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

type fakeFetcher map[string][]azuracast.MediaFile

func (f fakeFetcher) Files(_ context.Context, station string) ([]azuracast.MediaFile, error) {
	files, ok := f[station]
	if !ok {
		return nil, errors.New("station unavailable")
	}
	return files, nil
}

func media(id int, songID, artist, title, album string, playlists ...string) azuracast.MediaFile {
	f := azuracast.MediaFile{ID: id, SongID: songID, Artist: artist, Title: title, Album: album, Path: album + "/" + title + ".mp3", Length: 200}
	for _, p := range playlists {
		f.Playlists = append(f.Playlists, azuracast.MediaPlaylist{Name: p})
	}
	return f
}

func testLibrary(t *testing.T) *Library {
	t.Helper()
	fetcher := fakeFetcher{
		"chill": {
			media(1, "a", "Petit Biscuit", "Sunset Lover", "Presence", "Lowfi Jams"),
			media(2, "b", "Petit Biscuit", "Problems", "Presence"),
			media(3, "c", "M83", "Midnight City", "Hurry Up, We're Dreaming", "Night Drive"),
			{ID: 4, Path: "uploads/untitled-demo.mp3", Length: 90},
		},
		"hiphop": {
			media(10, "a", "Petit Biscuit", "Sunset Lover", "Presence", "Hip-Hop Rotation"),
			media(11, "d", "Joyner Lucas", "Devil's Work", "ADHD"),
		},
	}
	lib := NewLibrary(fetcher, func() []string { return []string{"chill", "hiphop"} })
	if err := lib.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	return lib
}

func TestLibrarySyncDedupesSongsAcrossStations(t *testing.T) {
	lib := testLibrary(t)
	count, lastSync, err := lib.Status()
	if err != nil || lastSync.IsZero() {
		t.Fatalf("status = %d, %v, %v", count, lastSync, err)
	}
	if count != 5 {
		t.Fatalf("got %d tracks, want 5 (one duplicate removed)", count)
	}
	track, ok := lib.Get("lib:chill:1")
	if !ok {
		t.Fatal("first copy of a duplicated song should be kept")
	}
	if len(track.Playlists) != 2 {
		t.Errorf("playlists from both stations should be merged, got %v", track.Playlists)
	}
	if _, ok := lib.Get("lib:hiphop:10"); ok {
		t.Error("duplicate from the second station should be dropped")
	}
	if untitled, _ := lib.Get("lib:chill:4"); untitled.Title != "untitled-demo" {
		t.Errorf("tracks without a title should fall back to the file name, got %q", untitled.Title)
	}
}

func TestLibrarySyncToleratesOneStationFailing(t *testing.T) {
	lib := NewLibrary(fakeFetcher{"chill": {media(1, "a", "A", "B", "C")}}, func() []string { return []string{"chill", "missing"} })
	if err := lib.Sync(context.Background()); err != nil {
		t.Fatalf("sync should succeed when some stations work: %v", err)
	}
	if _, _, err := lib.Status(); err == nil {
		t.Error("the failing station should still be reported")
	}

	broken := NewLibrary(fakeFetcher{}, func() []string { return []string{"missing"} })
	if err := broken.Sync(context.Background()); err == nil {
		t.Error("sync should fail when every station fails")
	}
	if broken.Ready() {
		t.Error("a library that never synced must not be ready")
	}
}

func TestLibrarySearchRanksTitleMatchesFirst(t *testing.T) {
	lib := testLibrary(t)

	results := lib.Search("sunset lover", 5)
	if len(results) == 0 || results[0].Title != "Sunset Lover" {
		t.Fatalf("exact title should rank first, got %+v", results)
	}
	if got := lib.Search("petit biscuit problems", 5); len(got) != 1 || got[0].Title != "Problems" {
		t.Errorf("artist + title query should narrow to one song, got %+v", got)
	}
	if got := lib.Search("PETIT", 5); len(got) != 2 {
		t.Errorf("search should be case-insensitive and match artists, got %d", len(got))
	}
	if got := lib.Search("nothing like this", 5); len(got) != 0 {
		t.Errorf("unexpected matches %+v", got)
	}
	if got := lib.Search("   ", 5); got != nil {
		t.Error("blank queries should return nothing")
	}
	if got := lib.Search("e", 2); len(got) != 2 {
		t.Errorf("limit ignored, got %d", len(got))
	}
}

func TestLibraryGroups(t *testing.T) {
	lib := testLibrary(t)

	name, tracks := lib.Group(GroupAlbum, "presence")
	if name != "Presence" || len(tracks) != 2 {
		t.Fatalf("album = %q with %d tracks", name, len(tracks))
	}
	if tracks[0].Title != "Problems" || tracks[1].Title != "Sunset Lover" {
		t.Errorf("album tracks should follow file order, got %s, %s", tracks[0].Title, tracks[1].Title)
	}
	if name, tracks := lib.Group(GroupArtist, "m83"); name != "M83" || len(tracks) != 1 {
		t.Errorf("artist = %q with %d tracks", name, len(tracks))
	}
	if name, tracks := lib.Group(GroupPlaylist, "hip-hop"); name != "Hip-Hop Rotation" || len(tracks) != 1 {
		t.Errorf("playlist from the deduped station = %q with %d tracks", name, len(tracks))
	}
	if name, tracks := lib.Group(GroupAlbum, "no such album"); name != "" || tracks != nil {
		t.Error("unknown groups should return nothing")
	}
}

func TestParseLibraryKey(t *testing.T) {
	track := LibraryTrack{Station: "247_hip-hop", ID: 42}
	station, id, ok := ParseLibraryKey(track.Key())
	if !ok || station != "247_hip-hop" || id != 42 {
		t.Fatalf("round trip failed: %q %d %v", station, id, ok)
	}
	for _, bad := range []string{"", "lib:", "lib:station", "lib::3", "lib:x:abc", "lib:x:0", "https://youtu.be/x"} {
		if _, _, ok := ParseLibraryKey(bad); ok {
			t.Errorf("%q should not parse", bad)
		}
	}
	if len(track.Key()) > 100 {
		t.Error("keys must fit in a Discord autocomplete value")
	}
}

func TestMediaSigner(t *testing.T) {
	signer := NewMediaSigner("secret", "https://api.example.com/")
	link := signer.URL("chill", 7)
	if !strings.HasPrefix(link, "https://api.example.com/api/v1/media/chill/7?sig=") {
		t.Fatalf("unexpected url %q", link)
	}
	sig := link[strings.Index(link, "sig=")+4:]
	if !signer.Verify("chill", 7, sig) {
		t.Error("a signature must verify for its own track")
	}
	if signer.Verify("chill", 8, sig) || signer.Verify("hiphop", 7, sig) || signer.Verify("chill", 7, "") {
		t.Error("signatures must not verify for other tracks")
	}
	if NewMediaSigner("other", "").Verify("chill", 7, sig) {
		t.Error("signatures must depend on the secret")
	}
}

func TestResolveRejectsExternalLinks(t *testing.T) {
	s := NewSearcher(nil, testLibrary(t), NewMediaSigner("k", "http://x"))
	if _, err := s.Resolve(context.Background(), "https://www.youtube.com/watch?v=abc", ""); !errors.Is(err, ErrExternalSource) {
		t.Errorf("got %v, want ErrExternalSource", err)
	}
	if _, err := s.Resolve(context.Background(), "lib:chill:999", ""); !errors.Is(err, ErrSelectionExpired) {
		t.Errorf("got %v, want ErrSelectionExpired", err)
	}
	if _, err := NewSearcher(nil, nil, nil).Resolve(context.Background(), "anything", ""); !errors.Is(err, ErrLibraryUnavailable) {
		t.Errorf("got %v, want ErrLibraryUnavailable", err)
	}
}

func TestWithLibraryInfoHidesSignedURL(t *testing.T) {
	uri := "https://api.example.com/api/v1/media/chill/1?sig=secret"
	track := lavalink.Track{Info: lavalink.TrackInfo{Title: "file.mp3", Identifier: uri, URI: &uri, SourceName: "http"}}
	lt := LibraryTrack{Station: "chill", ID: 1, Title: "Sunset Lover", Artist: "Petit Biscuit", ArtURL: "https://radio/art.jpg"}

	got := withLibraryInfo(track, lt)
	if got.Info.URI != nil || strings.Contains(got.Info.Identifier, "sig=") {
		t.Fatal("the signed media URL must not be exposed in track info")
	}
	if got.Info.Title != "Sunset Lover" || got.Info.Author != "Petit Biscuit" || got.Info.SourceName != LibrarySource {
		t.Errorf("unexpected info %+v", got.Info)
	}
	if got.Info.Identifier != "lib:chill:1" || TrackArtwork(got) != "https://radio/art.jpg" {
		t.Errorf("identifier or artwork wrong: %q %q", got.Info.Identifier, TrackArtwork(got))
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

func TestStoredLyrics(t *testing.T) {
	if (LibraryTrack{Lyrics: "   "}).StoredLyrics() != nil {
		t.Error("blank lyrics should be treated as missing")
	}

	plain := LibraryTrack{Title: "Song", Artist: "Artist", Lyrics: "line one\nline two"}.StoredLyrics()
	if plain == nil || plain.Plain != "line one\nline two" || len(plain.Synced) != 0 || plain.Source != "Chilly Library" {
		t.Fatalf("unexpected plain lyrics %+v", plain)
	}

	synced := LibraryTrack{Lyrics: "[00:01.00]first\n[00:02.50]second"}.StoredLyrics()
	if synced == nil || len(synced.Synced) != 2 || synced.Plain != "first\nsecond" {
		t.Fatalf("unexpected synced lyrics %+v", synced)
	}
}

func TestLibraryMatchLegacyTracks(t *testing.T) {
	lib := testLibrary(t)
	youtube := func(title, author string, lengthMs int64) SongQuery {
		return QueryFromTrack(lavalink.Track{Info: lavalink.TrackInfo{Title: title, Author: author, SourceName: "youtube", Length: lavalink.Duration(lengthMs)}})
	}

	if m, ok := lib.Match(youtube("Petit Biscuit - Sunset Lover (Official Video)", "Petit Biscuit", 237000)); !ok || m.Title != "Sunset Lover" {
		t.Errorf("official video title should match, got %+v %v", m, ok)
	}
	if m, ok := lib.Match(youtube("M83 - Midnight City [HD]", "M83VEVO", 243000)); !ok || m.Artist != "M83" {
		t.Errorf("artist from the title should match, got %+v %v", m, ok)
	}
	if _, ok := lib.Match(youtube("Someone Else - Problems", "Someone Else", 0)); ok {
		t.Error("same title by a different artist must not match without a close length")
	}
	if m, ok := lib.Match(SongQuery{Title: "Problems", Artist: "Unknown Uploader", DurationMs: 205000}); !ok || m.Title != "Problems" {
		t.Errorf("a unique title with a close length should match, got %+v %v", m, ok)
	}
	if _, ok := lib.Match(SongQuery{Title: "Not In Library", Artist: "Nobody"}); ok {
		t.Error("unknown songs must not match")
	}
}

func TestLibraryPlaylists(t *testing.T) {
	lib := testLibrary(t)
	playlists := lib.Playlists()
	if len(playlists) != 3 {
		t.Fatalf("got %d playlists, want 3: %+v", len(playlists), playlists)
	}
	if playlists[0].Name != "Hip-Hop Rotation" || playlists[1].Name != "Lowfi Jams" || playlists[2].Name != "Night Drive" {
		t.Errorf("playlists should be sorted by name: %+v", playlists)
	}
	if tracks := lib.PlaylistTracks("lowfi jams"); len(tracks) != 1 || tracks[0].Title != "Sunset Lover" {
		t.Errorf("playlist lookup should ignore case, got %+v", tracks)
	}
	if tracks := lib.PlaylistTracks("missing"); len(tracks) != 0 {
		t.Errorf("unknown playlist returned %+v", tracks)
	}
}

func TestLibraryBrowseAndGroups(t *testing.T) {
	lib := testLibrary(t)

	page, total := lib.Browse(LibraryQuery{Limit: 2})
	if total != 5 || len(page) != 2 {
		t.Fatalf("got %d of %d", len(page), total)
	}
	next, _ := lib.Browse(LibraryQuery{Offset: 4, Limit: 2})
	if len(next) != 1 {
		t.Errorf("last page should have 1 track, got %d", len(next))
	}
	if beyond, total := lib.Browse(LibraryQuery{Offset: 50, Limit: 10}); len(beyond) != 0 || total != 5 {
		t.Errorf("offset past the end: %d of %d", len(beyond), total)
	}
	if byArtist, total := lib.Browse(LibraryQuery{Artist: "petit biscuit"}); total != 2 || len(byArtist) != 2 {
		t.Errorf("artist filter: %d", total)
	}
	if byPlaylist, total := lib.Browse(LibraryQuery{Playlist: "night drive"}); total != 1 || byPlaylist[0].Title != "Midnight City" {
		t.Errorf("playlist filter: %+v", byPlaylist)
	}
	if searched, total := lib.Browse(LibraryQuery{Query: "sunset", Album: "Presence"}); total != 1 || searched[0].Title != "Sunset Lover" {
		t.Errorf("search with album filter: %+v", searched)
	}
	byTitle, _ := lib.Browse(LibraryQuery{Sort: "title"})
	if byTitle[0].Title != "Devil's Work" {
		t.Errorf("title sort should start with Devil's Work, got %q", byTitle[0].Title)
	}

	artists := lib.Groups(GroupArtist)
	if len(artists) != 3 || artists[0].Name != "Joyner Lucas" || artists[2].TrackCount != 2 {
		t.Errorf("unexpected artists %+v", artists)
	}
	albums := lib.Groups(GroupAlbum)
	if len(albums) != 3 || albums[2].Name != "Presence" || albums[2].Artist != "Petit Biscuit" {
		t.Errorf("unexpected albums %+v", albums)
	}
}

func TestLibraryHidesFolders(t *testing.T) {
	file := func(id int, path string) azuracast.MediaFile {
		return azuracast.MediaFile{ID: id, SongID: path, Title: path, Artist: "A", Path: path, Length: 100}
	}
	lib := NewLibrary(fakeFetcher{"chill": {
		file(1, "Random Mixes/night.mp3"),
		file(2, "random mixes/deep/long.mp3"),
		file(3, "Random Mixes 2/kept.mp3"),
		file(4, "Other/Random Mixes.mp3"),
		file(5, "Albums/song.mp3"),
	}}, func() []string { return []string{"chill"} })
	lib.HideFolders([]string{" /Random Mixes/ ", ""})

	if err := lib.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	var kept []int
	for _, track := range lib.All() {
		kept = append(kept, track.ID)
	}
	if len(kept) != 3 || kept[0] != 3 || kept[1] != 4 || kept[2] != 5 {
		t.Errorf("expected tracks 3, 4 and 5 to remain, got %v", kept)
	}

	lib.HideFolders(nil)
	if err := lib.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(lib.All()) != 5 {
		t.Error("clearing hidden folders should bring every track back")
	}
}

func TestLoadFailureCause(t *testing.T) {
	trace := strings.Join([]string{
		"com.sedmelluq.discord.lavaplayer.tools.FriendlyException: Something went wrong while looking up the track.",
		"\tat lavalink.server.util.LoadingKt.loadAudioItem(loading.kt:20)",
		"Caused by: java.lang.RuntimeException: Unknown file format.",
		"\tat com.sedmelluq.Foo(Foo.java:1)",
		"Caused by: java.io.IOException: Invalid status code 403",
	}, "\n")
	ex := lavalink.Exception{Message: "Something went wrong while looking up the track.", CauseStackTrace: trace}
	want := "java.lang.RuntimeException: Unknown file format. <- java.io.IOException: Invalid status code 403"
	if got := LoadFailureCause(ex); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := LoadFailureCause(lavalink.Exception{Message: "boom", Cause: "java.net.SocketTimeoutException: Read timed out"}); got != "java.net.SocketTimeoutException: Read timed out" {
		t.Errorf("plain cause: got %q", got)
	}
	if got := LoadFailureCause(lavalink.Exception{Message: "boom", Cause: "FriendlyException: boom"}); got != "" {
		t.Errorf("redundant cause: got %q", got)
	}
}
