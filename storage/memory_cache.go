// pkg/storage/memory_cache.go

package storage

import (
	"sync"
	"time"
)

// Item represents a cache item
type Item struct {
	Value      string
	Expiration int64
}

// MemoryCache implements a simple in-memory cache
type MemoryCache struct {
	items map[string]Item
	mu    sync.RWMutex
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]Item),
	}

	// Start janitor to clean expired items
	go cache.janitor()

	return cache
}

// Set adds an item to the cache
func (c *MemoryCache) Set(key string, value string, expiry time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration int64
	if expiry > 0 {
		expiration = time.Now().Add(expiry).UnixNano()
	}

	c.items[key] = Item{
		Value:      value,
		Expiration: expiration,
	}
}

// Get retrieves an item from the cache
func (c *MemoryCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return "", false
	}

	// Check if item has expired
	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		return "", false
	}

	return item.Value, true
}

// Delete removes an item from the cache
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Has checks if an item exists in the cache
func (c *MemoryCache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return false
	}

	// Check if item has expired
	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		return false
	}

	return true
}

// janitor cleans up expired items
func (c *MemoryCache) janitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		c.deleteExpired()
	}
}

// deleteExpired removes expired items
func (c *MemoryCache) deleteExpired() {
	now := time.Now().UnixNano()

	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.items {
		if v.Expiration > 0 && now > v.Expiration {
			delete(c.items, k)
		}
	}
}
