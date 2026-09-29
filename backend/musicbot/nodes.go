package musicbot

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

func BestNode(link disgolink.Client) disgolink.Node {
	var (
		best      disgolink.Node
		bestScore float64
	)
	link.ForNodes(func(node disgolink.Node) {
		if node.Status() != disgolink.StatusConnected {
			return
		}
		score := nodeScore(node.Stats())
		if best == nil || score < bestScore {
			best, bestScore = node, score
		}
	})
	return best
}

func nodeScore(stats lavalink.Stats) float64 {
	score := float64(stats.PlayingPlayers) * 10
	if stats.CPU.Cores > 0 {
		score += stats.CPU.LavalinkLoad * 100
	}
	if stats.FrameStats != nil {
		score += float64(stats.FrameStats.Deficit+stats.FrameStats.Nulled) / 100
	}
	return score
}

func lavalinkPlayer(link disgolink.Client, guildID snowflake.ID) disgolink.Player {
	if p := link.ExistingPlayer(guildID); p != nil {
		return p
	}
	if node := BestNode(link); node != nil {
		return link.PlayerOnNode(node, guildID)
	}
	return link.Player(guildID)
}

func (pm *PlayerManager) LavalinkPlayer(guildID snowflake.ID) disgolink.Player {
	return lavalinkPlayer(pm.link, guildID)
}

type voiceCreds struct {
	channelID snowflake.ID
	sessionID string
	token     string
	endpoint  string
}

func (pm *PlayerManager) SetVoiceSession(guildID, channelID snowflake.ID, sessionID string) {
	pm.voiceMu.Lock()
	defer pm.voiceMu.Unlock()
	c := pm.creds[guildID]
	c.channelID, c.sessionID = channelID, sessionID
	pm.creds[guildID] = c
}

func (pm *PlayerManager) SetVoiceServer(guildID snowflake.ID, token, endpoint string) {
	pm.voiceMu.Lock()
	defer pm.voiceMu.Unlock()
	c := pm.creds[guildID]
	c.token, c.endpoint = token, endpoint
	pm.creds[guildID] = c
}

func (pm *PlayerManager) voiceCreds(guildID snowflake.ID) (voiceCreds, bool) {
	pm.voiceMu.Lock()
	defer pm.voiceMu.Unlock()
	c, ok := pm.creds[guildID]
	return c, ok && c.sessionID != "" && c.token != "" && c.endpoint != "" && c.channelID != 0
}

type NodeSupervisor struct {
	link      disgolink.Client
	pm        *PlayerManager
	configs   []disgolink.NodeConfig
	mu        sync.Mutex
	sessions  map[string]string
	health    map[snowflake.ID]*voiceHealth
	Rejoin    func(ctx context.Context, guildID snowflake.ID) error
	Locations map[string]string
}

func NewNodeSupervisor(link disgolink.Client, pm *PlayerManager, configs []disgolink.NodeConfig) *NodeSupervisor {
	return &NodeSupervisor{link: link, pm: pm, configs: configs, sessions: make(map[string]string), health: make(map[snowflake.ID]*voiceHealth)}
}

func (s *NodeSupervisor) ConnectAll(ctx context.Context) int {
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		connected int
	)
	for _, cfg := range s.configs {
		wg.Add(1)
		go func(cfg disgolink.NodeConfig) {
			defer wg.Done()
			if s.connect(ctx, cfg) {
				mu.Lock()
				connected++
				mu.Unlock()
			}
		}(cfg)
	}
	wg.Wait()
	return connected
}

func (s *NodeSupervisor) connect(ctx context.Context, cfg disgolink.NodeConfig) bool {
	node, err := s.link.AddNode(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect lavalink node", slog.String("node", cfg.Name), slog.Any("error", err))
		return false
	}
	s.mu.Lock()
	s.sessions[cfg.Name] = node.SessionID()
	s.mu.Unlock()
	slog.Info("connected lavalink node", slog.String("node", cfg.Name), slog.String("address", cfg.Address))
	return true
}

func (s *NodeSupervisor) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	retry := time.NewTicker(30 * time.Second)
	defer retry.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-retry.C:
			s.retryMissing(ctx)
		case <-ticker.C:
			s.check(ctx)
		}
	}
}

func (s *NodeSupervisor) retryMissing(ctx context.Context) {
	for _, cfg := range s.configs {
		if s.link.Node(cfg.Name) != nil {
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		s.connect(cctx, cfg)
		cancel()
	}
}

func (s *NodeSupervisor) check(ctx context.Context) {
	restarted := make(map[string]bool)
	s.link.ForNodes(func(node disgolink.Node) {
		if node.Status() != disgolink.StatusConnected {
			return
		}
		name := node.Config().Name
		s.mu.Lock()
		prev, seen := s.sessions[name]
		s.sessions[name] = node.SessionID()
		s.mu.Unlock()
		if seen && prev != "" && prev != node.SessionID() {
			slog.Warn("lavalink node restarted, restoring players", slog.String("node", name))
			restarted[name] = true
		}
	})

	s.pm.ForEach(func(player *Player) {
		lp := s.link.ExistingPlayer(player.GuildID())
		if lp == nil || lp.Node() == nil {
			return
		}
		node := lp.Node()
		if node.Status() != disgolink.StatusConnected {
			if target := BestNode(s.link); target != nil && target != node {
				s.migrate(ctx, player, target)
			}
			return
		}
		if restarted[node.Config().Name] {
			s.migrate(ctx, player, node)
			return
		}
		s.checkVoice(ctx, player, lp)
	})
}

func (s *NodeSupervisor) migrate(ctx context.Context, player *Player, target disgolink.Node) {
	guildID := player.GuildID()
	creds, ok := s.pm.voiceCreds(guildID)
	if !ok {
		slog.Warn("cannot migrate player without voice credentials", slog.String("guild_id", guildID.String()))
		return
	}

	old := s.link.ExistingPlayer(guildID)
	var (
		position = lavalink.Duration(0)
		volume   = 100
	)
	if old != nil {
		position = old.Position()
		volume = old.Volume()
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	s.link.RemovePlayer(guildID)
	np := s.link.PlayerOnNode(target, guildID)
	np.OnVoiceStateUpdate(ctx, &creds.channelID, creds.sessionID)
	np.OnVoiceServerUpdate(ctx, creds.token, creds.endpoint)

	if err := player.restoreOn(ctx, position, volume); err != nil {
		slog.Error("failed to restore player on new node",
			slog.String("guild_id", guildID.String()), slog.String("node", target.Config().Name), slog.Any("error", err))
		return
	}
	slog.Info("moved player to lavalink node",
		slog.String("guild_id", guildID.String()), slog.String("node", target.Config().Name))
}

type NodeInfo struct {
	Name             string  `json:"name"`
	Location         string  `json:"location,omitempty"`
	Status           string  `json:"status"`
	Players          int     `json:"players"`
	PlayingPlayers   int     `json:"playing_players"`
	UptimeMs         int64   `json:"uptime_ms"`
	CPULoad          float64 `json:"cpu_load"`
	SystemLoad       float64 `json:"system_load"`
	Cores            int     `json:"cores"`
	MemoryUsed       int64   `json:"memory_used"`
	MemoryAllocated  int64   `json:"memory_allocated"`
	MemoryReservable int64   `json:"memory_reservable"`
	FramesSent       int     `json:"frames_sent"`
	FramesNulled     int     `json:"frames_nulled"`
	FramesDeficit    int     `json:"frames_deficit"`
}

func nodeInfo(node disgolink.Node) NodeInfo {
	stats := node.Stats()
	info := NodeInfo{
		Name:             node.Config().Name,
		Status:           string(node.Status()),
		Players:          stats.Players,
		PlayingPlayers:   stats.PlayingPlayers,
		UptimeMs:         int64(stats.Uptime),
		CPULoad:          stats.CPU.LavalinkLoad,
		SystemLoad:       stats.CPU.SystemLoad,
		Cores:            stats.CPU.Cores,
		MemoryUsed:       int64(stats.Memory.Used),
		MemoryAllocated:  int64(stats.Memory.Allocated),
		MemoryReservable: int64(stats.Memory.Reservable),
	}
	if stats.FrameStats != nil {
		info.FramesSent = stats.FrameStats.Sent
		info.FramesNulled = stats.FrameStats.Nulled
		info.FramesDeficit = stats.FrameStats.Deficit
	}
	return info
}

func Nodes(link disgolink.Client) []NodeInfo {
	nodes := make([]NodeInfo, 0)
	link.ForNodes(func(node disgolink.Node) {
		nodes = append(nodes, nodeInfo(node))
	})
	return nodes
}

func (s *NodeSupervisor) Nodes() []NodeInfo {
	nodes := make([]NodeInfo, 0, len(s.configs))
	for _, cfg := range s.configs {
		info := NodeInfo{Name: cfg.Name, Status: string(disgolink.StatusDisconnected)}
		if node := s.link.Node(cfg.Name); node != nil {
			info = nodeInfo(node)
		}
		info.Location = s.Locations[cfg.Name]
		nodes = append(nodes, info)
	}
	return nodes
}

var ErrNodeNotFound = errors.New("lavalink node not found or not connected")

func (s *NodeSupervisor) MoveTo(ctx context.Context, player *Player, nodeName string) error {
	target := s.link.Node(nodeName)
	if target == nil || target.Status() != disgolink.StatusConnected {
		return ErrNodeNotFound
	}
	s.migrate(ctx, player, target)
	return nil
}
