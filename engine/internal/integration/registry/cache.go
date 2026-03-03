// Package registry provides service registry functionality
package registry

import (
	"context"
	"sync"
)

// Cache is a simple in-memory cache for service registry
type Cache struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

// NewCache creates a new cache
func NewCache() *Cache {
	return &Cache{
		items: make(map[string]interface{}),
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(ctx context.Context, name string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.items[name]
	return val, ok
}

// Set stores an item in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]interface{})
}

// CacheUpdater updates cache from registry
type CacheUpdater struct {
	cache *Cache
	repo  RegistryRepository
}

// NewCacheUpdater creates a new cache updater
func NewCacheUpdater(cache *Cache, repo RegistryRepository) *CacheUpdater {
	return &CacheUpdater{
		cache: cache,
		repo:  repo,
	}
}

// Refresh updates cache from database
func (u *CacheUpdater) Refresh(ctx context.Context) error {
	services, err := u.repo.ListAll(ctx)
	if err != nil {
		return err
	}

	for _, svc := range services {
		u.cache.Set(svc.Name, svc)
	}

	return nil
}
