package musicbot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
)

type MediaSigner struct {
	key     []byte
	baseURL string
}

func NewMediaSigner(secret, baseURL string) *MediaSigner {
	sum := sha256.Sum256([]byte("chilly-media:" + secret))
	return &MediaSigner{key: sum[:], baseURL: strings.TrimRight(baseURL, "/")}
}

func (m *MediaSigner) BaseURL() string {
	return m.baseURL
}

func (m *MediaSigner) signature(station string, id int) string {
	mac := hmac.New(sha256.New, m.key)
	mac.Write([]byte(station + "/" + strconv.Itoa(id)))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

func (m *MediaSigner) URL(station string, id int) string {
	return m.baseURL + "/api/v1/media/" + url.PathEscape(station) + "/" + strconv.Itoa(id) + "?sig=" + m.signature(station, id)
}

func (m *MediaSigner) Verify(station string, id int, sig string) bool {
	return hmac.Equal([]byte(sig), []byte(m.signature(station, id)))
}
