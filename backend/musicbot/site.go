package musicbot

import "strings"

const (
	ColorMint  = 0xB5EBE5
	ColorPink  = 0xFF9AA2
	ColorPeach = 0xFFDAC1
)

func (b *Bot) SiteURL(path string) string {
	base := strings.TrimRight(b.Cfg.Site.URL, "/")
	if base == "" {
		base = "https://chillybot.space"
	}
	if path == "" || path == "/" {
		return base
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func (b *Bot) SiteHost() string {
	host := strings.TrimPrefix(strings.TrimPrefix(b.SiteURL(""), "https://"), "http://")
	return strings.TrimSuffix(host, "/")
}

var sourceLabels = map[string]string{
	LibrarySource: "Chilly Library",
	"youtube":     "YouTube",
	"soundcloud":  "SoundCloud",
	"spotify":     "Spotify",
	"deezer":      "Deezer",
	"applemusic":  "Apple Music",
	"bandcamp":    "Bandcamp",
	"twitch":      "Twitch",
	"http":        "Stream",
}

func SourceLabel(name string) string {
	if label, ok := sourceLabels[name]; ok {
		return label
	}
	if name == "" {
		return "Unknown source"
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
