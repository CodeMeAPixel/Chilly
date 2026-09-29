package musicbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
)

var ErrRadioDisabled = errors.New("radio is not enabled")

func (b *Bot) LoadRadioTrack(ctx context.Context, np azuracast.NowPlaying, meta TrackMeta) (lavalink.Track, error) {
	if b.Radio == nil {
		return lavalink.Track{}, ErrRadioDisabled
	}
	streamURL := b.Radio.StreamURL(np)
	if streamURL == "" {
		return lavalink.Track{}, errors.New("station has no stream url")
	}
	result, err := b.Searcher.LoadURL(ctx, streamURL)
	if err != nil {
		return lavalink.Track{}, err
	}
	track, ok := result.Data.(lavalink.Track)
	if !ok {
		return lavalink.Track{}, fmt.Errorf("unexpected load result %q for stream (is the lavalink http source enabled?)", result.LoadType)
	}

	track.Info.Title = np.Station.Name
	track.Info.Author = "Radio"
	track.Info.IsStream = true
	if np.Station.PublicPlayerURL != "" {
		track.Info.URI = json.Ptr(np.Station.PublicPlayerURL)
	}
	if np.NowPlaying != nil && np.NowPlaying.Song.Art != "" {
		track.Info.ArtworkURL = json.Ptr(np.NowPlaying.Song.Art)
	}
	meta.Radio = np.Station.Shortcode
	return WithTrackMeta([]lavalink.Track{track}, meta)[0], nil
}

var (
	ErrNotRadio       = errors.New("a station isn't playing right now")
	ErrNoOtherStation = errors.New("there's no other station on air to switch to")
)

func AdjacentStation(stations []azuracast.NowPlaying, current string, step int) (azuracast.NowPlaying, bool) {
	online := make([]azuracast.NowPlaying, 0, len(stations))
	index := -1
	for _, np := range stations {
		if np.Station.Shortcode == current {
			index = len(online)
			online = append(online, np)
			continue
		}
		if np.IsOnline {
			online = append(online, np)
		}
	}
	if index < 0 || len(online) < 2 || step == 0 {
		return azuracast.NowPlaying{}, false
	}
	next := ((index+step)%len(online) + len(online)) % len(online)
	return online[next], true
}

func (b *Bot) StepStation(ctx context.Context, guildID, userID snowflake.ID, step int, canManage bool) (azuracast.NowPlaying, error) {
	if b.Radio == nil {
		return azuracast.NowPlaying{}, ErrRadioDisabled
	}
	player, ok := b.PlayerManager.GetPlayer(guildID)
	if !ok {
		return azuracast.NowPlaying{}, ErrNotRadio
	}
	track, ok := player.Current()
	current := GetTrackMeta(track).Radio
	if !ok || current == "" {
		return azuracast.NowPlaying{}, ErrNotRadio
	}
	np, ok := AdjacentStation(b.Radio.Stations(), current, step)
	if !ok {
		return azuracast.NowPlaying{}, ErrNoOtherStation
	}

	next, err := b.LoadRadioTrack(ctx, np, TrackMeta{Requester: userID})
	if err != nil {
		return azuracast.NowPlaying{}, err
	}
	if err := player.PlayNow(ctx, next); err != nil {
		return azuracast.NowPlaying{}, err
	}

	if setting, enabled := b.Stays.Get(guildID); enabled && canManage && setting.Station != np.Station.Shortcode {
		setting.Station = np.Station.Shortcode
		if err := b.Stays.Set(ctx, setting); err != nil {
			slog.Warn("failed to move 24/7 radio to the new station", slog.String("guild_id", guildID.String()), slog.Any("error", err))
		}
	}
	return np, nil
}
