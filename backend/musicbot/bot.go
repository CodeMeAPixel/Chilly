package musicbot

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/CodeMeAPixel/Chilly/azuracast"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgolink/v3/disgolink"
)

type Bot struct {
	Cfg           Config
	Client        bot.Client
	Lavalink      disgolink.Client
	Db            *DB
	PlayerManager *PlayerManager
	Searcher      *Searcher
	Media         *MediaSigner
	Backfill      *LyricsBackfill
	Requests      *Requests
	Lyrics        *LyricsClient
	Nodes         *NodeSupervisor
	Radio         *azuracast.Service
	Stays         *StayStore
	Logs          *LogBuffer
	Version       string
	StartedAt     time.Time
}

func (b *Bot) Start(ctx context.Context) error {
	openCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := b.Client.OpenGateway(openCtx); err != nil {
		return err
	}

	configs := make([]disgolink.NodeConfig, 0, len(b.Cfg.Nodes))
	locations := make(map[string]string, len(b.Cfg.Nodes))
	for _, n := range b.Cfg.Nodes {
		if n.Location != "" {
			locations[n.Name] = n.Location
		}
		configs = append(configs, disgolink.NodeConfig{
			Name:      n.Name,
			Address:   n.Address,
			Password:  n.Password,
			Secure:    n.Secure,
			SessionID: n.SessionID,
		})
	}
	if len(configs) == 0 {
		return errors.New("no lavalink nodes configured (set NODE_ADDRESS or NODE_1_ADDRESS)")
	}

	b.Nodes = NewNodeSupervisor(b.Lavalink, b.PlayerManager, configs)
	b.Nodes.Rejoin = b.RejoinVoice
	b.Nodes.Locations = locations
	nodeCtx, cancelNodes := context.WithTimeout(ctx, 15*time.Second)
	defer cancelNodes()
	connected := b.Nodes.ConnectAll(nodeCtx)
	if connected == 0 {
		return errors.New("could not connect to any lavalink node")
	}
	slog.Info("lavalink nodes connected", slog.Int("connected", connected), slog.Int("configured", len(configs)))
	go b.Nodes.Run(ctx)

	b.StartedAt = time.Now()
	return nil
}
