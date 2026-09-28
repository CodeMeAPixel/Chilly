package musicbot

import (
	"sync"
	"time"
)

type ttlCache[V any] struct {
	mu      sync.Mutex
	ttl     time.Duration
	maxSize int
	items   map[string]ttlEntry[V]
}

type ttlEntry[V any] struct {
	value   V
	expires time.Time
}

func newTTLCache[V any](ttl time.Duration, maxSize int) *ttlCache[V] {
	return &ttlCache[V]{ttl: ttl, maxSize: maxSize, items: make(map[string]ttlEntry[V])}
}

func (c *ttlCache[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok || time.Now().After(entry.expires) {
		var zero V
		return zero, false
	}
	return entry.value, true
}

func (c *ttlCache[V]) Set(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if len(c.items) >= c.maxSize {
		for k, e := range c.items {
			if now.After(e.expires) {
				delete(c.items, k)
			}
		}
		for k := range c.items {
			if len(c.items) < c.maxSize {
				break
			}
			delete(c.items, k)
		}
	}
	c.items[key] = ttlEntry[V]{value: value, expires: now.Add(c.ttl)}
}
