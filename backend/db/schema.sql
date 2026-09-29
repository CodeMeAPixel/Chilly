CREATE TABLE IF NOT EXISTS playlists
(
    id          BIGSERIAL       PRIMARY KEY,
    name        VARCHAR(255)    NOT NULL,
    owner_id    BIGINT          NOT NULL,
    created_at  TIMESTAMP       DEFAULT CURRENT_TIMESTAMP,
    UNIQUE      (name, owner_id)
);

CREATE TABLE IF NOT EXISTS playlist_tracks
(
    id          BIGSERIAL       PRIMARY KEY,
    playlist_id BIGINT          NOT NULL REFERENCES playlists (id) ON DELETE CASCADE,
    track_title VARCHAR(255)    NOT NULL,
    track       JSONB           NOT NULL,
    added_at    TIMESTAMP       DEFAULT CURRENT_TIMESTAMP,
    added_by    BIGINT          NOT NULL
);

CREATE INDEX IF NOT EXISTS playlist_tracks_playlist_id_idx ON playlist_tracks (playlist_id);

/* dashboard / website sessions (the token itself is never stored, only its sha256) */
CREATE TABLE IF NOT EXISTS api_sessions
(
    token_hash   BYTEA           PRIMARY KEY,
    user_id      BIGINT          NOT NULL,
    username     VARCHAR(100)    NOT NULL,
    global_name  VARCHAR(100),
    avatar       VARCHAR(100),
    guilds       JSONB           NOT NULL DEFAULT '[]',
    created_at   TIMESTAMPTZ     NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ     NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ     NOT NULL
);

CREATE INDEX IF NOT EXISTS api_sessions_user_id_idx ON api_sessions (user_id);
CREATE INDEX IF NOT EXISTS api_sessions_expires_at_idx ON api_sessions (expires_at);

CREATE TABLE IF NOT EXISTS radio_stays
(
    guild_id         BIGINT          PRIMARY KEY,
    voice_channel_id BIGINT          NOT NULL,
    text_channel_id  BIGINT,
    station          VARCHAR(100)    NOT NULL,
    enabled_by       BIGINT          NOT NULL,
    enabled_at       TIMESTAMPTZ     NOT NULL DEFAULT now()
);

ALTER TABLE playlist_tracks ADD COLUMN IF NOT EXISTS library_key VARCHAR(100);
CREATE INDEX IF NOT EXISTS playlist_tracks_unlinked_idx ON playlist_tracks (id) WHERE library_key IS NULL;

CREATE TABLE IF NOT EXISTS lyrics_backfill
(
    song_key   VARCHAR(100)    PRIMARY KEY,
    checked_at TIMESTAMPTZ     NOT NULL DEFAULT now(),
    written    BOOLEAN         NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS song_requests
(
    id         BIGSERIAL       PRIMARY KEY,
    user_id    BIGINT          NOT NULL,
    guild_id   BIGINT,
    station    VARCHAR(100)    NOT NULL,
    media_key  VARCHAR(100)    NOT NULL,
    song_id    VARCHAR(100)    NOT NULL DEFAULT '',
    title      VARCHAR(255)    NOT NULL,
    artist     VARCHAR(255)    NOT NULL DEFAULT '',
    art        TEXT            NOT NULL DEFAULT '',
    source     VARCHAR(20)     NOT NULL,
    status     VARCHAR(20)     NOT NULL DEFAULT 'queued',
    created_at TIMESTAMPTZ     NOT NULL DEFAULT now(),
    played_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS song_requests_user_idx ON song_requests (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS song_requests_open_idx ON song_requests (station, status) WHERE status IN ('queued', 'playing');

CREATE TABLE IF NOT EXISTS song_suggestions
(
    id          BIGSERIAL       PRIMARY KEY,
    user_id     BIGINT          NOT NULL,
    artist      VARCHAR(255)    NOT NULL,
    title       VARCHAR(255)    NOT NULL,
    link        TEXT            NOT NULL DEFAULT '',
    note        TEXT            NOT NULL DEFAULT '',
    status      VARCHAR(20)     NOT NULL DEFAULT 'pending',
    reason      TEXT            NOT NULL DEFAULT '',
    library_key VARCHAR(100)    NOT NULL DEFAULT '',
    reviewed_by BIGINT,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS song_suggestions_user_idx ON song_suggestions (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS song_suggestions_open_idx ON song_suggestions (status) WHERE status IN ('pending', 'reviewing');
