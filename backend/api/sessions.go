package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
)

var errSessionNotFound = errors.New("session not found")

const (
	permAdministrator = 1 << 3
	permManageGuild   = 1 << 5
)

type SessionGuild struct {
	ID          snowflake.ID `json:"id"`
	Name        string       `json:"name"`
	Icon        *string      `json:"icon"`
	Owner       bool         `json:"owner"`
	Permissions string       `json:"permissions"`
}

func (g SessionGuild) CanManage() bool {
	if g.Owner {
		return true
	}
	perms, _ := strconv.ParseUint(g.Permissions, 10, 64)
	return perms&permAdministrator != 0 || perms&permManageGuild != 0
}

type Session struct {
	UserID     snowflake.ID
	Username   string
	GlobalName *string
	Avatar     *string
	Guilds     []SessionGuild
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

func (s *Session) Guild(id snowflake.ID) (SessionGuild, bool) {
	for _, g := range s.Guilds {
		if g.ID == id {
			return g, true
		}
	}
	return SessionGuild{}, false
}

type SessionStore struct {
	db *musicbot.DB
}

func NewSessionStore(db *musicbot.DB) *SessionStore {
	return &SessionStore{db: db}
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (st *SessionStore) Create(ctx context.Context, s Session, ttl time.Duration) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	guilds, err := json.Marshal(s.Guilds)
	if err != nil {
		return "", err
	}
	_, err = st.db.Pool.Exec(ctx, `
		INSERT INTO api_sessions (token_hash, user_id, username, global_name, avatar, guilds, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		hashToken(token), int64(s.UserID), s.Username, s.GlobalName, s.Avatar, guilds, time.Now().Add(ttl))
	return token, err
}

func (st *SessionStore) Get(ctx context.Context, token string) (*Session, error) {
	var (
		s      Session
		userID int64
		guilds []byte
	)
	err := st.db.Pool.QueryRow(ctx, `
		UPDATE api_sessions SET last_seen_at = now()
		WHERE token_hash = $1 AND expires_at > now()
		RETURNING user_id, username, global_name, avatar, guilds, created_at, expires_at`,
		hashToken(token)).Scan(&userID, &s.Username, &s.GlobalName, &s.Avatar, &guilds, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errSessionNotFound
		}
		return nil, err
	}
	s.UserID = snowflake.ID(userID)
	if err := json.Unmarshal(guilds, &s.Guilds); err != nil {
		return nil, err
	}
	return &s, nil
}

func (st *SessionStore) Delete(ctx context.Context, token string) error {
	_, err := st.db.Pool.Exec(ctx, "DELETE FROM api_sessions WHERE token_hash = $1", hashToken(token))
	return err
}

func (st *SessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := st.db.Pool.Exec(ctx, "DELETE FROM api_sessions WHERE expires_at <= now()")
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
