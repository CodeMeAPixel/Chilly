package main

import (
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/api"
	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/CodeMeAPixel/Chilly/commands"
	"github.com/CodeMeAPixel/Chilly/handlers"
	"github.com/CodeMeAPixel/Chilly/musicbot"

	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/handler/middleware"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgolink/v3/disgolink"
)

//go:embed db/schema.sql
var DBschema string

var version = "dev"

type concurrent struct {
	bot.EventListener
}

func (c concurrent) OnEvent(event bot.Event) {
	goSafe(func() { c.EventListener.OnEvent(event) })
}

func goSafe(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from panic in event handler", slog.Any("panic", r), slog.String("stack", string(debug.Stack())))
			}
		}()
		fn()
	}()
}

type minLevelHandler struct {
	slog.Handler
	min slog.Level
}

func (h minLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.min && h.Handler.Enabled(ctx, level)
}

func (h minLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return minLevelHandler{Handler: h.Handler.WithAttrs(attrs), min: h.min}
}

func (h minLevelHandler) WithGroup(name string) slog.Handler {
	return minLevelHandler{Handler: h.Handler.WithGroup(name), min: h.min}
}

func libraryLogger() *slog.Logger {
	return slog.New(minLevelHandler{Handler: slog.Default().Handler(), min: slog.LevelInfo})
}

func setupLogger(cfg musicbot.LogConfig, logs *musicbot.LogBuffer) {
	var level slog.Level
	switch strings.ToLower(strings.TrimSpace(cfg.Level)) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				a.Value = slog.StringValue(t.UTC().Format(time.RFC3339))
			}
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					src.File = filepath.Base(src.File)
					a.Value = slog.AnyValue(src)
				}
			}
			return a
		},
	}

	var (
		handler slog.Handler
		w       io.Writer = os.Stdout
	)

	if cfg.Env == "dev" {
		handler = slog.NewTextHandler(w, opts)
	} else {
		var mw io.Writer = os.Stdout
		if cfg.FilePath != "" {
			if f, err := os.OpenFile(cfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
				mw = io.MultiWriter(os.Stdout, f)
			} else {
				slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})).
					Error("failed to open log file; using stdout only", "path", cfg.FilePath, "err", err)
			}
		}
		if cfg.Format == "json" {
			handler = slog.NewJSONHandler(mw, opts)
		} else {
			handler = slog.NewTextHandler(mw, opts)
		}
	}

	slog.SetDefault(slog.New(logs.Handler(handler)))

	slog.Info("logger initialized")
	slog.Info("startup",
		"env", cfg.Env,
		"pid", strconv.Itoa(os.Getpid()),
	)
}

func main() {
	cfg, err := musicbot.ReadConfig(".env")
	if err != nil {
		slog.Error("failed to read config file", slog.Any("err", err))
		os.Exit(1)
	}
	logs := musicbot.NewLogBuffer(1000, slog.LevelInfo)
	setupLogger(cfg.Log, logs)
	slog.Info("starting Chilly",
		"version", version,
		"disgo", disgo.Version,
		"disgolink", disgolink.Version,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	defer stop()

	b := &musicbot.Bot{Cfg: cfg, Logs: logs, Version: version}
	if cfg.AzuraCast.Enabled {
		if cfg.AzuraCast.URL == "" {
			slog.Error("AZURACAST_ENABLED is true but AZURACAST_URL is empty")
			os.Exit(1)
		}
		b.Radio = azuracast.NewService(azuracast.Config{
			URL:           cfg.AzuraCast.URL,
			APIKey:        cfg.AzuraCast.APIKey,
			Stations:      cfg.AzuraCast.Stations,
			PollInterval:  cfg.AzuraCast.PollInterval,
			StreamBaseURL: cfg.AzuraCast.StreamBaseURL,
		})
	}
	cmds := &commands.Commands{Bot: b}
	hdlr := handlers.New(b)

	r := handler.New()
	r.Use(func(next handler.Handler) handler.Handler {
		return func(e *handler.InteractionEvent) error {
			if i, ok := any(e.Interaction).(discord.ApplicationCommandInteraction); ok {
				data := i.SlashCommandInteractionData()
				musicbot.LogCommand(data.CommandPath(), e.GuildID().String(), e.User().ID.String())
			}
			return next(e)
		}
	})
	r.Use(middleware.GoErr(func(e *handler.InteractionEvent, err error) {
		if i, ok := any(e.Interaction).(discord.ApplicationCommandInteraction); ok {
			data := i.SlashCommandInteractionData()
			musicbot.LogCommandError(err, data.CommandPath(), e.GuildID().String(), e.User().ID.String())
		}
	}))
	r.SlashCommand("/join", cmds.Connect)
	r.SlashCommand("/leave", cmds.Disconnect)
	r.SlashCommand("/ping", cmds.Ping)
	r.SlashCommand("/help", cmds.Help)
	r.SlashCommand("/invite", cmds.Invite)
	r.SlashCommand("/loop", cmds.Loop)
	r.SlashCommand("/now", cmds.Now)
	r.SlashCommand("/lyrics", cmds.Lyrics)
	r.SlashCommand("/pause", cmds.Pause)
	r.SlashCommand("/play", cmds.Play)
	r.SlashCommand("/playlist", cmds.PlayPlaylist)
	r.Autocomplete("/playlist", cmds.PlaylistAutocomplete)
	r.SlashCommand("/queue", cmds.Queue)
	r.SlashCommand("/remove", cmds.RemoveQueueTrack)
	r.Autocomplete("/remove", cmds.RemoveQueueTrackAutocomplete)
	r.SlashCommand("/resume", cmds.Resume)
	r.SlashCommand("/search", cmds.Play)
	r.Autocomplete("/search", cmds.SearchAutocomplete)
	r.SlashCommand("/seek", cmds.Seek)
	r.SlashCommand("/shuffle", cmds.Shuffle)
	r.SlashCommand("/skip", cmds.Skip)
	r.SlashCommand("/stop", cmds.Stop)
	r.Route("/list", func(r handler.Router) {
		r.SlashCommand("/add", cmds.AddPlaylistTrack)
		r.Autocomplete("/add", cmds.AddPlaylistTrackAutocomplete)
		r.SlashCommand("/create", cmds.CreatePlaylist)
		r.SlashCommand("/delete", cmds.DeletePlaylist)
		r.Autocomplete("/delete", cmds.PlaylistAutocomplete)
		r.SlashCommand("/list", cmds.ListPlaylists)
		r.SlashCommand("/remove", cmds.RemovePlaylistTrack)
		r.Autocomplete("/remove", cmds.RemovePlaylistTrackAutocomplete)
	})

	commandCreates := commands.CommandCreates
	if b.Radio != nil {
		commandCreates = append(commandCreates, commands.RadioCommand)
		r.Route("/radio", func(r handler.Router) {
			r.SlashCommand("/play", cmds.RadioPlay)
			r.Autocomplete("/play", cmds.RadioStationAutocomplete)
			r.SlashCommand("/now", cmds.RadioNow)
			r.Autocomplete("/now", cmds.RadioStationAutocomplete)
			r.SlashCommand("/stations", cmds.RadioStations)
		})
		commandCreates = append(commandCreates, commands.StayCommand)
		commandCreates = append(commandCreates, commands.RequestCommands...)
		r.SlashCommand("/request", cmds.Request)
		r.Autocomplete("/request", cmds.RequestAutocomplete)
		r.SlashCommand("/suggest", cmds.Suggest)
		r.SlashCommand("/requests", cmds.MyRequests)
		r.Route("/247", func(r handler.Router) {
			r.SlashCommand("/on", cmds.StayOn)
			r.Autocomplete("/on", cmds.RadioStationAutocomplete)
			r.SlashCommand("/off", cmds.StayOff)
			r.SlashCommand("/status", cmds.StayStatus)
		})
	}

	presenceOpts := []gateway.PresenceOpt{}
	status := strings.TrimSpace(cfg.Bot.Status)
	if status == "" {
		status = string(discord.OnlineStatusOnline)
	}
	presenceOpts = append(presenceOpts, gateway.WithOnlineStatus(discord.OnlineStatus(status)))

	activityName := strings.TrimSpace(cfg.Bot.ActivityName)
	if activityName == "" {
		activityName = "24/7 radio • /radio"
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Bot.ActivityType)) {
	case "playing":
		presenceOpts = append(presenceOpts, gateway.WithPlayingActivity(activityName))
	case "watching":
		presenceOpts = append(presenceOpts, gateway.WithWatchingActivity(activityName))
	case "streaming":
		presenceOpts = append(presenceOpts, gateway.WithStreamingActivity(activityName, "https://www.twitch.tv/"))
	case "custom":
		presenceOpts = append(presenceOpts, gateway.WithCustomActivity(activityName))
	default:
		presenceOpts = append(presenceOpts, gateway.WithListeningActivity(activityName))
	}

	b.Client, err = disgo.New(cfg.Bot.Token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildVoiceStates,
			),
			gateway.WithPresenceOpts(presenceOpts...),
		),
		bot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagGuilds, cache.FlagVoiceStates),
		),

		bot.WithLogger(libraryLogger()),
		bot.WithRestClientConfigOpts(rest.WithHTTPClient(musicbot.NewDiscordHTTPClient())),
		bot.WithEventListeners(concurrent{r}),
		bot.WithEventListenerFunc(hdlr.OnVoiceStateUpdate),
		bot.WithEventListenerFunc(hdlr.OnVoiceServerUpdate),
		bot.WithEventListenerFunc(func(e *events.ComponentInteractionCreate) {
			goSafe(func() { hdlr.OnPlayerInteraction(e) })
		}),
	)
	if err != nil {
		slog.Error("failed to create disgo client", slog.Any("error", err))
		os.Exit(1)
	}

	if err = handler.SyncCommands(b.Client, commandCreates, nil); err != nil {
		slog.Error("failed to sync commands", slog.Any("error", err))
	}

	b.Lavalink = disgolink.New(b.Client.ApplicationID(),
		disgolink.WithLogger(libraryLogger()),
		disgolink.WithListenerFunc(hdlr.OnTrackStart),
		disgolink.WithListenerFunc(hdlr.OnTrackEnd),
		disgolink.WithListenerFunc(hdlr.OnTrackException),
		disgolink.WithListenerFunc(hdlr.OnTrackStuck),
		disgolink.WithListenerFunc(hdlr.OnWebSocketClosed),
	)
	b.PlayerManager = musicbot.NewPlayerManager(b.Lavalink)
	var library *musicbot.Library
	if b.Radio != nil {
		library = musicbot.NewLibrary(b.Radio.Client(), b.Radio.Shortcodes)
		library.HideFolders(cfg.AzuraCast.LibraryHiddenFolders)
		b.Media = newMediaSigner(cfg)
	}
	b.Searcher = musicbot.NewSearcher(b.Lavalink, library, b.Media)
	b.Lyrics = musicbot.NewLyricsClient(cfg.Lyrics.URL)

	b.Db, err = musicbot.NewDB(cfg.DB, DBschema)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("connected to database")
	defer b.Db.Close()

	b.Stays = musicbot.NewStayStore(b.Db)
	if err = b.Stays.Load(ctx); err != nil {
		slog.Error("failed to load 24/7 radio settings", slog.Any("error", err))
	}

	if err = b.Start(ctx); err != nil {
		slog.Error("failed to start bot", slog.Any("err", err))
		os.Exit(1)
	}
	defer b.Client.Close(context.TODO())

	checkNodeSources(ctx, b)
	go hdlr.RunPlayerMessageSync(ctx)

	if b.Radio != nil {
		b.Radio.OnSongChange(hdlr.OnRadioSongChange)
		go b.Radio.Run(ctx)
		if library != nil {
			b.Requests = musicbot.NewRequests(cfg.Requests, b.Db, b.Radio.Client(), library, b.SendDM, b.SiteURL)
			b.Radio.OnSongChange(func(np azuracast.NowPlaying) {
				songID := ""
				if np.NowPlaying != nil {
					songID = np.NowPlaying.Song.ID
				}
				reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				b.Requests.OnSongChange(reqCtx, np.Station.Shortcode, songID)
			})
			library.OnSync(func() {
				matchCtx, cancel := context.WithTimeout(ctx, time.Minute)
				defer cancel()
				if matched, err := b.Requests.MatchSuggestions(matchCtx); err != nil {
					slog.Warn("failed to match suggestions to the library", slog.Any("error", err))
				} else if matched > 0 {
					slog.Info("suggestions added to the library", slog.Int("count", matched))
				}
			})
			library.OnSync(func() {
				linkCtx, cancel := context.WithTimeout(ctx, time.Minute)
				defer cancel()
				linked, err := b.Db.LinkLegacyPlaylistTracks(linkCtx, library)
				if err != nil {
					slog.Warn("failed to link playlist songs to the library", slog.Any("error", err))
				} else if linked > 0 {
					slog.Info("linked playlist songs to the library", slog.Int("songs", linked))
				}
			})
			go library.Run(ctx, cfg.AzuraCast.LibrarySyncInterval)
			if cfg.AzuraCast.LyricsBackfill {
				b.Backfill = musicbot.NewLyricsBackfill(library, b.Lyrics, b.Radio.Client(), b.Db)
				go b.Backfill.Run(ctx, cfg.AzuraCast.LyricsBackfillInterval)
				slog.Info("lyrics backfill enabled", slog.Duration("interval", cfg.AzuraCast.LyricsBackfillInterval))
			}
		}
		go b.RunStaySupervisor(ctx)
		slog.Info("azuracast radio enabled", slog.String("url", cfg.AzuraCast.URL))
	}

	if cfg.API.Enabled {
		apiServer := api.New(b)
		go apiServer.Start()
		go apiServer.RunJanitor(ctx)
		go apiServer.RunStatus(ctx)
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			apiServer.Shutdown(shutdownCtx)
		}()
		if cfg.API.ClientSecret == "" || cfg.API.PublicURL == "" {
			slog.Warn("api is enabled but DISCORD_CLIENT_SECRET or API_PUBLIC_URL is missing; login will not work")
		}
	}

	slog.Info("Chilly is now running")

	if b.Cfg.MusicTracker.Enabled {
		wsServer := musicbot.NewWsServer(b.Cfg.MusicTracker.AllowedOrigins)
		go wsServer.Run()

		trackerHandler := handlers.TrackerHandler{
			ChannelID: b.Cfg.MusicTracker.ChannelID,
			GuildID:   b.Cfg.MusicTracker.GuildID,
			WsServer:  wsServer,
		}

		trackerServer := musicbot.NewTrackerServer(
			wsServer,
			trackerHandler.ServeHTTP,
			b.Cfg.MusicTracker.HostAddress,
			b.Cfg.MusicTracker.HttpPath,
			b.Cfg.MusicTracker.WebsocketPath)

		go trackerServer.Start()
		defer trackerServer.Close(context.TODO())

		b.Lavalink.AddListeners(
			disgolink.NewListenerFunc(trackerHandler.OnTrackStart),
			disgolink.NewListenerFunc(trackerHandler.OnTrackEnd),
			disgolink.NewListenerFunc(trackerHandler.OnPlayerUpdate),
		)
		slog.Info(
			"music tracker is enabled",
			slog.String("http", fmt.Sprintf("%s/%s",
				b.Cfg.MusicTracker.HostAddress,
				b.Cfg.MusicTracker.HttpPath)),
			slog.String("ws", fmt.Sprintf("%s/%s",
				b.Cfg.MusicTracker.HostAddress,
				b.Cfg.MusicTracker.WebsocketPath)),
		)
	}

	<-ctx.Done()
	slog.Info("shutting down")
}

func checkNodeSources(ctx context.Context, b *musicbot.Bot) {
	node := musicbot.BestNode(b.Lavalink)
	if node == nil {
		return
	}
	infoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	info, err := node.Info(infoCtx)
	if err != nil {
		slog.Warn("failed to read lavalink node info", slog.Any("error", err))
		return
	}

	plugins := make([]string, 0, len(info.Plugins))
	for _, p := range info.Plugins {
		plugins = append(plugins, p.Name+"@"+p.Version)
	}
	slog.Info("lavalink node capabilities",
		slog.String("node", node.Config().Name),
		slog.String("version", info.Version.Semver),
		slog.Any("sources", info.SourceManagers),
		slog.Any("plugins", plugins),
	)

	if !slices.Contains(info.SourceManagers, "http") {
		slog.Error("lavalink node has the http source disabled; stations and library songs can't play",
			slog.String("node", node.Config().Name))
	}
}

func newMediaSigner(cfg musicbot.Config) *musicbot.MediaSigner {
	if cfg.AzuraCast.APIKey == "" {
		slog.Warn("AZURACAST_API_KEY is not set; the music library and /play won't work")
	}
	if !cfg.API.Enabled {
		slog.Warn("the API is disabled; Lavalink can't fetch library songs until API_ENABLED=true")
	}
	baseURL := cfg.Media.BaseURL
	if baseURL == "" {
		baseURL = cfg.API.PublicURL
		if baseURL == "" {
			slog.Warn("MEDIA_BASE_URL is not set; library playback is disabled")
			return nil
		}
		slog.Warn("MEDIA_BASE_URL is not set, falling back to API_PUBLIC_URL; point it at this bot directly so seeking works",
			slog.String("url", baseURL))
	}
	secret := cfg.Media.SigningKey
	if secret == "" {
		secret = cfg.Bot.Token
	}
	return musicbot.NewMediaSigner(secret, baseURL)
}
