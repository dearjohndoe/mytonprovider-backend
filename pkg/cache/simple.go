package cache

import (
	"sync"
	"time"
)

type item struct {
	value      interface{}
	expiryTime time.Time
}

type SimpleCache struct {
	mu    sync.Mutex
	items map[string]item
	ttl   time.Duration
}

func NewSimpleCache(ttl time.Duration) *SimpleCache {
	c := &SimpleCache{
		items: make(map[string]item),
		ttl:   ttl,
	}
	go c.cleanup()
	return c
}

func (c *SimpleCache) GetAll() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	result := make(map[string]interface{}, len(c.items))
	for k, it := range c.items {
		if now.Before(it.expiryTime) {
			result[k] = it.value
		}
	}
	return result
}

// Set stores a value with the given key and resets its TTL.
func (c *SimpleCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = item{
		value:      value,
		expiryTime: time.Now().Add(c.ttl),
	}
}

// Gets the value for the given key if present and not expired.
func (c *SimpleCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.items[key]
	if !ok || time.Now().After(it.expiryTime) {
		delete(c.items, key)
		return nil, false
	}
	return it.value, true
}

// Release gets and deletes the value for the given key if present and not expired.
func (c *SimpleCache) Release(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.items[key]
	if !ok || time.Now().After(it.expiryTime) {
		delete(c.items, key)
		return nil, false
	}
	delete(c.items, key)
	return it.value, true
}

// cleanup periodically removes expired items.
func (c *SimpleCache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, it := range c.items {
			if now.After(it.expiryTime) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
