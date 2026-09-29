package musicbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

const (
	RequestQueued  = "queued"
	RequestPlaying = "playing"
	RequestPlayed  = "played"
	RequestExpired = "expired"

	SuggestionPending   = "pending"
	SuggestionReviewing = "reviewing"
	SuggestionAdded     = "added"
	SuggestionDeclined  = "declined"

	requestExpiry = 12 * time.Hour
)

var (
	ErrRequestsUnavailable = errors.New("song requests aren't available right now")
	ErrSuggestionNotFound  = errors.New("suggestion not found")
	ErrInvalidStatus       = errors.New("status must be pending, reviewing, added or declined")
)

type LimitError struct {
	Message string
}

func (e *LimitError) Error() string {
	return e.Message
}

type AlreadyInLibraryError struct {
	Track LibraryTrack
}

func (e *AlreadyInLibraryError) Error() string {
	return fmt.Sprintf("%s by %s is already in the library", e.Track.Title, e.Track.Artist)
}

type SongRequest struct {
	ID        int64        `json:"id"`
	UserID    snowflake.ID `json:"user_id"`
	GuildID   snowflake.ID `json:"guild_id,omitempty"`
	Station   string       `json:"station"`
	MediaKey  string       `json:"media_key"`
	SongID    string       `json:"-"`
	Title     string       `json:"title"`
	Artist    string       `json:"artist"`
	Art       string       `json:"art,omitempty"`
	Source    string       `json:"source"`
	Status    string       `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	PlayedAt  *time.Time   `json:"played_at,omitempty"`
}

type SongSuggestion struct {
	ID         int64        `json:"id"`
	UserID     snowflake.ID `json:"user_id"`
	Artist     string       `json:"artist"`
	Title      string       `json:"title"`
	Link       string       `json:"link,omitempty"`
	Note       string       `json:"note,omitempty"`
	Status     string       `json:"status"`
	Reason     string       `json:"reason,omitempty"`
	LibraryKey string       `json:"library_key,omitempty"`
	ReviewedBy snowflake.ID `json:"reviewed_by,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

type RequestStore interface {
	LastRequestAt(ctx context.Context, userID snowflake.ID) (time.Time, error)
	CountOpenRequests(ctx context.Context, userID snowflake.ID) (int, error)
	InsertRequest(ctx context.Context, r *SongRequest) error
	AdvanceRequests(ctx context.Context, station, songID string) error
	ExpireRequests(ctx context.Context, before time.Time) error
	UserRequests(ctx context.Context, userID snowflake.ID, limit int) ([]SongRequest, error)
	RecentRequests(ctx context.Context, limit int) ([]SongRequest, error)

	CountOpenSuggestions(ctx context.Context, userID snowflake.ID) (int, error)
	InsertSuggestion(ctx context.Context, s *SongSuggestion) error
	UserSuggestions(ctx context.Context, userID snowflake.ID, limit int) ([]SongSuggestion, error)
	ListSuggestions(ctx context.Context, status string, limit int) ([]SongSuggestion, error)
	GetSuggestion(ctx context.Context, id int64) (SongSuggestion, error)
	UpdateSuggestion(ctx context.Context, s SongSuggestion) error
	DeleteOpenSuggestion(ctx context.Context, id int64, userID snowflake.ID) (bool, error)
}

type RequestSubmitter interface {
	SubmitRequest(ctx context.Context, station, uniqueID string) error
}

type RequestConfig struct {
	Cooldown           time.Duration `yaml:"cooldown"`
	MaxPending         int           `yaml:"max_pending"`
	MaxOpenSuggestions int           `yaml:"max_open_suggestions"`
}

type Requests struct {
	cfg       RequestConfig
	store     RequestStore
	submitter RequestSubmitter
	library   *Library
	notify    func(ctx context.Context, userID snowflake.ID, message string) error
	siteURL   func(path string) string
	now       func() time.Time
}

func NewRequests(cfg RequestConfig, store RequestStore, submitter RequestSubmitter, library *Library,
	notify func(ctx context.Context, userID snowflake.ID, message string) error, siteURL func(string) string) *Requests {
	if siteURL == nil {
		siteURL = func(path string) string { return path }
	}
	return &Requests{cfg: cfg, store: store, submitter: submitter, library: library, notify: notify, siteURL: siteURL, now: time.Now}
}

func (r *Requests) Config() RequestConfig {
	return r.cfg
}

func (r *Requests) Submit(ctx context.Context, userID, guildID snowflake.ID, key, source string) (SongRequest, error) {
	if r == nil || r.library == nil || !r.library.Ready() {
		return SongRequest{}, ErrRequestsUnavailable
	}
	track, ok := r.library.Get(key)
	if !ok {
		return SongRequest{}, ErrSelectionExpired
	}

	if r.cfg.Cooldown > 0 {
		last, err := r.store.LastRequestAt(ctx, userID)
		if err != nil {
			return SongRequest{}, err
		}
		if wait := r.cfg.Cooldown - r.now().Sub(last); !last.IsZero() && wait > 0 {
			return SongRequest{}, &LimitError{Message: fmt.Sprintf("You can request another song in %s.", formatWait(wait))}
		}
	}
	if r.cfg.MaxPending > 0 {
		open, err := r.store.CountOpenRequests(ctx, userID)
		if err != nil {
			return SongRequest{}, err
		}
		if open >= r.cfg.MaxPending {
			return SongRequest{}, &LimitError{Message: fmt.Sprintf("You already have %d requests waiting. Try again once one has played.", open)}
		}
	}

	if err := r.submitter.SubmitRequest(ctx, track.Station, track.UniqueID); err != nil {
		return SongRequest{}, err
	}

	req := SongRequest{
		UserID:   userID,
		GuildID:  guildID,
		Station:  track.Station,
		MediaKey: track.Key(),
		SongID:   track.SongID,
		Title:    Trim(track.Title, 255),
		Artist:   Trim(track.Artist, 255),
		Art:      track.ArtURL,
		Source:   source,
		Status:   RequestQueued,
	}
	if err := r.store.InsertRequest(ctx, &req); err != nil {
		slog.Error("request was sent to AzuraCast but could not be saved", slog.Any("error", err), slog.String("key", key))
	}
	return req, nil
}

func (r *Requests) OnSongChange(ctx context.Context, station, songID string) {
	if err := r.store.AdvanceRequests(ctx, station, songID); err != nil {
		slog.Warn("failed to update request statuses", slog.String("station", station), slog.Any("error", err))
	}
	if err := r.store.ExpireRequests(ctx, r.now().Add(-requestExpiry)); err != nil {
		slog.Warn("failed to expire old requests", slog.Any("error", err))
	}
}

func (r *Requests) Suggest(ctx context.Context, userID snowflake.ID, artist, title, link, note string) (SongSuggestion, error) {
	artist, title = strings.TrimSpace(artist), strings.TrimSpace(title)
	link, note = strings.TrimSpace(link), strings.TrimSpace(note)
	switch {
	case artist == "" || title == "":
		return SongSuggestion{}, &LimitError{Message: "Tell us the artist and the song title."}
	case len(artist) > 200 || len(title) > 200:
		return SongSuggestion{}, &LimitError{Message: "Artist and title must be 200 characters or fewer."}
	case len(note) > 500:
		return SongSuggestion{}, &LimitError{Message: "Notes must be 500 characters or fewer."}
	case link != "" && (!IsURL(link) || len(link) > 500):
		return SongSuggestion{}, &LimitError{Message: "The link must be a full http(s) URL."}
	}

	if r.library != nil {
		if match, ok := r.library.Match(SongQuery{Title: title, Artist: artist}); ok {
			return SongSuggestion{}, &AlreadyInLibraryError{Track: match}
		}
	}
	if r.cfg.MaxOpenSuggestions > 0 {
		open, err := r.store.CountOpenSuggestions(ctx, userID)
		if err != nil {
			return SongSuggestion{}, err
		}
		if open >= r.cfg.MaxOpenSuggestions {
			return SongSuggestion{}, &LimitError{Message: fmt.Sprintf("You have %d suggestions waiting for review. Please wait until some are reviewed.", open)}
		}
	}

	s := SongSuggestion{UserID: userID, Artist: artist, Title: title, Link: link, Note: note, Status: SuggestionPending}
	if err := r.store.InsertSuggestion(ctx, &s); err != nil {
		return SongSuggestion{}, err
	}
	return s, nil
}

func (r *Requests) Withdraw(ctx context.Context, id int64, userID snowflake.ID) error {
	deleted, err := r.store.DeleteOpenSuggestion(ctx, id, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrSuggestionNotFound
	}
	return nil
}

func (r *Requests) Review(ctx context.Context, id int64, status, reason, libraryKey string, reviewer snowflake.ID) (SongSuggestion, error) {
	switch status {
	case SuggestionPending, SuggestionReviewing, SuggestionAdded, SuggestionDeclined:
	default:
		return SongSuggestion{}, ErrInvalidStatus
	}
	s, err := r.store.GetSuggestion(ctx, id)
	if err != nil {
		return SongSuggestion{}, err
	}
	previous := s.Status
	s.Status = status
	s.Reason = strings.TrimSpace(reason)
	s.ReviewedBy = reviewer
	if libraryKey != "" {
		s.LibraryKey = libraryKey
	}
	if err := r.store.UpdateSuggestion(ctx, s); err != nil {
		return SongSuggestion{}, err
	}
	if previous != status {
		r.notifySuggestion(ctx, s)
	}
	return s, nil
}

func (r *Requests) MatchSuggestions(ctx context.Context) (int, error) {
	if r.library == nil {
		return 0, nil
	}
	matched := 0
	for _, status := range []string{SuggestionPending, SuggestionReviewing} {
		open, err := r.store.ListSuggestions(ctx, status, 500)
		if err != nil {
			return matched, err
		}
		for _, s := range open {
			track, ok := r.library.Match(SongQuery{Title: s.Title, Artist: s.Artist})
			if !ok {
				continue
			}
			s.Status = SuggestionAdded
			s.LibraryKey = track.Key()
			if err := r.store.UpdateSuggestion(ctx, s); err != nil {
				return matched, err
			}
			matched++
			r.notifySuggestion(ctx, s)
		}
	}
	return matched, nil
}

func (r *Requests) notifySuggestion(ctx context.Context, s SongSuggestion) {
	if r.notify == nil {
		return
	}
	song := fmt.Sprintf("**%s** by %s", EscapeMarkdown(s.Title), EscapeMarkdown(s.Artist))
	var message string
	switch s.Status {
	case SuggestionAdded:
		message = fmt.Sprintf("🎉 Good news! %s is now in the Chilly library. Play it with `/play`, or request it on a station: %s", song, r.siteURL("/tracks"))
	case SuggestionDeclined:
		message = fmt.Sprintf("Thanks for suggesting %s. It won't be added to the library this time.", song)
		if s.Reason != "" {
			message += "\n> " + EscapeMarkdown(s.Reason)
		}
	default:
		return
	}
	message += fmt.Sprintf("\n\nSee all your requests at %s", r.siteURL("/dashboard/requests"))
	if err := r.notify(ctx, s.UserID, message); err != nil {
		slog.Info("could not DM suggestion update", slog.String("user_id", s.UserID.String()), slog.Any("error", err))
	}
}

func formatWait(d time.Duration) string {
	if d < time.Minute {
		secs := int(d.Seconds()) + 1
		return fmt.Sprintf("%d second%s", secs, plural(secs))
	}
	mins := int(d.Minutes() + 0.999)
	return fmt.Sprintf("%d minute%s", mins, plural(mins))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func (b *Bot) SendDM(ctx context.Context, userID snowflake.ID, message string) error {
	channel, err := b.Client.Rest().CreateDMChannel(userID, rest.WithCtx(ctx))
	if err != nil {
		return err
	}
	_, err = b.Client.Rest().CreateMessage(channel.ID(), discord.MessageCreate{Content: message}, rest.WithCtx(ctx))
	return err
}
