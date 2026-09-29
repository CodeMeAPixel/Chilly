package musicbot

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"gopkg.in/yaml.v3"
)

func ReadConfig(path string) (Config, error) {
	cfg := defaultConfig()

	if path == "" {
		path = ".env"
	}

	if _, err := os.Stat(path); err == nil {
		if strings.HasSuffix(strings.ToLower(path), ".env") {
			if err := loadEnvFile(path); err != nil {
				return Config{}, fmt.Errorf("failed to load .env file: %w", err)
			}
			if err := loadFromEnvironment(&cfg); err != nil {
				return Config{}, fmt.Errorf("failed to parse .env config: %w", err)
			}
			return cfg, nil
		}

		file, err := os.Open(path)
		if err != nil {
			return Config{}, fmt.Errorf("failed to open config: %w", err)
		}
		defer file.Close()

		if err = yaml.NewDecoder(file).Decode(&cfg); err != nil {
			return Config{}, fmt.Errorf("failed to decode config: %w", err)
		}
		return cfg, nil
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("failed to access config path %q: %w", path, err)
	}

	if err := loadFromEnvironment(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to read environment config: %w", err)
	}
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		MusicTracker: TrackerConfig{
			Enabled: false,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
			Env:    "prod",
		},
		Site: SiteConfig{
			URL: "https://chillybot.space",
		},
		Lyrics: LyricsConfig{
			URL: "https://lrclib.net",
		},
		Search: SearchConfig{
			Providers: []string{"scsearch", "spsearch", "dzsearch"},
		},
		API: APIConfig{
			Address:      ":8080",
			CookieSecure: true,
			SessionTTL:   7 * 24 * time.Hour,
		},
		AzuraCast: AzuraCastConfig{
			PollInterval: 15 * time.Second,
		},
	}
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func loadFromEnvironment(cfg *Config) error {
	cfg.Bot.Token = getenv("BOT_TOKEN", cfg.Bot.Token)
	cfg.Bot.Status = getenv("BOT_STATUS", cfg.Bot.Status)
	cfg.Bot.ActivityName = getenv("BOT_ACTIVITY_NAME", cfg.Bot.ActivityName)
	cfg.Bot.ActivityType = getenv("BOT_ACTIVITY_TYPE", cfg.Bot.ActivityType)

	cfg.Nodes = parseNodesFromEnvironment(cfg.Nodes)
	cfg.MusicTracker.Enabled = parseBoolEnv("MUSIC_TRACKER_ENABLED", cfg.MusicTracker.Enabled)
	cfg.MusicTracker.ChannelID = parseSnowflakeEnv("MUSIC_TRACKER_CHANNEL_ID", cfg.MusicTracker.ChannelID)
	cfg.MusicTracker.GuildID = parseSnowflakeEnv("MUSIC_TRACKER_GUILD_ID", cfg.MusicTracker.GuildID)
	cfg.MusicTracker.HostAddress = getenv("MUSIC_TRACKER_HOST_ADDRESS", cfg.MusicTracker.HostAddress)
	cfg.MusicTracker.HttpPath = getenv("MUSIC_TRACKER_HTTP_PATH", cfg.MusicTracker.HttpPath)
	cfg.MusicTracker.WebsocketPath = getenv("MUSIC_TRACKER_WEBSOCKET_PATH", cfg.MusicTracker.WebsocketPath)
	cfg.MusicTracker.AllowedOrigins = parseStringSliceEnv("MUSIC_TRACKER_ALLOWED_ORIGINS", cfg.MusicTracker.AllowedOrigins)

	cfg.DB.Host = getenv("DB_HOST", cfg.DB.Host)
	cfg.DB.Port = parseIntEnv("DB_PORT", cfg.DB.Port)
	cfg.DB.Username = getenv("DB_USERNAME", cfg.DB.Username)
	cfg.DB.Password = getenv("DB_PASSWORD", cfg.DB.Password)
	cfg.DB.Database = getenv("DB_NAME", cfg.DB.Database)

	cfg.Log.Level = getenv("LOG_LEVEL", cfg.Log.Level)
	cfg.Log.Format = getenv("LOG_FORMAT", cfg.Log.Format)
	cfg.Log.FilePath = getenv("LOG_FILE_PATH", cfg.Log.FilePath)
	cfg.Log.AddSource = parseBoolEnv("LOG_ADD_SOURCE", cfg.Log.AddSource)
	cfg.Log.Env = getenv("LOG_ENV", cfg.Log.Env)

	cfg.Site.URL = strings.TrimRight(getenv("SITE_URL", cfg.Site.URL), "/")
	cfg.Lyrics.URL = strings.TrimRight(getenv("LYRICS_API_URL", cfg.Lyrics.URL), "/")
	cfg.Search.Providers = parseStringSliceEnv("SEARCH_PROVIDERS", cfg.Search.Providers)

	cfg.API.Enabled = parseBoolEnv("API_ENABLED", cfg.API.Enabled)
	cfg.API.Address = getenv("API_ADDRESS", cfg.API.Address)
	cfg.API.PublicURL = strings.TrimRight(getenv("API_PUBLIC_URL", cfg.API.PublicURL), "/")
	cfg.API.DashboardURL = strings.TrimRight(getenv("API_DASHBOARD_URL", cfg.API.DashboardURL), "/")
	cfg.API.AllowedOrigins = parseStringSliceEnv("API_ALLOWED_ORIGINS", cfg.API.AllowedOrigins)
	cfg.API.ClientID = getenv("DISCORD_CLIENT_ID", cfg.API.ClientID)
	cfg.API.ClientSecret = getenv("DISCORD_CLIENT_SECRET", cfg.API.ClientSecret)
	cfg.API.CookieDomain = getenv("API_COOKIE_DOMAIN", cfg.API.CookieDomain)
	cfg.API.CookieSecure = parseBoolEnv("API_COOKIE_SECURE", cfg.API.CookieSecure)
	cfg.API.SessionTTL = parseDurationEnv("API_SESSION_TTL", cfg.API.SessionTTL)
	cfg.API.TrustProxy = parseBoolEnv("API_TRUST_PROXY", cfg.API.TrustProxy)
	cfg.API.AdminUserIDs = parseSnowflakeSliceEnv("API_ADMIN_USER_IDS", cfg.API.AdminUserIDs)

	cfg.AzuraCast.Enabled = parseBoolEnv("AZURACAST_ENABLED", cfg.AzuraCast.Enabled)
	cfg.AzuraCast.URL = strings.TrimRight(getenv("AZURACAST_URL", cfg.AzuraCast.URL), "/")
	cfg.AzuraCast.APIKey = getenv("AZURACAST_API_KEY", cfg.AzuraCast.APIKey)
	cfg.AzuraCast.Stations = parseStringSliceEnv("AZURACAST_STATIONS", cfg.AzuraCast.Stations)
	cfg.AzuraCast.PollInterval = parseDurationEnv("AZURACAST_POLL_INTERVAL", cfg.AzuraCast.PollInterval)
	cfg.AzuraCast.StreamBaseURL = strings.TrimRight(getenv("AZURACAST_STREAM_BASE_URL", cfg.AzuraCast.StreamBaseURL), "/")

	return nil
}

func parseNodesFromEnvironment(existing []NodeConfig) []NodeConfig {
	var nodes []NodeConfig

	addNode := func(node NodeConfig) {
		if node.Name == "" && node.Address == "" && node.Password == "" && !node.Secure && node.SessionID == "" {
			return
		}
		nodes = append(nodes, node)
	}

	if v := getenv("NODE_NAME", ""); v != "" || getenv("NODE_ADDRESS", "") != "" || getenv("NODE_PASSWORD", "") != "" || getenv("NODE_SESSION_ID", "") != "" {
		addNode(NodeConfig{
			Name:      getenv("NODE_NAME", "node-1"),
			Address:   getenv("NODE_ADDRESS", ""),
			Password:  getenv("NODE_PASSWORD", ""),
			Secure:    parseBoolEnv("NODE_SECURE", false),
			SessionID: getenv("NODE_SESSION_ID", ""),
			Location:  strings.TrimSpace(getenv("NODE_LOCATION", "")),
		})
	}

	for i := 1; i <= 10; i++ {
		nameKey := fmt.Sprintf("NODE_%d_NAME", i)
		addressKey := fmt.Sprintf("NODE_%d_ADDRESS", i)
		passwordKey := fmt.Sprintf("NODE_%d_PASSWORD", i)
		secureKey := fmt.Sprintf("NODE_%d_SECURE", i)
		sessionKey := fmt.Sprintf("NODE_%d_SESSION_ID", i)
		locationKey := fmt.Sprintf("NODE_%d_LOCATION", i)

		if getenv(nameKey, "") == "" && getenv(addressKey, "") == "" && getenv(passwordKey, "") == "" && getenv(sessionKey, "") == "" {
			continue
		}

		addNode(NodeConfig{
			Name:      getenv(nameKey, fmt.Sprintf("node-%d", i)),
			Address:   getenv(addressKey, ""),
			Password:  getenv(passwordKey, ""),
			Secure:    parseBoolEnv(secureKey, false),
			SessionID: getenv(sessionKey, ""),
			Location:  strings.TrimSpace(getenv(locationKey, "")),
		})
	}

	if len(nodes) > 0 {
		return nodes
	}
	return existing
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func parseBoolEnv(key string, fallback bool) bool {
	value := getenv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseIntEnv(key string, fallback int) int {
	value := getenv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseSnowflakeEnv(key string, fallback snowflake.ID) snowflake.ID {
	value := getenv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := snowflake.Parse(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	value := getenv(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseSnowflakeSliceEnv(key string, fallback []snowflake.ID) []snowflake.ID {
	values := parseStringSliceEnv(key, nil)
	if len(values) == 0 {
		return fallback
	}
	result := make([]snowflake.ID, 0, len(values))
	for _, value := range values {
		if id, err := snowflake.Parse(value); err == nil {
			result = append(result, id)
		}
	}
	return result
}

func parseStringSliceEnv(key string, fallback []string) []string {
	value := getenv(key, "")
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

type Config struct {
	Bot          BotConfig       `yaml:"bot"`
	Nodes        []NodeConfig    `yaml:"nodes"`
	MusicTracker TrackerConfig   `yaml:"music_tracker"`
	DB           DBConfig        `yaml:"database"`
	Log          LogConfig       `yaml:"log"`
	Site         SiteConfig      `yaml:"site"`
	Lyrics       LyricsConfig    `yaml:"lyrics"`
	Search       SearchConfig    `yaml:"search"`
	API          APIConfig       `yaml:"api"`
	AzuraCast    AzuraCastConfig `yaml:"azuracast"`
}

type SiteConfig struct {
	URL string `yaml:"url"`
}

type LyricsConfig struct {
	URL string `yaml:"url"`
}

type SearchConfig struct {
	Providers []string `yaml:"providers"`
}

type APIConfig struct {
	Enabled        bool           `yaml:"enabled"`
	Address        string         `yaml:"address"`
	PublicURL      string         `yaml:"public_url"`
	DashboardURL   string         `yaml:"dashboard_url"`
	AllowedOrigins []string       `yaml:"allowed_origins"`
	ClientID       string         `yaml:"client_id"`
	ClientSecret   string         `yaml:"client_secret"`
	CookieDomain   string         `yaml:"cookie_domain"`
	CookieSecure   bool           `yaml:"cookie_secure"`
	SessionTTL     time.Duration  `yaml:"session_ttl"`
	TrustProxy     bool           `yaml:"trust_proxy"`
	AdminUserIDs   []snowflake.ID `yaml:"admin_user_ids"`
}

type AzuraCastConfig struct {
	Enabled       bool          `yaml:"enabled"`
	URL           string        `yaml:"url"`
	APIKey        string        `yaml:"api_key"`
	Stations      []string      `yaml:"stations"`
	PollInterval  time.Duration `yaml:"poll_interval"`
	StreamBaseURL string        `yaml:"stream_base_url"`
}

type BotConfig struct {
	Token        string `yaml:"token"`
	Status       string `yaml:"status"`
	ActivityName string `yaml:"activity_name"`
	ActivityType string `yaml:"activity_type"`
}

type LogConfig struct {
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	FilePath  string `yaml:"file_path"`
	AddSource bool   `yaml:"add_source"`
	Env       string `yaml:"env"`
}

type NodeConfig struct {
	Name      string `yaml:"name"`
	Address   string `yaml:"address"`
	Password  string `yaml:"password"`
	Secure    bool   `yaml:"secure"`
	SessionID string `yaml:"session_id"`
	Location  string `yaml:"location"`
}

type TrackerConfig struct {
	Enabled        bool         `yaml:"enabled"`
	ChannelID      snowflake.ID `yaml:"channel_id"`
	GuildID        snowflake.ID `yaml:"guild_id"`
	HostAddress    string       `yaml:"host_address"`
	HttpPath       string       `yaml:"http_path"`
	WebsocketPath  string       `yaml:"websocket_path"`
	AllowedOrigins []string     `yaml:"allowed_origins"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}
