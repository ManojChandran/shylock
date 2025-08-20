package performance

import (
	"context"
	"runtime"
	"sync"
	"time"

	"shylock/internal/cache"
	"shylock/internal/errors"
	"shylock/internal/estimators"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// OptimizedFactory provides performance-enhanced cost estimation
type OptimizedFactory struct {
	*estimators.Factory
	cache          *cache.PricingCache
	maxConcurrency int
	enableCaching  bool
	enableBatching bool
	batchSize      int
}

// OptimizedFactoryConfig holds configuration for the optimized factory
type OptimizedFactoryConfig struct {
	MaxConcurrency int
	EnableCaching  bool
	EnableBatching bool
	BatchSize      int
	CacheTTL       time.Duration
	MaxCacheSize   int
}

// DefaultOptimizedConfig returns default configuration for optimized factory
func DefaultOptimizedConfig() *OptimizedFactoryConfig {
	return &OptimizedFactoryConfig{
		MaxConcurrency: runtime.NumCPU() * 2, // 2x CPU cores
		EnableCaching:  true,
		EnableBatching: true,
		BatchSize:      10,
		CacheTTL:       15 * time.Minute,
		MaxCacheSize:   1000,
	}
}

// NewOptimizedFactory creates a new performance-optimized factory
func NewOptimizedFactory(awsClient interfaces.AWSPricingClient, config *OptimizedFactoryConfig) *OptimizedFactory {
	if config == nil {
		config = DefaultOptimizedConfig()
	}

	var clientToUse interfaces.AWSPricingClient = awsClient
	var pricingCache *cache.PricingCache

	// Wrap client with caching if enabled
	if config.EnableCaching {
		pricingCache = cache.NewPricingCache(config.CacheTTL, config.MaxCacheSize)
		clientToUse = cache.NewCachedAWSClient(awsClient, pricingCache)
	}

	// Create base factory with potentially cached client
	baseFactory := estimators.NewFactory(clientToUse)

	return &OptimizedFactory{
		Factory:        baseFactory,
		cache:          pricingCache,
		maxConcurrency: config.MaxConcurrency,
		enableCaching:  config.EnableCaching,
		enableBatching: config.EnableBatching,
		batchSize:      config.BatchSize,
	}
}

// EstimateFromConfigOptimized performs optimized cost estimation
func (f *OptimizedFactory) EstimateFromConfigOptimized(ctx context.Context, config *models.EstimationConfig) (*models.EstimationResult, error) {
	if config == nil {
		return nil, errors.ValidationError("configuration cannot be nil").
			WithSuggestion("Provide a valid estimation configuration")
	}

	if len(config.Resources) == 0 {
		return nil, errors.ValidationError("no resources to estimate").
			WithSuggestion("Add at least one resource to the configuration")
	}

	// Use concurrent processing for multiple resources
	if len(config.Resources) > 1 && f.maxConcurrency > 1 {
		return f.estimateConcurrently(ctx, config)
	}

	// Fall back to sequential processing for single resource or when concurrency is disabled
	return f.Factory.EstimateFromConfig(ctx, config)
}

// estimateConcurrently processes resources concurrently with batching
func (f *OptimizedFactory) estimateConcurrently(ctx context.Context, config *models.EstimationConfig) (*models.EstimationResult, error) {
	result := &models.EstimationResult{
		Currency:    "USD",
		GeneratedAt: time.Now(),
	}

	// Apply configuration options
	if config.Options.Currency != "" {
		result.Currency = config.Options.Currency
	}

	// Process resources in batches if batching is enabled
	if f.enableBatching && len(config.Resources) > f.batchSize {
		return f.estimateInBatches(ctx, config, result)
	}

	// Process all resources concurrently
	estimates, errs := f.processResourcesConcurrently(ctx, config.Resources)

	// Collect successful estimates and handle errors
	var estimationErrors []error
	var totalHourly, totalDaily, totalMonthly float64

	for i, estimate := range estimates {
		if errs[i] != nil {
			estimationErrors = append(estimationErrors, errs[i])
			continue
		}

		if estimate != nil {
			result.ResourceCosts = append(result.ResourceCosts, *estimate)
			totalHourly += estimate.HourlyCost
			totalDaily += estimate.DailyCost
			totalMonthly += estimate.MonthlyCost
		}
	}

	// Check if we have any successful estimates
	if len(result.ResourceCosts) == 0 {
		if len(estimationErrors) > 0 {
			return nil, estimationErrors[0]
		}
		return nil, errors.ValidationError("no resources could be estimated").
			WithSuggestion("Check that all resources have valid configurations")
	}

	// Set totals
	result.TotalHourlyCost = totalHourly
	result.TotalDailyCost = totalDaily
	result.TotalMonthlyCost = totalMonthly

	return result, nil
}

// estimateInBatches processes resources in batches to control memory usage
func (f *OptimizedFactory) estimateInBatches(ctx context.Context, config *models.EstimationConfig, result *models.EstimationResult) (*models.EstimationResult, error) {
	var allEstimates []models.CostEstimate
	var totalHourly, totalDaily, totalMonthly float64
	var estimationErrors []error

	// Process resources in batches
	for i := 0; i < len(config.Resources); i += f.batchSize {
		end := i + f.batchSize
		if end > len(config.Resources) {
			end = len(config.Resources)
		}

		batch := config.Resources[i:end]
		estimates, errs := f.processResourcesConcurrently(ctx, batch)

		// Collect results from this batch
		for j, estimate := range estimates {
			if errs[j] != nil {
				estimationErrors = append(estimationErrors, errs[j])
				continue
			}

			if estimate != nil {
				allEstimates = append(allEstimates, *estimate)
				totalHourly += estimate.HourlyCost
				totalDaily += estimate.DailyCost
				totalMonthly += estimate.MonthlyCost
			}
		}
	}

	// Check if we have any successful estimates
	if len(allEstimates) == 0 {
		if len(estimationErrors) > 0 {
			return nil, estimationErrors[0]
		}
		return nil, errors.ValidationError("no resources could be estimated").
			WithSuggestion("Check that all resources have valid configurations")
	}

	// Set results
	result.ResourceCosts = allEstimates
	result.TotalHourlyCost = totalHourly
	result.TotalDailyCost = totalDaily
	result.TotalMonthlyCost = totalMonthly

	return result, nil
}

// processResourcesConcurrently processes a slice of resources concurrently
func (f *OptimizedFactory) processResourcesConcurrently(ctx context.Context, resources []models.ResourceSpec) ([]*models.CostEstimate, []error) {
	if len(resources) == 0 {
		return nil, nil
	}

	// Create channels for results
	type result struct {
		index    int
		estimate *models.CostEstimate
		err      error
	}

	resultChan := make(chan result, len(resources))
	semaphore := make(chan struct{}, f.maxConcurrency)

	// Start goroutines with concurrency control
	var wg sync.WaitGroup
	for i, resource := range resources {
		wg.Add(1)
		go func(index int, res models.ResourceSpec) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			estimate, err := f.Factory.EstimateResource(ctx, res)
			resultChan <- result{
				index:    index,
				estimate: estimate,
				err:      err,
			}
		}(i, resource)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	estimates := make([]*models.CostEstimate, len(resources))
	errs := make([]error, len(resources))

	for res := range resultChan {
		estimates[res.index] = res.estimate
		errs[res.index] = res.err
	}

	return estimates, errs
}

// GetCacheStats returns cache statistics if caching is enabled
func (f *OptimizedFactory) GetCacheStats() *cache.CacheStats {
	if !f.enableCaching || f.cache == nil {
		return nil
	}

	stats := f.cache.Stats()
	return &stats
}

// ClearCache clears the pricing cache if caching is enabled
func (f *OptimizedFactory) ClearCache() {
	if f.enableCaching && f.cache != nil {
		f.cache.Clear()
	}
}

// CleanupExpiredCache removes expired entries from the cache
func (f *OptimizedFactory) CleanupExpiredCache() int {
	if !f.enableCaching || f.cache == nil {
		return 0
	}

	return f.cache.CleanupExpired()
}

// GetPerformanceConfig returns the current performance configuration
func (f *OptimizedFactory) GetPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		MaxConcurrency: f.maxConcurrency,
		EnableCaching:  f.enableCaching,
		EnableBatching: f.enableBatching,
		BatchSize:      f.batchSize,
	}
}

// PerformanceConfig represents the current performance settings
type PerformanceConfig struct {
	MaxConcurrency int
	EnableCaching  bool
	EnableBatching bool
	BatchSize      int
}

// EstimateWithTimeout performs estimation with a timeout
func (f *OptimizedFactory) EstimateWithTimeout(config *models.EstimationConfig, timeout time.Duration) (*models.EstimationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return f.EstimateFromConfigOptimized(ctx, config)
}

// WarmupCache pre-loads common pricing data into the cache
func (f *OptimizedFactory) WarmupCache(ctx context.Context, commonConfigs []*models.EstimationConfig) error {
	if !f.enableCaching {
		return nil
	}

	// Process common configurations to populate cache
	for _, config := range commonConfigs {
		_, _ = f.EstimateFromConfigOptimized(ctx, config) // Ignore errors during warmup
	}

	return nil
}
