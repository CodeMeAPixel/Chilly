package musicbot

import (
	"sync"

	"github.com/disgoorg/snowflake/v2"
)

type Hub struct {
	mu   sync.RWMutex
	subs map[snowflake.ID]map[chan struct{}]struct{}
	all  map[chan snowflake.ID]struct{}
}

func NewHub() *Hub {
	return &Hub{
		subs: make(map[snowflake.ID]map[chan struct{}]struct{}),
		all:  make(map[chan snowflake.ID]struct{}),
	}
}

func (h *Hub) SubscribeAll() (<-chan snowflake.ID, func()) {
	ch := make(chan snowflake.ID, 256)
	h.mu.Lock()
	h.all[ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.all, ch)
			h.mu.Unlock()
		})
	}
}

func (h *Hub) Subscribe(guildID snowflake.ID) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	if h.subs[guildID] == nil {
		h.subs[guildID] = make(map[chan struct{}]struct{})
	}
	h.subs[guildID][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs[guildID], ch)
			if len(h.subs[guildID]) == 0 {
				delete(h.subs, guildID)
			}
			h.mu.Unlock()
		})
	}
}

func (h *Hub) Publish(guildID snowflake.ID) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[guildID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	for ch := range h.all {
		select {
		case ch <- guildID:
		default:
		}
	}
}
