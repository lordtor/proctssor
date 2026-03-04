// Package registry provides service registry functionality
package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

const (
	// DefaultMaxEntries is the default maximum number of cache entries
	DefaultMaxEntries = 1000
	// DefaultTTL is the default time-to-live for cache entries (5 minutes)
	DefaultTTL = 5 * time.Minute
)

// CacheEntry represents a single cache entry with metadata
type CacheEntry struct {
	Value      interface{}
	InsertedAt time.Time
	AccessedAt time.Time
}

// LRUCache is an LRU cache with TTL support for service registry
type LRUCache struct {
	mu         sync.RWMutex
	items      map[string]*CacheEntry
	list       *list // doubly linked list for LRU ordering
	maxEntries int
	ttl        time.Duration
	stats      CacheStats
}

// CacheStats holds cache statistics
type CacheStats struct {
	Hits   int64
	Misses int64
	Evicts int64
}

// list represents a doubly linked list for LRU ordering
type list struct {
	head *listNode
	tail *listNode
	len  int
}

type listNode struct {
	prev *listNode
	next *listNode
	key  string
}

// NewLRUCache creates a new LRU cache with the specified parameters
func NewLRUCache(maxEntries int, ttl time.Duration) *LRUCache {
	if maxEntries <= 0 {
		maxEntries = DefaultMaxEntries
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	c := &LRUCache{
		items:      make(map[string]*CacheEntry),
		list:       newList(),
		maxEntries: maxEntries,
		ttl:        ttl,
	}

	// Start background cleanup
	go c.cleanupLoop()

	return c
}

// newList creates a new doubly linked list
func newList() *list {
	l := &list{}
	l.head = &listNode{}
	l.tail = &listNode{}
	l.head.next = l.tail
	l.tail.prev = l.head
	return l
}

// Get retrieves an item from the cache
func (c *LRUCache) Get(ctx context.Context, key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.items[key]
	if !ok {
		c.stats.Misses++
		return nil, false
	}

	// Check TTL
	if time.Since(entry.InsertedAt) > c.ttl {
		c.stats.Misses++
		return nil, false
	}

	// Update access time and move to front
	c.list.MoveToFront(key)
	entry.AccessedAt = time.Now()
	c.stats.Hits++

	return entry.Value, true
}

// Set stores an item in the cache
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if entry, ok := c.items[key]; ok {
		entry.Value = value
		entry.InsertedAt = time.Now()
		entry.AccessedAt = time.Now()
		c.list.MoveToFront(key)
		return
	}

	// Evict oldest entry if at capacity
	if len(c.items) >= c.maxEntries {
		c.evictOldest()
	}

	// Add new entry
	entry := &CacheEntry{
		Value:      value,
		InsertedAt: time.Now(),
		AccessedAt: time.Now(),
	}
	c.items[key] = entry
	c.list.PushFront(key)
}

// Delete removes an item from the cache
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.items[key]; ok {
		c.list.Remove(key)
		delete(c.items, key)
	}
}

// Clear removes all items from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheEntry)
	c.list = newList()
}

// Size returns the current number of items in the cache
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Stats returns cache statistics
func (c *LRUCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// HitRatio returns the cache hit ratio
func (c *LRUCache) HitRatio() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total)
}

// evictOldest removes the least recently used item
func (c *LRUCache) evictOldest() {
	key := c.list.Back()
	if key != "" {
		c.list.Remove(key)
		delete(c.items, key)
		c.stats.Evicts++
	}
}

// cleanupLoop runs background cleanup of expired entries
func (c *LRUCache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

// cleanupExpired removes all expired entries
func (c *LRUCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.items {
		if now.Sub(entry.InsertedAt) > c.ttl {
			c.list.Remove(key)
			delete(c.items, key)
		}
	}
}

// =============================================================================
// List operations for LRU tracking
// =============================================================================

// PushFront adds a new key to the front of the list
func (l *list) PushFront(key string) {
	node := &listNode{key: key}
	node.next = l.head.next
	node.prev = l.head
	l.head.next.prev = node
	l.head.next = node
	l.len++
}

// MoveToFront moves an existing key to the front
func (l *list) MoveToFront(key string) {
	l.Remove(key)
	l.PushFront(key)
}

// Remove removes a key from the list
func (l *list) Remove(key string) {
	node := l.findNode(key)
	if node != nil {
		node.prev.next = node.next
		node.next.prev = node.prev
		l.len--
	}
}

// Back returns the least recently used key
func (l *list) Back() string {
	if l.tail.prev == l.head {
		return ""
	}
	return l.tail.prev.key
}

// findNode finds a node by key (linear search - not efficient for large lists)
func (l *list) findNode(key string) *listNode {
	for node := l.head.next; node != l.tail; node = node.next {
		if node.key == key {
			return node
		}
	}
	return nil
}

// =============================================================================
// Cache with PostgreSQL NOTIFY support
// =============================================================================

// CachedRegistryRepository wraps RegistryRepository with caching
type CachedRegistryRepository struct {
	repo  RegistryRepository
	cache *LRUCache
}

// NewCachedRegistryRepository creates a new cached registry repository
func NewCachedRegistryRepository(repo RegistryRepository, cache *LRUCache) *CachedRegistryRepository {
	return &CachedRegistryRepository{
		repo:  repo,
		cache: cache,
	}
}

// GetCached returns the cached repository
func (r *CachedRegistryRepository) GetCache() *LRUCache {
	return r.cache
}

// Discover finds active services by type (with cache)
func (r *CachedRegistryRepository) Discover(ctx context.Context, serviceType string) ([]*Service, error) {
	// Try cache first
	key := fmt.Sprintf("discover:%s", serviceType)
	if cached, ok := r.cache.Get(ctx, key); ok {
		if services, ok := cached.([]*Service); ok {
			return services, nil
		}
	}

	// Fetch from database
	services, err := r.repo.Discover(ctx, serviceType)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cache.Set(key, services)
	return services, nil
}

// DiscoverByName finds a service by name (with cache)
func (r *CachedRegistryRepository) DiscoverByName(ctx context.Context, name string) (*Service, error) {
	// Try cache first
	key := fmt.Sprintf("name:%s", name)
	if cached, ok := r.cache.Get(ctx, key); ok {
		if service, ok := cached.(*Service); ok {
			return service, nil
		}
	}

	// Fetch from database
	service, err := r.repo.DiscoverByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cache.Set(key, service)
	return service, nil
}

// ListAll lists all registered services (with cache)
func (r *CachedRegistryRepository) ListAll(ctx context.Context) ([]*Service, error) {
	// Try cache first
	key := "list:all"
	if cached, ok := r.cache.Get(ctx, key); ok {
		if services, ok := cached.([]*Service); ok {
			return services, nil
		}
	}

	// Fetch from database
	services, err := r.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result
	r.cache.Set(key, services)
	return services, nil
}

// Register registers a new service (invalidates cache)
func (r *CachedRegistryRepository) Register(ctx context.Context, service *Service) error {
	err := r.repo.Register(ctx, service)
	if err != nil {
		return err
	}

	// Invalidate related cache entries
	r.invalidateCache(service.Name)
	return nil
}

// Heartbeat updates the heartbeat timestamp for a service
func (r *CachedRegistryRepository) Heartbeat(ctx context.Context, serviceID string) error {
	return r.repo.Heartbeat(ctx, serviceID)
}

// Unregister removes a service from the registry (invalidates cache)
func (r *CachedRegistryRepository) Unregister(ctx context.Context, serviceID string) error {
	err := r.repo.Unregister(ctx, serviceID)
	if err != nil {
		return err
	}

	// Invalidate all cache entries
	r.cache.Clear()
	return nil
}

// invalidateCache invalidates cache entries for a specific service
func (r *CachedRegistryRepository) invalidateCache(serviceName string) {
	// Invalidate name-based cache
	r.cache.Delete(fmt.Sprintf("name:%s", serviceName))

	// For discover cache, we need to clear since we don't know all service types
	// This is a trade-off - we could store service types with each service
	r.cache.Delete("list:all")
}

// =============================================================================
// Cache Warmer
// =============================================================================

// CacheWarmer preloads cache from database
type CacheWarmer struct {
	cache *LRUCache
	repo  RegistryRepository
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cache *LRUCache, repo RegistryRepository) *CacheWarmer {
	return &CacheWarmer{
		cache: cache,
		repo:  repo,
	}
}

// Warmup loads all services into the cache
func (w *CacheWarmer) Warmup(ctx context.Context) error {
	services, err := w.repo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load services for cache warmup: %w", err)
	}

	for _, svc := range services {
		// Cache by name
		w.cache.Set(fmt.Sprintf("name:%s", svc.Name), svc)
	}

	return nil
}

// =============================================================================
// PostgreSQL NOTIFY Listener
// =============================================================================

// RegistryChangeListener listens for PostgreSQL NOTIFY events
type RegistryChangeListener struct {
	cache *LRUCache
	db    *sql.DB
	done  chan struct{}
	wg    sync.WaitGroup
}

// NewRegistryChangeListener creates a new registry change listener
func NewRegistryChangeListener(cache *LRUCache) *RegistryChangeListener {
	return &RegistryChangeListener{
		cache: cache,
		done:  make(chan struct{}),
	}
}

// Start begins listening for change notifications
func (l *RegistryChangeListener) Start(ctx context.Context, db *sql.DB) error {
	l.db = db

	// Listen for registry_changes notification
	_, err := db.ExecContext(ctx, "LISTEN registry_changes")
	if err != nil {
		return fmt.Errorf("failed to listen for registry changes: %w", err)
	}

	// Start notification handler
	l.wg.Add(1)
	go l.handleNotifications(ctx)

	return nil
}

// Stop stops the listener
func (l *RegistryChangeListener) Stop() {
	close(l.done)
	l.wg.Wait()
}

// handleNotifications processes PostgreSQL notifications in a loop
func (l *RegistryChangeListener) handleNotifications(ctx context.Context) {
	defer l.wg.Done()

	for {
		select {
		case <-l.done:
			return
		case <-ctx.Done():
			return
		default:
			// Check for notifications with timeout
			if l.db != nil {
				var notificationID int

				// Use PostgreSQL notification polling
				// In production, you might use pgx or a library with proper notification support
				err := l.db.QueryRowContext(ctx,
					"SELECT pg_notification_queue_usage()").Scan(&notificationID)
				if err != nil {
					time.Sleep(100 * time.Millisecond)
					continue
				}

				// If there are notifications, process them
				if notificationID > 0 {
					// Would need pgx for proper async notification handling
					// For now, we rely on TTL cleanup
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// HandleNotification processes a registry change notification
func (l *RegistryChangeListener) HandleNotification(payload []byte) {
	var notification struct {
		ServiceID   string `json:"service_id"`
		ServiceName string `json:"service_name"`
		Operation   string `json:"operation"`
	}

	if err := json.Unmarshal(payload, &notification); err != nil {
		return
	}

	// Invalidate cache based on operation
	switch notification.Operation {
	case "INSERT", "UPDATE", "DELETE":
		l.cache.Delete(fmt.Sprintf("name:%s", notification.ServiceName))
		l.cache.Delete("list:all")
	}
}

// =============================================================================
// Legacy Cache compatibility (wraps LRUCache)
// =============================================================================

// Cache is a simple in-memory cache for service registry (legacy interface)
type Cache struct {
	lru *LRUCache
}

// NewCache creates a new cache (legacy interface)
func NewCache() *Cache {
	return &Cache{
		lru: NewLRUCache(DefaultMaxEntries, DefaultTTL),
	}
}

// Get retrieves an item from the cache (legacy interface)
func (c *Cache) Get(ctx context.Context, name string) (interface{}, bool) {
	return c.lru.Get(ctx, name)
}

// Set stores an item in the cache (legacy interface)
func (c *Cache) Set(key string, value interface{}) {
	c.lru.Set(key, value)
}

// Delete removes an item from the cache (legacy interface)
func (c *Cache) Delete(key string) {
	c.lru.Delete(key)
}

// Clear removes all items from the cache (legacy interface)
func (c *Cache) Clear() {
	c.lru.Clear()
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	return c.lru.Stats()
}

// HitRatio returns the cache hit ratio
func (c *Cache) HitRatio() float64 {
	return c.lru.HitRatio()
}

// =============================================================================
// Legacy CacheUpdater for backward compatibility
// =============================================================================

// CacheUpdater updates cache from registry
type CacheUpdater struct {
	cache *LRUCache
	repo  RegistryRepository
}

// NewCacheUpdater creates a new cache updater
func NewCacheUpdater(cache *Cache, repo RegistryRepository) *CacheUpdater {
	return &CacheUpdater{
		cache: cache.lru,
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
