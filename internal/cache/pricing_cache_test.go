package cache

import (
	"context"
	"testing"
	"time"

	"shylock/internal/interfaces"
)

// MockAWSClient for testing
type MockAWSClient struct {
	products  []interfaces.PricingProduct
	err       error
	callCount int
}

func (m *MockAWSClient) GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]interfaces.PricingProduct, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.products, nil
}

func (m *MockAWSClient) DescribeServices(ctx context.Context) ([]interfaces.ServiceInfo, error) {
	return []interfaces.ServiceInfo{}, nil
}

func (m *MockAWSClient) GetRegions(ctx context.Context, serviceCode string) ([]string, error) {
	return []string{"us-east-1", "us-west-2"}, nil
}

func TestNewPricingCache(t *testing.T) {
	cache := NewPricingCache(10*time.Minute, 100)

	if cache == nil {
		t.Fatal("Expected cache but got nil")
	}

	if cache.ttl != 10*time.Minute {
		t.Errorf("Expected TTL 10m, got %v", cache.ttl)
	}

	if cache.maxEntries != 100 {
		t.Errorf("Expected max entries 100, got %d", cache.maxEntries)
	}
}

func TestNewPricingCache_Defaults(t *testing.T) {
	cache := NewPricingCache(0, 0) // Should use defaults

	if cache.ttl != 15*time.Minute {
		t.Errorf("Expected default TTL 15m, got %v", cache.ttl)
	}

	if cache.maxEntries != 1000 {
		t.Errorf("Expected default max entries 1000, got %d", cache.maxEntries)
	}
}

func TestPricingCache_SetAndGet(t *testing.T) {
	cache := NewPricingCache(10*time.Minute, 100)

	serviceCode := "AmazonEC2"
	filters := map[string]string{
		"instanceType": "t3.micro",
		"region":       "us-east-1",
	}

	products := []interfaces.PricingProduct{
		{SKU: "TEST001", ServiceCode: "AmazonEC2"},
	}

	// Initially should not be in cache
	_, found := cache.Get(serviceCode, filters)
	if found {
		t.Errorf("Expected cache miss, but found entry")
	}

	// Set in cache
	cache.Set(serviceCode, filters, products)

	// Should now be in cache
	cachedProducts, found := cache.Get(serviceCode, filters)
	if !found {
		t.Errorf("Expected cache hit, but got miss")
	}

	if len(cachedProducts) != len(products) {
		t.Errorf("Expected %d products, got %d", len(products), len(cachedProducts))
	}

	if cachedProducts[0].SKU != products[0].SKU {
		t.Errorf("Expected SKU %s, got %s", products[0].SKU, cachedProducts[0].SKU)
	}
}

func TestPricingCache_Expiration(t *testing.T) {
	cache := NewPricingCache(100*time.Millisecond, 100) // Very short TTL

	serviceCode := "AmazonEC2"
	filters := map[string]string{"instanceType": "t3.micro"}
	products := []interfaces.PricingProduct{{SKU: "TEST001"}}

	// Set in cache
	cache.Set(serviceCode, filters, products)

	// Should be in cache immediately
	_, found := cache.Get(serviceCode, filters)
	if !found {
		t.Errorf("Expected cache hit immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should now be expired
	_, found = cache.Get(serviceCode, filters)
	if found {
		t.Errorf("Expected cache miss after expiration")
	}
}

func TestPricingCache_Stats(t *testing.T) {
	cache := NewPricingCache(10*time.Minute, 100)

	// Initially empty
	stats := cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries, got %d", stats.TotalEntries)
	}

	if stats.TotalHits != 0 {
		t.Errorf("Expected 0 hits, got %d", stats.TotalHits)
	}

	// Add some entries
	cache.Set("EC2", map[string]string{"type": "t3.micro"}, []interfaces.PricingProduct{{SKU: "1"}})
	cache.Set("S3", map[string]string{"class": "STANDARD"}, []interfaces.PricingProduct{{SKU: "2"}})

	// Get some hits
	cache.Get("EC2", map[string]string{"type": "t3.micro"})
	cache.Get("EC2", map[string]string{"type": "t3.micro"})

	stats = cache.Stats()
	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.TotalEntries)
	}

	if stats.TotalHits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.TotalHits)
	}
}

func TestPricingCache_Clear(t *testing.T) {
	cache := NewPricingCache(10*time.Minute, 100)

	// Add entries
	cache.Set("EC2", map[string]string{"type": "t3.micro"}, []interfaces.PricingProduct{{SKU: "1"}})
	cache.Set("S3", map[string]string{"class": "STANDARD"}, []interfaces.PricingProduct{{SKU: "2"}})

	// Verify entries exist
	stats := cache.Stats()
	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 entries before clear, got %d", stats.TotalEntries)
	}

	// Clear cache
	cache.Clear()

	// Verify cache is empty
	stats = cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.TotalEntries)
	}
}

func TestPricingCache_Eviction(t *testing.T) {
	cache := NewPricingCache(10*time.Minute, 2) // Small cache for testing eviction

	// Fill cache to capacity
	cache.Set("EC2", map[string]string{"type": "t3.micro"}, []interfaces.PricingProduct{{SKU: "1"}})
	cache.Set("S3", map[string]string{"class": "STANDARD"}, []interfaces.PricingProduct{{SKU: "2"}})

	// Verify cache is full
	stats := cache.Stats()
	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.TotalEntries)
	}

	// Add one more entry, should trigger eviction
	cache.Set("RDS", map[string]string{"engine": "mysql"}, []interfaces.PricingProduct{{SKU: "3"}})

	// Cache should still have 2 entries (one evicted)
	stats = cache.Stats()
	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 entries after eviction, got %d", stats.TotalEntries)
	}
}

func TestPricingCache_CleanupExpired(t *testing.T) {
	cache := NewPricingCache(50*time.Millisecond, 100) // Very short TTL

	// Add entries
	cache.Set("EC2", map[string]string{"type": "t3.micro"}, []interfaces.PricingProduct{{SKU: "1"}})
	cache.Set("S3", map[string]string{"class": "STANDARD"}, []interfaces.PricingProduct{{SKU: "2"}})

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Cleanup expired entries
	removed := cache.CleanupExpired()

	if removed != 2 {
		t.Errorf("Expected 2 expired entries removed, got %d", removed)
	}

	stats := cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", stats.TotalEntries)
	}
}

func TestCachedAWSClient(t *testing.T) {
	mockClient := &MockAWSClient{
		products: []interfaces.PricingProduct{{SKU: "TEST001"}},
	}

	cache := NewPricingCache(10*time.Minute, 100)
	cachedClient := NewCachedAWSClient(mockClient, cache)

	ctx := context.Background()
	serviceCode := "AmazonEC2"
	filters := map[string]string{"instanceType": "t3.micro"}

	// First call should hit the underlying client
	products1, err := cachedClient.GetProducts(ctx, serviceCode, filters)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 call to underlying client, got %d", mockClient.callCount)
	}

	// Second call should hit the cache
	products2, err := cachedClient.GetProducts(ctx, serviceCode, filters)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mockClient.callCount != 1 {
		t.Errorf("Expected still 1 call to underlying client (cache hit), got %d", mockClient.callCount)
	}

	// Results should be the same
	if len(products1) != len(products2) {
		t.Errorf("Expected same number of products, got %d vs %d", len(products1), len(products2))
	}

	if products1[0].SKU != products2[0].SKU {
		t.Errorf("Expected same SKU, got %s vs %s", products1[0].SKU, products2[0].SKU)
	}
}

func TestCachedAWSClient_Passthrough(t *testing.T) {
	mockClient := &MockAWSClient{}
	cache := NewPricingCache(10*time.Minute, 100)
	cachedClient := NewCachedAWSClient(mockClient, cache)

	ctx := context.Background()

	// Test DescribeServices passthrough
	_, err := cachedClient.DescribeServices(ctx)
	if err != nil {
		t.Errorf("Unexpected error in DescribeServices: %v", err)
	}

	// Test GetRegions passthrough
	_, err = cachedClient.GetRegions(ctx, "AmazonEC2")
	if err != nil {
		t.Errorf("Unexpected error in GetRegions: %v", err)
	}
}

func TestPricingCache_GenerateKey(t *testing.T) {
	cache := NewPricingCache(10*time.Minute, 100)

	// Same inputs should generate same key
	key1 := cache.generateKey("EC2", map[string]string{"type": "t3.micro", "region": "us-east-1"})
	key2 := cache.generateKey("EC2", map[string]string{"type": "t3.micro", "region": "us-east-1"})

	if key1 != key2 {
		t.Errorf("Expected same key for same inputs, got %s vs %s", key1, key2)
	}

	// Different inputs should generate different keys
	key3 := cache.generateKey("EC2", map[string]string{"type": "t3.small", "region": "us-east-1"})

	if key1 == key3 {
		t.Errorf("Expected different keys for different inputs, but got same: %s", key1)
	}

	// Key should be reasonably short
	if len(key1) > 20 {
		t.Errorf("Expected key length <= 20, got %d", len(key1))
	}
}
