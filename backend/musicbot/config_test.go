package musicbot

import "testing"

func TestReadConfigFromEnv(t *testing.T) {
	t.Setenv("BOT_TOKEN", "token-from-env")
	t.Setenv("NODE_NAME", "node-1")
	t.Setenv("NODE_ADDRESS", "localhost:2333")
	t.Setenv("NODE_PASSWORD", "secure-pass")
	t.Setenv("NODE_SECURE", "true")
	t.Setenv("MUSIC_TRACKER_ENABLED", "true")
	t.Setenv("MUSIC_TRACKER_CHANNEL_ID", "123456789012345678")
	t.Setenv("MUSIC_TRACKER_GUILD_ID", "876543210987654321")
	t.Setenv("MUSIC_TRACKER_HOST_ADDRESS", "https://tracker.example.com")
	t.Setenv("MUSIC_TRACKER_HTTP_PATH", "/stream")
	t.Setenv("MUSIC_TRACKER_WEBSOCKET_PATH", "/socket")
	t.Setenv("MUSIC_TRACKER_ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USERNAME", "chillcord")
	t.Setenv("DB_PASSWORD", "super-secret")
	t.Setenv("DB_NAME", "chillcord_db")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("LOG_FILE_PATH", "/tmp/chillcord.log")
	t.Setenv("LOG_ADD_SOURCE", "true")
	t.Setenv("LOG_ENV", "prod")

	cfg, err := ReadConfig(".env")
	if err != nil {
		t.Fatalf("ReadConfig returned error: %v", err)
	}

	if cfg.Bot.Token != "token-from-env" {
		t.Fatalf("expected bot token to be loaded from env, got %q", cfg.Bot.Token)
	}
	if len(cfg.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(cfg.Nodes))
	}
	if cfg.Nodes[0].Address != "localhost:2333" {
		t.Fatalf("expected lavalink address to be loaded from env, got %q", cfg.Nodes[0].Address)
	}
	if !cfg.MusicTracker.Enabled {
		t.Fatal("expected music tracker to be enabled from env")
	}
	if cfg.DB.Port != 5432 {
		t.Fatalf("expected DB port to be 5432, got %d", cfg.DB.Port)
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("expected log level to be debug, got %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Fatalf("expected log format to be json, got %q", cfg.Log.Format)
	}
}

func TestParseNodeLocations(t *testing.T) {
	t.Setenv("NODE_1_NAME", "node-1")
	t.Setenv("NODE_1_ADDRESS", "a:2333")
	t.Setenv("NODE_1_LOCATION", "  Frankfurt, DE ")
	t.Setenv("NODE_2_ADDRESS", "b:2333")

	nodes := parseNodesFromEnvironment(nil)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Location != "Frankfurt, DE" {
		t.Errorf("node-1 location = %q, want trimmed value", nodes[0].Location)
	}
	if nodes[1].Location != "" {
		t.Errorf("node-2 location = %q, want empty", nodes[1].Location)
	}
}
