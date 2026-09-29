package musicbot

import (
	"testing"

	"github.com/CodeMeAPixel/Chilly/azuracast"
)

func station(code string, online bool) azuracast.NowPlaying {
	return azuracast.NowPlaying{Station: azuracast.Station{Shortcode: code}, IsOnline: online}
}

func TestAdjacentStation(t *testing.T) {
	stations := []azuracast.NowPlaying{station("a", true), station("b", false), station("c", true), station("d", true)}

	cases := []struct {
		current string
		step    int
		want    string
	}{
		{"a", 1, "c"},
		{"c", 1, "d"},
		{"d", 1, "a"},
		{"a", -1, "d"},
		{"c", -1, "a"},
		{"b", 1, "c"},
	}
	for _, c := range cases {
		got, ok := AdjacentStation(stations, c.current, c.step)
		if !ok || got.Station.Shortcode != c.want {
			t.Errorf("AdjacentStation(%s, %d) = %q %v, want %q", c.current, c.step, got.Station.Shortcode, ok, c.want)
		}
	}
	if _, ok := AdjacentStation(stations, "missing", 1); ok {
		t.Error("unknown current station should not resolve")
	}
	if _, ok := AdjacentStation([]azuracast.NowPlaying{station("a", true), station("b", false)}, "a", 1); ok {
		t.Error("with no other station online there is nothing to switch to")
	}
}
