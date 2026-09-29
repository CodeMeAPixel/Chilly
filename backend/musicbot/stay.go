package musicbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

const (
	stayFirstCheck    = 10 * time.Second
	stayCheckInterval = 30 * time.Second
	stayBaseBackoff   = 30 * time.Second
	stayMaxBackoff    = 30 * time.Minute
)

var ErrStayEnabled = errors.New("24/7 radio is on in this server")

type StaySetting struct {
	GuildID        snowflake.ID `json:"guild_id"`
	VoiceChannelID snowflake.ID `json:"voice_channel_id"`
	TextChannelID  snowflake.ID `json:"text_channel_id,omitempty"`
	Station        string       `json:"station"`
	EnabledBy      snowflake.ID `json:"enabled_by"`
	EnabledAt      time.Time    `json:"enabled_at"`
}

type StayHealth struct {
	Failures  int       `json:"failures"`
	LastError string    `json:"last_error,omitempty"`
	RetryAt   time.Time `json:"retry_at,omitempty"`
}

type StayStore struct {
	db       *DB
	mu       sync.RWMutex
	settings map[snowflake.ID]StaySetting
	health   map[snowflake.ID]StayHealth
}

func NewStayStore(db *DB) *StayStore {
	return &StayStore{
		db:       db,
		settings: make(map[snowflake.ID]StaySetting),
		health:   make(map[snowflake.ID]StayHealth),
	}
}

func (s *StayStore) Load(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.Pool.Query(ctx,
		"SELECT guild_id, voice_channel_id, COALESCE(text_channel_id, 0), station, enabled_by, enabled_at FROM radio_stays")
	if err != nil {
		return err
	}
	defer rows.Close()

	loaded := make(map[snowflake.ID]StaySetting)
	for rows.Next() {
		var (
			guildID, voiceID, textID, enabledBy int64
			setting                             StaySetting
		)
		if err := rows.Scan(&guildID, &voiceID, &textID, &setting.Station, &enabledBy, &setting.EnabledAt); err != nil {
			return err
		}
		setting.GuildID = snowflake.ID(guildID)
		setting.VoiceChannelID = snowflake.ID(voiceID)
		setting.TextChannelID = snowflake.ID(textID)
		setting.EnabledBy = snowflake.ID(enabledBy)
		loaded[setting.GuildID] = setting
	}
	if err := rows.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.settings = loaded
	s.mu.Unlock()
	return nil
}

func (s *StayStore) Get(guildID snowflake.ID) (StaySetting, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	setting, ok := s.settings[guildID]
	return setting, ok
}

func (s *StayStore) Enabled(guildID snowflake.ID) bool {
	_, ok := s.Get(guildID)
	return ok
}

func (s *StayStore) All() []StaySetting {
	s.mu.RLock()
	out := make([]StaySetting, 0, len(s.settings))
	for _, setting := range s.settings {
		out = append(out, setting)
	}
	s.mu.RUnlock()
	slices.SortFunc(out, func(a, b StaySetting) int { return a.EnabledAt.Compare(b.EnabledAt) })
	return out
}

func (s *StayStore) Health(guildID snowflake.ID) StayHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.health[guildID]
}

func (s *StayStore) Set(ctx context.Context, setting StaySetting) error {
	if setting.EnabledAt.IsZero() {
		setting.EnabledAt = time.Now()
	}
	if s.db != nil {
		var textID *int64
		if setting.TextChannelID != 0 {
			id := int64(setting.TextChannelID)
			textID = &id
		}
		if _, err := s.db.Pool.Exec(ctx, `
			INSERT INTO radio_stays (guild_id, voice_channel_id, text_channel_id, station, enabled_by, enabled_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (guild_id) DO UPDATE SET
				voice_channel_id = EXCLUDED.voice_channel_id,
				text_channel_id = EXCLUDED.text_channel_id,
				station = EXCLUDED.station,
				enabled_by = EXCLUDED.enabled_by,
				enabled_at = EXCLUDED.enabled_at`,
			int64(setting.GuildID), int64(setting.VoiceChannelID), textID, setting.Station,
			int64(setting.EnabledBy), setting.EnabledAt); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.settings[setting.GuildID] = setting
	delete(s.health, setting.GuildID)
	s.mu.Unlock()
	return nil
}

func (s *StayStore) Delete(ctx context.Context, guildID snowflake.ID) (bool, error) {
	if s.db != nil {
		if _, err := s.db.Pool.Exec(ctx, "DELETE FROM radio_stays WHERE guild_id = $1", int64(guildID)); err != nil {
			return false, err
		}
	}
	s.mu.Lock()
	_, existed := s.settings[guildID]
	delete(s.settings, guildID)
	delete(s.health, guildID)
	s.mu.Unlock()
	return existed, nil
}

func (s *StayStore) due(guildID snowflake.ID, now time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !now.Before(s.health[guildID].RetryAt)
}

func (s *StayStore) recordFailure(guildID snowflake.ID, err error, now time.Time) StayHealth {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.settings[guildID]; !ok {
		return StayHealth{}
	}
	h := s.health[guildID]
	h.Failures++
	h.LastError = err.Error()
	backoff := stayBaseBackoff << min(h.Failures-1, 10)
	h.RetryAt = now.Add(min(backoff, stayMaxBackoff))
	s.health[guildID] = h
	return h
}

func (s *StayStore) recordSuccess(guildID snowflake.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.health, guildID)
}

func (b *Bot) CanManageStay(guildID, userID snowflake.ID, member *discord.ResolvedMember) bool {
	if slices.Contains(b.Cfg.API.AdminUserIDs, userID) {
		return true
	}
	return member != nil && member.Permissions.Has(discord.PermissionManageGuild)
}

func (b *Bot) EnableStay(ctx context.Context, setting StaySetting) (*Player, error) {
	if b.Radio == nil {
		return nil, ErrRadioDisabled
	}
	np, ok := b.Radio.Station(setting.Station)
	if !ok {
		return nil, fmt.Errorf("unknown station %q", setting.Station)
	}
	if !np.IsOnline {
		return nil, fmt.Errorf("%s is offline right now", np.Station.Name)
	}
	setting.Station = np.Station.Shortcode

	track, err := b.LoadRadioTrack(ctx, np, TrackMeta{Requester: setting.EnabledBy})
	if err != nil {
		return nil, err
	}
	player, err := b.Enqueue(ctx, EnqueueRequest{
		GuildID:        setting.GuildID,
		VoiceChannelID: setting.VoiceChannelID,
		TextChannelID:  setting.TextChannelID,
		Tracks:         []lavalink.Track{track},
		PlayNow:        true,
	})
	if err != nil {
		return player, err
	}
	if err := b.Stays.Set(ctx, setting); err != nil {
		return player, fmt.Errorf("failed to save 24/7 setting: %w", err)
	}
	slog.Info("24/7 radio enabled",
		slog.String("guild_id", setting.GuildID.String()),
		slog.String("station", setting.Station),
		slog.String("voice_channel_id", setting.VoiceChannelID.String()))
	return player, nil
}

func (b *Bot) DisableStay(ctx context.Context, guildID snowflake.ID) (bool, error) {
	existed, err := b.Stays.Delete(ctx, guildID)
	if err == nil && existed {
		slog.Info("24/7 radio disabled", slog.String("guild_id", guildID.String()))
	}
	return existed, err
}

func (b *Bot) RunStaySupervisor(ctx context.Context) {
	timer := time.NewTimer(stayFirstCheck)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			b.checkStays(ctx)
			timer.Reset(stayCheckInterval)
		}
	}
}

func (b *Bot) checkStays(ctx context.Context) {
	now := time.Now()
	for _, setting := range b.Stays.All() {
		if !b.Stays.due(setting.GuildID, now) {
			continue
		}
		started, err := b.ensureStay(ctx, setting)
		if err != nil {
			h := b.Stays.recordFailure(setting.GuildID, err, now)
			slog.Warn("failed to keep 24/7 radio running",
				slog.String("guild_id", setting.GuildID.String()),
				slog.String("station", setting.Station),
				slog.Int("failures", h.Failures),
				slog.Time("retry_at", h.RetryAt),
				slog.Any("error", err))
			continue
		}
		b.Stays.recordSuccess(setting.GuildID)
		if started {
			slog.Info("restarted 24/7 radio",
				slog.String("guild_id", setting.GuildID.String()), slog.String("station", setting.Station))
		}
	}
}

func (b *Bot) ensureStay(ctx context.Context, setting StaySetting) (bool, error) {
	if b.Radio == nil {
		return false, ErrRadioDisabled
	}
	if _, ok := b.Client.Caches().Guild(setting.GuildID); !ok {
		return false, nil
	}
	_, inVoice := b.BotVoiceChannel(setting.GuildID)
	if player, ok := b.PlayerManager.GetPlayer(setting.GuildID); ok && inVoice && player.IsPlaying() {
		return false, nil
	}

	np, ok := b.Radio.Station(setting.Station)
	if !ok {
		return false, fmt.Errorf("station %q is not available", setting.Station)
	}
	if !np.IsOnline {
		return false, fmt.Errorf("station %q is offline", setting.Station)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	track, err := b.LoadRadioTrack(ctx, np, TrackMeta{Requester: setting.EnabledBy})
	if err != nil {
		return false, err
	}
	if _, err := b.Enqueue(ctx, EnqueueRequest{
		GuildID:        setting.GuildID,
		VoiceChannelID: setting.VoiceChannelID,
		TextChannelID:  setting.TextChannelID,
		Tracks:         []lavalink.Track{track},
		PlayNow:        true,
	}); err != nil {
		return false, err
	}
	return true, nil
}
