package musicbot

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrPlaylistNotFound = errors.New("playlist not found")
	ErrPlaylistExists   = errors.New("a playlist with that name already exists")
)

type Playlist struct {
	ID         int          `db:"id" json:"id"`
	Name       string       `db:"name" json:"name"`
	OwnerID    snowflake.ID `db:"owner_id" json:"owner_id"`
	CreatedAt  time.Time    `db:"created_at" json:"created_at"`
	TrackCount int          `db:"-" json:"track_count"`
}

type PlaylistTrack struct {
	ID         int            `db:"id"`
	PlaylistID int            `db:"playlist_id"`
	TrackTitle string         `db:"track_title"`
	Track      lavalink.Track `db:"track"`
	AddedAt    time.Time      `db:"added_at"`
	AddedBy    snowflake.ID   `db:"added_by"`
	LibraryKey string         `db:"library_key"`
}

const playlistColumns = "p.id, p.name, p.owner_id, p.created_at, (SELECT COUNT(*) FROM playlist_tracks t WHERE t.playlist_id = p.id)"

func scanPlaylist(row pgx.Row) (Playlist, error) {
	var (
		playlist Playlist
		ownerID  int64
		created  *time.Time
	)
	if err := row.Scan(&playlist.ID, &playlist.Name, &ownerID, &created, &playlist.TrackCount); err != nil {
		return Playlist{}, err
	}
	playlist.OwnerID = snowflake.ID(ownerID)
	if created != nil {
		playlist.CreatedAt = *created
	}
	return playlist, nil
}

func (d *DB) scanPlaylists(rows pgx.Rows) ([]Playlist, error) {
	defer rows.Close()

	playlists := make([]Playlist, 0)
	for rows.Next() {
		playlist, err := scanPlaylist(rows)
		if err != nil {
			slog.Error("failed to parse playlist", slog.Any("err", err))
			continue
		}
		playlists = append(playlists, playlist)
	}
	return playlists, rows.Err()
}

func (d *DB) scanTracks(rows pgx.Rows) ([]PlaylistTrack, error) {
	defer rows.Close()

	playlistTracks := make([]PlaylistTrack, 0)
	for rows.Next() {
		var (
			track    PlaylistTrack
			rawTrack json.RawMessage
			addedBy  int64
			addedAt  *time.Time
			key      *string
		)
		if err := rows.Scan(&track.ID, &track.PlaylistID, &track.TrackTitle, &rawTrack, &addedAt, &addedBy, &key); err != nil {
			slog.Error("failed to parse playlist track from database", slog.Any("err", err))
			continue
		}
		if err := json.Unmarshal(rawTrack, &track.Track); err != nil {
			slog.Error("failed to decode track object", slog.Any("err", err))
			continue
		}
		track.AddedBy = snowflake.ID(addedBy)
		if key != nil {
			track.LibraryKey = *key
		}
		if addedAt != nil {
			track.AddedAt = *addedAt
		}
		playlistTracks = append(playlistTracks, track)
	}
	return playlistTracks, rows.Err()
}

func (d *DB) CreatePlaylist(ctx context.Context, userID snowflake.ID, playlistName string) (Playlist, error) {
	row := d.Pool.QueryRow(ctx,
		"INSERT INTO playlists (owner_id, name) VALUES ($1, $2) RETURNING id, name, owner_id, created_at, 0",
		int64(userID), playlistName)
	playlist, err := scanPlaylist(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Playlist{}, ErrPlaylistExists
		}
		return Playlist{}, err
	}
	return playlist, nil
}

func (d *DB) RenamePlaylist(ctx context.Context, userID snowflake.ID, playlistID int, name string) error {
	tag, err := d.Pool.Exec(ctx, "UPDATE playlists SET name = $1 WHERE id = $2 AND owner_id = $3", name, playlistID, int64(userID))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrPlaylistExists
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPlaylistNotFound
	}
	return nil
}

func (d *DB) DeletePlaylist(ctx context.Context, userID snowflake.ID, playlistID int) error {
	tag, err := d.Pool.Exec(ctx, "DELETE FROM playlists WHERE id = $1 AND owner_id = $2", playlistID, int64(userID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPlaylistNotFound
	}
	return nil
}

func (d *DB) SearchPlaylist(ctx context.Context, userID snowflake.ID, query string, limit int) ([]Playlist, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if query == "" {
		rows, err = d.Pool.Query(ctx,
			"SELECT "+playlistColumns+" FROM playlists p WHERE p.owner_id = $1 ORDER BY p.name LIMIT $2",
			int64(userID), limit)
	} else {
		rows, err = d.Pool.Query(ctx,
			"SELECT "+playlistColumns+" FROM playlists p WHERE p.owner_id = $1 AND p.name ILIKE '%' || $2 || '%' ORDER BY p.name LIMIT $3",
			int64(userID), escapeLike(query), limit)
	}
	if err != nil {
		return nil, err
	}
	return d.scanPlaylists(rows)
}

func (d *DB) GetPlaylist(ctx context.Context, userID snowflake.ID, playlistID int) (Playlist, []PlaylistTrack, error) {
	row := d.Pool.QueryRow(ctx,
		"SELECT "+playlistColumns+" FROM playlists p WHERE p.id = $1 AND p.owner_id = $2",
		playlistID, int64(userID))

	playlist, err := scanPlaylist(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Playlist{}, nil, ErrPlaylistNotFound
		}
		return Playlist{}, nil, err
	}

	rows, err := d.Pool.Query(ctx,
		"SELECT id, playlist_id, track_title, track, added_at, added_by, library_key FROM playlist_tracks WHERE playlist_id = $1 ORDER BY id",
		playlistID)
	if err != nil {
		return playlist, nil, err
	}

	playlistTracks, err := d.scanTracks(rows)
	if err != nil {
		return Playlist{}, nil, err
	}
	return playlist, playlistTracks, nil
}

func (d *DB) AddTracksToPlaylist(ctx context.Context, playlistId int, userId snowflake.ID, tracks []lavalink.Track) error {
	if len(tracks) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, track := range tracks {
		track.UserData = nil
		var key *string
		if track.Info.SourceName == LibrarySource {
			id := track.Info.Identifier
			key = &id
		}
		batch.Queue(
			"INSERT INTO playlist_tracks (playlist_id, track_title, added_by, track, library_key) VALUES ($1, $2, $3, $4, $5)",
			playlistId, Trim(TrackTitle(track), 255), int64(userId), track, key)
	}
	return d.Pool.SendBatch(ctx, batch).Close()
}

func (d *DB) RemoveTrackFromPlaylist(ctx context.Context, userID snowflake.ID, playlistID int, trackID int) error {
	tag, err := d.Pool.Exec(ctx, `
		DELETE FROM playlist_tracks t
		USING playlists p
		WHERE t.id = $1 AND t.playlist_id = $2 AND p.id = t.playlist_id AND p.owner_id = $3`,
		trackID, playlistID, int64(userID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPlaylistNotFound
	}
	return nil
}

func (d *DB) LinkLegacyPlaylistTracks(ctx context.Context, lib *Library) (int, error) {
	rows, err := d.Pool.Query(ctx, "SELECT id, track FROM playlist_tracks WHERE library_key IS NULL")
	if err != nil {
		return 0, err
	}
	type link struct {
		id  int
		key string
	}
	var links []link
	for rows.Next() {
		var (
			id  int
			raw json.RawMessage
		)
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return 0, err
		}
		var track lavalink.Track
		if err := json.Unmarshal(raw, &track); err != nil {
			continue
		}
		if match, ok := lib.Match(QueryFromTrack(track)); ok {
			links = append(links, link{id: id, key: match.Key()})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(links) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, l := range links {
		batch.Queue("UPDATE playlist_tracks SET library_key = $1 WHERE id = $2 AND library_key IS NULL", l.key, l.id)
	}
	if err := d.Pool.SendBatch(ctx, batch).Close(); err != nil {
		return 0, err
	}
	return len(links), nil
}

func escapeLike(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '%' || r == '_' || r == '\\' {
			out = append(out, '\\')
		}
		out = append(out, r)
	}
	return string(out)
}

func (d *DB) Ping(ctx context.Context) error {
	return d.Pool.Ping(ctx)
}
