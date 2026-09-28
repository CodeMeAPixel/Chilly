package musicbot

import (
	"context"
	"errors"
	"fmt"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
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
	result, err := b.Searcher.Resolve(ctx, streamURL, "")
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
