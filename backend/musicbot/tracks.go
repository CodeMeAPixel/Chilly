package musicbot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type TrackMeta struct {
	Requester    snowflake.ID `json:"requester"`
	PlaylistName string       `json:"playlistName,omitempty"`
	PlaylistURL  string       `json:"playlistUrl,omitempty"`

	Radio    string `json:"radio,omitempty"`
	Fallback bool   `json:"fallback,omitempty"`
	PlayID   string `json:"playId,omitempty"`
}

func GetTrackMeta(track lavalink.Track) TrackMeta {
	var meta TrackMeta
	if len(track.UserData) > 0 {
		_ = track.UserData.Unmarshal(&meta)
	}
	return meta
}

func WithTrackMeta(tracks []lavalink.Track, meta TrackMeta) []lavalink.Track {
	data, err := json.Marshal(meta)
	if err != nil {
		return tracks
	}
	raw := lavalink.RawData(data)
	out := make([]lavalink.Track, len(tracks))
	for i, track := range tracks {
		track.UserData = raw
		out[i] = track
	}
	return out
}

func TrackURL(track lavalink.Track) string {
	if track.Info.URI == nil {
		return ""
	}
	return *track.Info.URI
}

func TrackArtwork(track lavalink.Track) string {
	if track.Info.ArtworkURL == nil {
		return ""
	}
	return *track.Info.ArtworkURL
}

var markdownEscaper = strings.NewReplacer(
	"[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "`", "\\`", "~", "\\~", "|", "\\|",
)

func EscapeMarkdown(s string) string {
	return markdownEscaper.Replace(s)
}

func TrackLink(track lavalink.Track) string {
	title := EscapeMarkdown(Trim(TrackTitle(track), 90))
	if url := TrackURL(track); url != "" {
		return fmt.Sprintf("[%s](%s)", title, url)
	}
	return title
}

func TrackTitle(track lavalink.Track) string {
	if strings.TrimSpace(track.Info.Title) == "" {
		return "Unknown title"
	}
	return track.Info.Title
}

func TrackDuration(track lavalink.Track) string {
	if track.Info.IsStream {
		return "LIVE"
	}
	return FormatTime(track.Info.Length)
}

type TrackView struct {
	Title        string `json:"title"`
	Author       string `json:"author"`
	URI          string `json:"uri,omitempty"`
	ArtworkURL   string `json:"artwork_url,omitempty"`
	SourceName   string `json:"source"`
	Identifier   string `json:"identifier"`
	LengthMs     int64  `json:"length_ms"`
	IsStream     bool   `json:"is_stream"`
	Requester    string `json:"requester_id,omitempty"`
	PlaylistName string `json:"playlist_name,omitempty"`
	Radio        string `json:"radio_station,omitempty"`
}

func NewTrackView(track lavalink.Track) TrackView {
	meta := GetTrackMeta(track)
	view := TrackView{
		Title:        TrackTitle(track),
		Author:       track.Info.Author,
		URI:          TrackURL(track),
		ArtworkURL:   TrackArtwork(track),
		SourceName:   track.Info.SourceName,
		Identifier:   track.Info.Identifier,
		LengthMs:     int64(track.Info.Length),
		IsStream:     track.Info.IsStream,
		PlaylistName: meta.PlaylistName,
		Radio:        meta.Radio,
	}
	if meta.Requester != 0 {
		view.Requester = meta.Requester.String()
	}
	return view
}

func NewTrackViews(tracks []lavalink.Track) []TrackView {
	views := make([]TrackView, len(tracks))
	for i, track := range tracks {
		views[i] = NewTrackView(track)
	}
	return views
}

type PlayerSnapshot struct {
	GuildID        string      `json:"guild_id"`
	VoiceChannelID string      `json:"voice_channel_id,omitempty"`
	TextChannelID  string      `json:"text_channel_id,omitempty"`
	Playing        bool        `json:"playing"`
	Paused         bool        `json:"paused"`
	PositionMs     int64       `json:"position_ms"`
	Volume         int         `json:"volume"`
	Loop           LoopMode    `json:"loop"`
	Shuffle        bool        `json:"shuffle"`
	Current        *TrackView  `json:"current"`
	Queue          []TrackView `json:"queue"`
	QueueLength    int         `json:"queue_length"`
	History        []TrackView `json:"history"`
	SessionStart   time.Time   `json:"session_start"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

var playCounter atomic.Uint64

func stampPlay(track lavalink.Track) lavalink.Track {
	meta := GetTrackMeta(track)
	meta.PlayID = strconv.FormatUint(playCounter.Add(1), 36)
	return WithTrackMeta([]lavalink.Track{track}, meta)[0]
}

func SameTrack(a, b lavalink.Track) bool {
	if idA, idB := GetTrackMeta(a).PlayID, GetTrackMeta(b).PlayID; idA != "" && idB != "" {
		return idA == idB
	}
	if a.Encoded != "" && a.Encoded == b.Encoded {
		return true
	}
	return a.Info.Identifier != "" && a.Info.Identifier == b.Info.Identifier && a.Info.SourceName == b.Info.SourceName
}
