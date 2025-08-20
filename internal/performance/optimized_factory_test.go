package performance

import (
	"context"
	"testing"
	"time"

	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// MockAWSClient for testing
type MockAWSClient struct {
	products  []interfaces.PricingProduct
	err       error
	callCount int
	delay     time.Duration
}

func (m *MockAWSClient) GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]interfaces.PricingProduct, error) {
	m.callCount++
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
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

func createTestConfig() *models.EstimationConfig {
	return &models.EstimationConfig{
		Version: "1.0",
		Resources: []models.ResourceSpec{
			{
				Type:   "EC2",
				Name:   "web-server-1",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
					"count":        1,
				},
			},
			{
				Type:   "EC2",
				Name:   "web-server-2",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.small",
					"count":        2,
				},
			},
			{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
					"sizeGB":       100,
				},
			},
		},
		Options: models.ConfigOptions{
			Currency: "USD",
		},
	}
}

func createMockProducts() []interfaces.PricingProduct {
	return []interfaces.PricingProduct{
		{
			SKU:           "TEST001",
			ProductFamily: "Compute Instance",
			ServiceCode:   "AmazonEC2",
			Attributes: map[string]string{
				"instanceType": "t3.micro",
				"location":     "US East (N. Virginia)",
			},
			Terms: map[string]interface{}{
				"OnDemand": map[string]interface{}{
					"TEST001.JRTCKXETXF": map[string]interface{}{
						"priceDimensions": map[string]interface{}{
							"TEST001.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
								"pricePerUnit": map[string]interface{}{
									"USD": "0.0104",
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestDefaultOptimizedConfig(t *testing.T) {
	config := DefaultOptimizedConfig()

	if config == nil {
		t.Fatal("Expected config but got nil")
	}

	if config.MaxConcurrency <= 0 {
		t.Errorf("Expected positive MaxConcurrency, got %d", config.MaxConcurrency)
	}

	if !config.EnableCaching {
		t.Errorf("Expected EnableCaching to be true")
	}

	if !config.EnableBatching {
		t.Errorf("Expected EnableBatching to be true")
	}

	if config.BatchSize <= 0 {
		t.Errorf("Expected positive BatchSize, got %d", config.BatchSize)
	}

	if config.CacheTTL <= 0 {
		t.Errorf("Expected positive CacheTTL, got %v", config.CacheTTL)
	}
}

func TestNewOptimizedFactory(t *testing.T) {
	mockClient := &MockAWSClient{products: createMockProducts()}

	factory := NewOptimizedFactory(mockClient, nil) // Should use default config

	if factory == nil {
		t.Fatal("Expected factory but got nil")
	}

	if factory.Factory == nil {
		t.Fatal("Expected base factory but got nil")
	}

	config := factory.GetPerformanceConfig()
	if config.MaxConcurrency <= 0 {
		t.Errorf("Expected positive MaxConcurrency, got %d", config.MaxConcurrency)
	}
}

func TestOptimizedFactory_EstimateFromConfigOptimized_Sequential(t *testing.T) {
	mockClient := &MockAWSClient{products: createMockProducts()}

	config := &OptimizedFactoryConfig{
		MaxConcurrency: 1, // Force sequential processing
		EnableCaching:  false,
		EnableBatching: false,
		BatchSize:      10,
	}

	factory := NewOptimizedFactory(mockClient, config)

	testConfig := &models.EstimationConfig{
		Version: "1.0",
		Resources: []models.ResourceSpec{
			{
				Type:   "EC2",
				Name:   "test-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
					"count":        1,
				},
			},
		},
	}

	ctx := context.Background()
	result, err := factory.EstimateFromConfigOptimized(ctx, testConfig)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result but got nil")
	}

	if len(result.ResourceCosts) != 1 {
		t.Errorf("Expected 1 resource cost, got %d", len(result.ResourceCosts))
	}
}

func TestOptimizedFactory_EstimateFromConfigOptimized_Concurrent(t *testing.T) {
	mockClient := &MockAWSClient{
		products: createMockProducts(),
		delay:    10 * time.Millisecond, // Small delay to test concurrency
	}

	config := &OptimizedFactoryConfig{
		MaxConcurrency: 4,
		EnableCaching:  false,
		EnableBatching: false,
		BatchSize:      10,
	}

	factory := NewOptimizedFactory(mockClient, config)
	testConfig := createTestConfig()

	ctx := context.Background()
	start := time.Now()

	result, err := factory.EstimateFromConfigOptimized(ctx, testConfig)

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result but got nil")
	}

	// With 3 resources and 10ms delay each, concurrent processing should be faster than 30ms
	// Allow some margin for test execution overhead
	if elapsed > 25*time.Millisecond {
		t.Logf("Concurrent processing took %v, might not be truly concurrent", elapsed)
	}
}

func TestOptimizedFactory_EstimateFromConfigOptimized_Batching(t *testing.T) {
	mockClient := &MockAWSClient{products: createMockProducts()}

	config := &OptimizedFactoryConfig{
		MaxConcurrency: 2,
		EnableCaching:  false,
		EnableBatching: true,
		BatchSize:      2, // Small batch size for testing
	}

	factory := NewOptimizedFactory(mockClient, config)

	// Create config with more resources than batch size
	testConfig := &models.EstimationConfig{
		Version: "1.0",
		Resources: []models.ResourceSpec{
			{Type: "EC2", Name: "server-1", Region: "us-east-1", Properties: map[string]interface{}{"instanceType": "t3.micro"}},
			{Type: "EC2", Name: "server-2", Region: "us-east-1", Properties: map[string]interface{}{"instanceType": "t3.micro"}},
			{Type: "EC2", Name: "server-3", Region: "us-east-1", Properties: map[string]interface{}{"instanceType": "t3.micro"}},
			{Type: "EC2", Name: "server-4", Region: "us-east-1", Properties: map[string]interface{}{"instanceType": "t3.micro"}},
		},
	}

	ctx := context.Background()
	result, err := factory.EstimateFromConfigOptimized(ctx, testConfig)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result but got nil")
	}

	if len(result.ResourceCosts) != 4 {
		t.Errorf("Expected 4 resource costs, got %d", len(result.ResourceCosts))
	}
}

func TestOptimizedFactory_Caching(t *testing.T) {
	mockClient := &MockAWSClient{products: createMockProducts()}

	config := &OptimizedFactoryConfig{
		MaxConcurrency: 2,
		EnableCaching:  true,
		EnableBatching: false,
		CacheTTL:       1 * time.Minute,
		MaxCacheSize:   100,
	}

	factory := NewOptimizedFactory(mockClient, config)

	testConfig := &models.EstimationConfig{
		Version: "1.0",
		Resources: []models.ResourceSpec{
			{
				Type:   "EC2",
				Name:   "test-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
		},
	}

	ctx := context.Background()

	// First call should hit the AWS client
	_, err := factory.EstimateFromConfigOptimized(ctx, testConfig)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	firstCallCount := mockClient.callCount

	// Second call should hit the cache
	_, err = factory.EstimateFromConfigOptimized(ctx, testConfig)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mockClient.callCount != firstCallCount {
		t.Errorf("Expected cache hit (same call count), but got %d calls after second request", mockClient.callCount)
	}

	// Check cache stats
	stats := factory.GetCacheStats()
	if stats == nil {
		t.Fatal("Expected cache stats but got nil")
	}

	if stats.TotalEntries == 0 {
		t.Errorf("Expected cache entries but got 0")
	}
}

func TestOptimizedFactory_GetCacheStats_NoCaching(t *testing.T) {
	mockClient := &MockAWSClient{products: createMockProducts()}

	config := &OptimizedFactoryConfig{
		EnableCaching: false,
	}

	factory := NewOptimizedFactory(mockClient, config)

	stats := factory.GetCacheStats()
	if stats != nil {
		t.Errorf("Expected nil cache stats when caching disabled, got %+v", stats)
	}
}

func TestOptimizedFactory_ClearCache(t *testing.T) {
	mockClient := &MockAWSClient{products: createMockProducts()}

	config := &OptimizedFactoryConfig{
		EnableCaching: true,
		CacheTTL:      1 * time.Minute,
		MaxCacheSize:  100,
	}

	factory := NewOptimizedFactory(mockClient, config)

	testConfig := &models.EstimationConfig{
		Version: "1.0",
		Resources: []models.ResourceSpec{
			{
				Type:   "EC2",
				Name:   "test-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
		},
	}

	ctx := context.Background()

	// Make a call to populate cache
	_, err := factory.EstimateFromConfigOptimized(ctx, testConfig)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify cache has entries
	stats := factory.GetCacheStats()
	if stats.TotalEntries == 0 {
		t.Fatal("Expected cache entries but got 0")
	}

	// Clear cache
	factory.ClearCache()

	// Verify cache is empty
	stats = factory.GetCacheStats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 cache entries after clear, got %d", stats.TotalEntries)
	}
}

func TestOptimizedFactory_WarmupCache(t *testing.T) {
	mockClient := &MockAWSClient{products: createMockProducts()}

	config := &OptimizedFactoryConfig{
		EnableCaching: true,
		CacheTTL:      1 * time.Minute,
		MaxCacheSize:  100,
	}

	factory := NewOptimizedFactory(mockClient, config)

	warmupConfigs := []*models.EstimationConfig{
		{
			Version: "1.0",
			Resources: []models.ResourceSpec{
				{
					Type:   "EC2",
					Name:   "warmup-server",
					Region: "us-east-1",
					Properties: map[string]interface{}{
						"instanceType": "t3.micro",
					},
				},
			},
		},
	}

	ctx := context.Background()
	err := factory.WarmupCache(ctx, warmupConfigs)

	if err != nil {
		t.Errorf("Unexpected error during warmup: %v", err)
	}

	// Verify cache has entries
	stats := factory.GetCacheStats()
	if stats.TotalEntries == 0 {
		t.Errorf("Expected cache entries after warmup but got 0")
	}
}

func TestOptimizedFactory_ErrorHandling(t *testing.T) {
	mockClient := &MockAWSClient{
		err: errors.APIError("mock API error"),
	}

	factory := NewOptimizedFactory(mockClient, nil)

	testConfig := &models.EstimationConfig{
		Version: "1.0",
		Resources: []models.ResourceSpec{
			{
				Type:   "EC2",
				Name:   "test-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
		},
	}

	ctx := context.Background()
	_, err := factory.EstimateFromConfigOptimized(ctx, testConfig)

	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestOptimizedFactory_EmptyConfig(t *testing.T) {
	mockClient := &MockAWSClient{products: createMockProducts()}
	factory := NewOptimizedFactory(mockClient, nil)

	ctx := context.Background()

	// Test nil config
	_, err := factory.EstimateFromConfigOptimized(ctx, nil)
	if err == nil {
		t.Errorf("Expected error for nil config but got none")
	}

	// Test empty resources
	emptyConfig := &models.EstimationConfig{
		Version:   "1.0",
		Resources: []models.ResourceSpec{},
	}

	_, err = factory.EstimateFromConfigOptimized(ctx, emptyConfig)
	if err == nil {
		t.Errorf("Expected error for empty resources but got none")
	}
}
