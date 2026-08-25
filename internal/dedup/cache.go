package dedup

import (
	"sync"

	"github.com/cespare/xxhash/v2"
)

type Cache struct {
	mu     sync.Mutex
	keys   map[uint64]bool
	groups map[string]map[uint64]bool
}

func NewCache() *Cache {
	return &Cache{
		keys:   make(map[uint64]bool),
		groups: make(map[string]map[uint64]bool),
	}
}

func (c *Cache) Key(parts ...string) uint64 {
	digest := xxhash.New()
	for _, part := range parts {
		_, _ = digest.WriteString(part)
		_, _ = digest.Write([]byte{0})
	}
	return digest.Sum64()
}

func (c *Cache) Mark(key uint64, group string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys[key] = true
	groupKeys := c.groups[group]
	if groupKeys == nil {
		groupKeys = make(map[uint64]bool)
		c.groups[group] = groupKeys
	}
	groupKeys[key] = true
}

func (c *Cache) Check(key uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.keys[key]
}

func (c *Cache) ClearGroup(group string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.groups[group] {
		delete(c.keys, key)
	}
	delete(c.groups, group)
}
