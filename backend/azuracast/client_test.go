package azuracast

import (
	"encoding/json"
	"testing"
)

func TestNowPlayingAcceptsFractionalDurations(t *testing.T) {
	payload := `{
		"now_playing": {"duration": 212.4, "elapsed": 30, "remaining": 182.6, "song": {"title": "a"}},
		"playing_next": {"cued_at": 1790000000, "duration": 5379.7920206649, "song": {"title": "b"}},
		"song_history": [{"played_at": 1789999000, "duration": null, "song": {"title": "c"}}]
	}`

	var np NowPlaying
	if err := json.Unmarshal([]byte(payload), &np); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if np.NowPlaying.Duration != 212 || np.NowPlaying.Elapsed != 30 || np.NowPlaying.Remaining != 183 {
		t.Errorf("now playing = %d/%d/%d, want 212/30/183", np.NowPlaying.Duration, np.NowPlaying.Elapsed, np.NowPlaying.Remaining)
	}
	if np.PlayingNext.Duration != 5380 {
		t.Errorf("playing next duration = %d, want 5380", np.PlayingNext.Duration)
	}
	if np.SongHistory[0].Duration != 0 {
		t.Errorf("history duration = %d, want 0", np.SongHistory[0].Duration)
	}
}

func TestSecondsRejectsNonNumbers(t *testing.T) {
	var s Seconds
	if err := json.Unmarshal([]byte(`"12"`), &s); err == nil {
		t.Error("expected an error for a string value")
	}
}
