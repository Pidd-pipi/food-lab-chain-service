package custody

import "sync"

// Cache is a small bounded LRU-style cache for assembled chains. Entries are
// returned as deep copies so a caller that annotates a cached chain cannot
// poison the cache for the next reader.
type Cache struct {
	mu         sync.Mutex
	entries    map[string]chainEntry
	maxEntries int
}

type chainEntry struct {
	chain *Chain
	seq   int
}

func NewCache(maxEntries int) *Cache {
	if maxEntries < 1 {
		maxEntries = 100
	}
	return &Cache{entries: map[string]chainEntry{}, maxEntries: maxEntries}
}

func (c *Cache) Get(key string) (*Chain, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	return entry.chain.Clone(), true
}

func (c *Cache) Put(key string, chain *Chain) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = chainEntry{chain: chain.Clone(), seq: len(c.entries)}
	if len(c.entries) > c.maxEntries {
		c.evictLocked()
	}
	return nil
}

func (c *Cache) evictLocked() {
	if len(c.entries) <= c.maxEntries {
		return
	}
	oldestKey := ""
	oldestSeq := int(^uint(0) >> 1)
	for key, entry := range c.entries {
		if entry.seq < oldestSeq {
			oldestSeq = entry.seq
			oldestKey = key
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *Cache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
