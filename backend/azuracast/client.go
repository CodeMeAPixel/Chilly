package azuracast

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("azuracast %s: %s: %s", path, resp.Status, body)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out)
}

func (c *Client) NowPlayingAll(ctx context.Context) ([]NowPlaying, error) {
	var out []NowPlaying
	if err := c.get(ctx, "/api/nowplaying", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) NowPlaying(ctx context.Context, station string) (*NowPlaying, error) {
	var out NowPlaying
	if err := c.get(ctx, "/api/nowplaying/"+url.PathEscape(station), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type NowPlaying struct {
	Station     Station        `json:"station"`
	Listeners   Listeners      `json:"listeners"`
	Live        Live           `json:"live"`
	NowPlaying  *CurrentSong   `json:"now_playing"`
	PlayingNext *StationQueue  `json:"playing_next"`
	SongHistory []HistoryEntry `json:"song_history"`
	IsOnline    bool           `json:"is_online"`
}

type Station struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Shortcode       string  `json:"shortcode"`
	Description     string  `json:"description"`
	ListenURL       string  `json:"listen_url"`
	URL             string  `json:"url"`
	PublicPlayerURL string  `json:"public_player_url"`
	IsPublic        bool    `json:"is_public"`
	Mounts          []Mount `json:"mounts"`
	HLSEnabled      bool    `json:"hls_enabled"`
	HLSURL          *string `json:"hls_url"`
}

type Mount struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Bitrate   int    `json:"bitrate"`
	Format    string `json:"format"`
	Path      string `json:"path"`
	IsDefault bool   `json:"is_default"`
}

type Listeners struct {
	Total   int `json:"total"`
	Unique  int `json:"unique"`
	Current int `json:"current"`
}

type Live struct {
	IsLive       bool    `json:"is_live"`
	StreamerName string  `json:"streamer_name"`
	Art          *string `json:"art"`
}

type Song struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Album  string `json:"album"`
	Genre  string `json:"genre"`
	Art    string `json:"art"`
}

type CurrentSong struct {
	ShID      int     `json:"sh_id"`
	PlayedAt  int64   `json:"played_at"`
	Duration  Seconds `json:"duration"`
	Playlist  string  `json:"playlist"`
	Streamer  string  `json:"streamer"`
	IsRequest bool    `json:"is_request"`
	Song      Song    `json:"song"`
	Elapsed   Seconds `json:"elapsed"`
	Remaining Seconds `json:"remaining"`
}

type StationQueue struct {
	CuedAt   int64   `json:"cued_at"`
	Duration Seconds `json:"duration"`
	Playlist string  `json:"playlist"`
	Song     Song    `json:"song"`
}

type HistoryEntry struct {
	ShID     int     `json:"sh_id"`
	PlayedAt int64   `json:"played_at"`
	Duration Seconds `json:"duration"`
	Playlist string  `json:"playlist"`
	Song     Song    `json:"song"`
}

type Seconds int

func (s *Seconds) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = 0
		return nil
	}
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*s = Seconds(math.Round(value))
	return nil
}
