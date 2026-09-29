package musicbot

import (
	"context"
	"errors"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
)

const requestColumns = "id, user_id, COALESCE(guild_id, 0), station, media_key, song_id, title, artist, art, source, status, created_at, played_at"

func scanRequests(rows pgx.Rows) ([]SongRequest, error) {
	defer rows.Close()
	out := make([]SongRequest, 0)
	for rows.Next() {
		var (
			r               SongRequest
			userID, guildID int64
		)
		if err := rows.Scan(&r.ID, &userID, &guildID, &r.Station, &r.MediaKey, &r.SongID, &r.Title, &r.Artist, &r.Art,
			&r.Source, &r.Status, &r.CreatedAt, &r.PlayedAt); err != nil {
			return nil, err
		}
		r.UserID, r.GuildID = snowflake.ID(userID), snowflake.ID(guildID)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d *DB) LastRequestAt(ctx context.Context, userID snowflake.ID) (time.Time, error) {
	var last *time.Time
	if err := d.Pool.QueryRow(ctx, "SELECT max(created_at) FROM song_requests WHERE user_id = $1", int64(userID)).Scan(&last); err != nil {
		return time.Time{}, err
	}
	if last == nil {
		return time.Time{}, nil
	}
	return *last, nil
}

func (d *DB) CountOpenRequests(ctx context.Context, userID snowflake.ID) (int, error) {
	var n int
	err := d.Pool.QueryRow(ctx, "SELECT count(*) FROM song_requests WHERE user_id = $1 AND status = $2", int64(userID), RequestQueued).Scan(&n)
	return n, err
}

func (d *DB) InsertRequest(ctx context.Context, r *SongRequest) error {
	var guildID *int64
	if r.GuildID != 0 {
		id := int64(r.GuildID)
		guildID = &id
	}
	return d.Pool.QueryRow(ctx, `
		INSERT INTO song_requests (user_id, guild_id, station, media_key, song_id, title, artist, art, source, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`,
		int64(r.UserID), guildID, r.Station, r.MediaKey, r.SongID, r.Title, r.Artist, r.Art, r.Source, r.Status,
	).Scan(&r.ID, &r.CreatedAt)
}

func (d *DB) AdvanceRequests(ctx context.Context, station, songID string) error {
	if _, err := d.Pool.Exec(ctx,
		"UPDATE song_requests SET status = $1 WHERE station = $2 AND status = $3 AND song_id <> $4",
		RequestPlayed, station, RequestPlaying, songID); err != nil {
		return err
	}
	if songID == "" {
		return nil
	}
	_, err := d.Pool.Exec(ctx, `
		UPDATE song_requests SET status = $1, played_at = now()
		WHERE id = (
			SELECT id FROM song_requests
			WHERE station = $2 AND status = $3 AND song_id = $4
			ORDER BY created_at LIMIT 1
		)`, RequestPlaying, station, RequestQueued, songID)
	return err
}

func (d *DB) ExpireRequests(ctx context.Context, before time.Time) error {
	_, err := d.Pool.Exec(ctx, "UPDATE song_requests SET status = $1 WHERE status = $2 AND created_at < $3",
		RequestExpired, RequestQueued, before)
	return err
}

func (d *DB) UserRequests(ctx context.Context, userID snowflake.ID, limit int) ([]SongRequest, error) {
	rows, err := d.Pool.Query(ctx, "SELECT "+requestColumns+" FROM song_requests WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2",
		int64(userID), limit)
	if err != nil {
		return nil, err
	}
	return scanRequests(rows)
}

func (d *DB) RecentRequests(ctx context.Context, limit int) ([]SongRequest, error) {
	rows, err := d.Pool.Query(ctx, "SELECT "+requestColumns+" FROM song_requests ORDER BY created_at DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	return scanRequests(rows)
}

const suggestionColumns = "id, user_id, artist, title, link, note, status, reason, library_key, COALESCE(reviewed_by, 0), created_at, updated_at"

func scanSuggestion(row pgx.Row) (SongSuggestion, error) {
	var (
		s                  SongSuggestion
		userID, reviewedBy int64
	)
	if err := row.Scan(&s.ID, &userID, &s.Artist, &s.Title, &s.Link, &s.Note, &s.Status, &s.Reason, &s.LibraryKey,
		&reviewedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return SongSuggestion{}, err
	}
	s.UserID, s.ReviewedBy = snowflake.ID(userID), snowflake.ID(reviewedBy)
	return s, nil
}

func scanSuggestions(rows pgx.Rows) ([]SongSuggestion, error) {
	defer rows.Close()
	out := make([]SongSuggestion, 0)
	for rows.Next() {
		s, err := scanSuggestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (d *DB) CountOpenSuggestions(ctx context.Context, userID snowflake.ID) (int, error) {
	var n int
	err := d.Pool.QueryRow(ctx, "SELECT count(*) FROM song_suggestions WHERE user_id = $1 AND status IN ($2, $3)",
		int64(userID), SuggestionPending, SuggestionReviewing).Scan(&n)
	return n, err
}

func (d *DB) InsertSuggestion(ctx context.Context, s *SongSuggestion) error {
	return d.Pool.QueryRow(ctx, `
		INSERT INTO song_suggestions (user_id, artist, title, link, note, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		int64(s.UserID), s.Artist, s.Title, s.Link, s.Note, s.Status,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (d *DB) UserSuggestions(ctx context.Context, userID snowflake.ID, limit int) ([]SongSuggestion, error) {
	rows, err := d.Pool.Query(ctx, "SELECT "+suggestionColumns+" FROM song_suggestions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2",
		int64(userID), limit)
	if err != nil {
		return nil, err
	}
	return scanSuggestions(rows)
}

func (d *DB) ListSuggestions(ctx context.Context, status string, limit int) ([]SongSuggestion, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if status == "" {
		rows, err = d.Pool.Query(ctx, "SELECT "+suggestionColumns+" FROM song_suggestions ORDER BY created_at DESC LIMIT $1", limit)
	} else {
		rows, err = d.Pool.Query(ctx, "SELECT "+suggestionColumns+" FROM song_suggestions WHERE status = $1 ORDER BY created_at LIMIT $2", status, limit)
	}
	if err != nil {
		return nil, err
	}
	return scanSuggestions(rows)
}

func (d *DB) GetSuggestion(ctx context.Context, id int64) (SongSuggestion, error) {
	s, err := scanSuggestion(d.Pool.QueryRow(ctx, "SELECT "+suggestionColumns+" FROM song_suggestions WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return SongSuggestion{}, ErrSuggestionNotFound
	}
	return s, err
}

func (d *DB) UpdateSuggestion(ctx context.Context, s SongSuggestion) error {
	var reviewedBy *int64
	if s.ReviewedBy != 0 {
		id := int64(s.ReviewedBy)
		reviewedBy = &id
	}
	tag, err := d.Pool.Exec(ctx, `
		UPDATE song_suggestions
		SET status = $1, reason = $2, library_key = $3, reviewed_by = COALESCE($4, reviewed_by), updated_at = now()
		WHERE id = $5`,
		s.Status, s.Reason, s.LibraryKey, reviewedBy, s.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSuggestionNotFound
	}
	return nil
}

func (d *DB) DeleteOpenSuggestion(ctx context.Context, id int64, userID snowflake.ID) (bool, error) {
	tag, err := d.Pool.Exec(ctx, "DELETE FROM song_suggestions WHERE id = $1 AND user_id = $2 AND status = $3",
		id, int64(userID), SuggestionPending)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

var _ RequestStore = (*DB)(nil)
