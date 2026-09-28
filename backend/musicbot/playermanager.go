package musicbot

import (
	"sync"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/snowflake/v2"
)

type PlayerManager struct {
	link disgolink.Client

	Hub *Hub

	players map[snowflake.ID]*Player
	mu      sync.RWMutex

	voiceMu      sync.Mutex
	voiceReady   map[snowflake.ID]snowflake.ID
	voiceWaiters map[snowflake.ID][]chan struct{}
	creds        map[snowflake.ID]voiceCreds
}

func NewPlayerManager(link disgolink.Client) *PlayerManager {
	return &PlayerManager{
		link:         link,
		Hub:          NewHub(),
		players:      make(map[snowflake.ID]*Player),
		voiceReady:   make(map[snowflake.ID]snowflake.ID),
		voiceWaiters: make(map[snowflake.ID][]chan struct{}),
		creds:        make(map[snowflake.ID]voiceCreds),
	}
}

func (pm *PlayerManager) GetPlayer(guildID snowflake.ID) (*Player, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	player, ok := pm.players[guildID]
	return player, ok
}

func (pm *PlayerManager) GetOrCreatePlayer(guildID snowflake.ID) *Player {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	player, ok := pm.players[guildID]
	if ok {
		return player
	}

	player = NewPlayer(guildID, pm.link, pm.Hub.Publish)
	pm.players[guildID] = player
	pm.Hub.Publish(guildID)
	return player
}

func (pm *PlayerManager) DeletePlayer(guildID snowflake.ID) {
	pm.mu.Lock()
	delete(pm.players, guildID)
	pm.mu.Unlock()
	pm.Hub.Publish(guildID)
}

func (pm *PlayerManager) ForEach(fn func(*Player)) {
	pm.mu.RLock()
	players := make([]*Player, 0, len(pm.players))
	for _, p := range pm.players {
		players = append(players, p)
	}
	pm.mu.RUnlock()
	for _, p := range players {
		fn(p)
	}
}

func (pm *PlayerManager) Count() (total int, playing int) {
	pm.ForEach(func(p *Player) {
		total++
		if p.IsPlaying() {
			playing++
		}
	})
	return
}

func (pm *PlayerManager) MarkVoiceReady(guildID, channelID snowflake.ID) {
	pm.voiceMu.Lock()
	defer pm.voiceMu.Unlock()
	pm.voiceReady[guildID] = channelID
	for _, ch := range pm.voiceWaiters[guildID] {
		close(ch)
	}
	delete(pm.voiceWaiters, guildID)
}

func (pm *PlayerManager) MoveVoice(guildID, channelID snowflake.ID) {
	pm.voiceMu.Lock()
	_, ready := pm.voiceReady[guildID]
	pm.voiceMu.Unlock()
	if ready {
		pm.MarkVoiceReady(guildID, channelID)
	}
}

func (pm *PlayerManager) ClearVoice(guildID snowflake.ID) {
	pm.voiceMu.Lock()
	defer pm.voiceMu.Unlock()
	delete(pm.voiceReady, guildID)
	delete(pm.creds, guildID)
}

func (pm *PlayerManager) VoiceChannel(guildID snowflake.ID) (snowflake.ID, bool) {
	pm.voiceMu.Lock()
	defer pm.voiceMu.Unlock()
	ch, ok := pm.voiceReady[guildID]
	return ch, ok
}

func (pm *PlayerManager) voiceWaiter(guildID snowflake.ID) chan struct{} {
	pm.voiceMu.Lock()
	defer pm.voiceMu.Unlock()
	ch := make(chan struct{})
	pm.voiceWaiters[guildID] = append(pm.voiceWaiters[guildID], ch)
	return ch
}
