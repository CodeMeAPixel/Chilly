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
