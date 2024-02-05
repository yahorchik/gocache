package gocache

import (
	"errors"
	"sync"
	"time"
)

type Cache struct {
	mu                sync.RWMutex
	defaultExpiration time.Duration
	cleanupInterval   time.Duration
	items             map[string]Item
}

type Item struct {
	Value      interface{}
	Created    time.Time
	Expiration int64
}

func New(defaultExpiration, cleanupInterval time.Duration) *Cache {
	items := make(map[string]Item)

	cache := Cache{
		items:             items,
		defaultExpiration: defaultExpiration,
		cleanupInterval:   cleanupInterval,
	}

	if cleanupInterval > 0 {
		cache.startGC()
	}
	return &cache
}

func (c *Cache) Set(key string, value interface{}, duration time.Duration) {
	var expiration int64

	if duration == 0 {
		duration = c.defaultExpiration
	}

	if duration > 0 {
		expiration = time.Now().Add(duration).UnixNano()
	}
	c.mu.Lock()
	c.items[key] = Item{
		Value:      value,
		Expiration: expiration,
		Created:    time.Now(),
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()
	if !found {
		return nil, false
	}

	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return nil, false
		}
	}

	return item.Value, true
}

func (c *Cache) Delete(key string) error {
	c.mu.Lock()
	if _, found := c.items[key]; !found {
		return errors.New("key not found")
	}

	delete(c.items, key)
	c.mu.Unlock()
	return nil
}

func (c *Cache) startGC() {

	for {
		<-time.After(c.cleanupInterval)

		if c.items == nil {
			return
		}

		if keys := c.expiredKeys(); len(keys) != 0 {
			c.clearItems(keys)
		}
	}

}

func (c *Cache) expiredKeys() (keys []string) {
	c.mu.RLock()
	for k, i := range c.items {
		if time.Now().UnixNano() > i.Expiration && i.Expiration > 0 {
			keys = append(keys, k)
		}
	}
	c.mu.RUnlock()
	return
}

func (c *Cache) clearItems(keys []string) {
	c.mu.Lock()
	for _, k := range keys {
		delete(c.items, k)
	}
}
