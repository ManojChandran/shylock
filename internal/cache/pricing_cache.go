package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"shylock/internal/interfaces"
)

// PricingCache provides caching for AWS pricing data to reduce API calls
type PricingCache struct {
	cache      map[string]*CacheEntry
	mutex      sync.RWMutex
	ttl        time.Duration
	maxEntries int
}

// CacheEntry represents a cached pricing result
type CacheEntry struct {
	Products  []interfaces.PricingProduct
	Timestamp time.Time
	Hits      int64
}

// NewPricingCache creates a new pricing cache
func NewPricingCache(ttl time.Duration, maxEntries int) *PricingCache {
	if ttl <= 0 {
		ttl = 15 * time.Minute // Default 15 minutes
	}
	if maxEntries <= 0 {
		maxEntries = 1000 // Default 1000 entries
	}

	return &PricingCache{
		cache:      make(map[string]*CacheEntry),
		ttl:        ttl,
		maxEntries: maxEntries,
	}
}

// Get retrieves cached pricing data if available and not expired
func (c *PricingCache) Get(serviceCode string, filters map[string]string) ([]interfaces.PricingProduct, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	key := c.generateKey(serviceCode, filters)
	entry, exists := c.cache[key]

	if !exists {
		return nil, false
	}

	// Check if entry is expired
	if time.Since(entry.Timestamp) > c.ttl {
		return nil, false
	}

	// Increment hit counter
	entry.Hits++

	return entry.Products, true
}

// Set stores pricing data in the cache
func (c *PricingCache) Set(serviceCode string, filters map[string]string, products []interfaces.PricingProduct) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := c.generateKey(serviceCode, filters)

	// Check if we need to evict entries
	if len(c.cache) >= c.maxEntries {
		c.evictLRU()
	}

	c.cache[key] = &CacheEntry{
		Products:  products,
		Timestamp: time.Now(),
		Hits:      0,
	}
}

// Clear removes all entries from the cache
func (c *PricingCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*CacheEntry)
}

// Stats returns cache statistics
func (c *PricingCache) Stats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var totalHits int64
	var expiredEntries int
	now := time.Now()

	for _, entry := range c.cache {
		totalHits += entry.Hits
		if now.Sub(entry.Timestamp) > c.ttl {
			expiredEntries++
		}
	}

	return CacheStats{
		TotalEntries:   len(c.cache),
		ExpiredEntries: expiredEntries,
		TotalHits:      totalHits,
		TTL:            c.ttl,
		MaxEntries:     c.maxEntries,
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalEntries   int
	ExpiredEntries int
	TotalHits      int64
	TTL            time.Duration
	MaxEntries     int
}

// generateKey creates a cache key from service code and filters
func (c *PricingCache) generateKey(serviceCode string, filters map[string]string) string {
	// Create a deterministic key from service code and filters
	hasher := sha256.New()
	hasher.Write([]byte(serviceCode))

	// Sort filters for consistent key generation
	var keys []string
	for key := range filters {
		keys = append(keys, key)
	}

	// Sort keys to ensure deterministic order
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		hasher.Write([]byte(fmt.Sprintf("%s=%s", key, filters[key])))
	}

	return hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars for shorter keys
}

// evictLRU removes the least recently used entry
func (c *PricingCache) evictLRU() {
	if len(c.cache) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	var lowestHits int64 = -1

	// Find entry with oldest timestamp and lowest hits
	for key, entry := range c.cache {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) ||
			(entry.Timestamp.Equal(oldestTime) && (lowestHits == -1 || entry.Hits < lowestHits)) {
			oldestKey = key
			oldestTime = entry.Timestamp
			lowestHits = entry.Hits
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// CleanupExpired removes expired entries from the cache
func (c *PricingCache) CleanupExpired() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var removed int
	now := time.Now()

	for key, entry := range c.cache {
		if now.Sub(entry.Timestamp) > c.ttl {
			delete(c.cache, key)
			removed++
		}
	}

	return removed
}

// CachedAWSClient wraps an AWS client with caching capabilities
type CachedAWSClient struct {
	client interfaces.AWSPricingClient
	cache  *PricingCache
}

// NewCachedAWSClient creates a new cached AWS client
func NewCachedAWSClient(client interfaces.AWSPricingClient, cache *PricingCache) interfaces.AWSPricingClient {
	return &CachedAWSClient{
		client: client,
		cache:  cache,
	}
}

// GetProducts retrieves products with caching
func (c *CachedAWSClient) GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]interfaces.PricingProduct, error) {
	// Try to get from cache first
	if products, found := c.cache.Get(serviceCode, filters); found {
		return products, nil
	}

	// Cache miss - fetch from AWS
	products, err := c.client.GetProducts(ctx, serviceCode, filters)
	if err != nil {
		return nil, err
	}

	// Store in cache for future use
	c.cache.Set(serviceCode, filters, products)

	return products, nil
}

// DescribeServices delegates to the underlying client (no caching needed for this)
func (c *CachedAWSClient) DescribeServices(ctx context.Context) ([]interfaces.ServiceInfo, error) {
	return c.client.DescribeServices(ctx)
}

// GetRegions delegates to the underlying client (no caching needed for this)
func (c *CachedAWSClient) GetRegions(ctx context.Context, serviceCode string) ([]string, error) {
	return c.client.GetRegions(ctx, serviceCode)
}
