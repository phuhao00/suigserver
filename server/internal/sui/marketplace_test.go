package sui

import (
	"fmt"
	"testing"
	"time"

	"github.com/phuhao00/suigserver/server/configs"
)

func TestMarketplaceServiceManager(t *testing.T) {
	// Create test configuration
	config := &configs.MarketplaceConfig{
		SuiNodeURL:          "https://fullnode.testnet.sui.io:443",
		PackageID:           "0x1234567890abcdef",
		MarketplaceObjectID: "0xabcdef1234567890",
		Module:              "marketplace",
		DefaultGasBudget:    1000000,
		MaxListingDuration:  168,
		EnableCaching:       true,
		CacheExpiration:     300,
		RateLimitEnabled:    true,
		RateLimitPerMin:     10,
	}

	// Note: This test will fail with real Sui calls since we're using test data
	// In a real test environment, you would use a local Sui network or mock the client
	manager, err := NewMarketplaceServiceManager(config)
	if err != nil {
		t.Fatalf("Failed to create marketplace service manager: %v", err)
	}
	defer manager.Close()

	t.Run("TestRateLimit", func(t *testing.T) {
		userID := "test_user"

		// Should allow first request
		if !manager.checkRateLimit(userID) {
			t.Error("Rate limit should allow first request")
		}

		// Simulate hitting rate limit
		for i := 0; i < config.RateLimitPerMin; i++ {
			manager.checkRateLimit(userID)
		}

		// Should block next request
		if manager.checkRateLimit(userID) {
			t.Error("Rate limit should block request after limit exceeded")
		}
	})

	t.Run("TestCache", func(t *testing.T) {
		key := "test_key"
		value := "test_value"

		// Set cache
		manager.setCache(key, value)

		// Get from cache
		cached, found := manager.getFromCache(key)
		if !found {
			t.Error("Should find cached value")
		}

		if cached != value {
			t.Errorf("Expected %v, got %v", value, cached)
		}
	})

	t.Run("TestCacheExpiration", func(t *testing.T) {
		// Set short expiration for testing
		oldExpiration := manager.config.CacheExpiration
		manager.config.CacheExpiration = 1 // 1 second
		defer func() {
			manager.config.CacheExpiration = oldExpiration
		}()

		key := "test_expiry"
		value := "test_value"

		manager.setCache(key, value)

		// Should be available immediately
		_, found := manager.getFromCache(key)
		if !found {
			t.Error("Should find cached value immediately")
		}

		// Wait for expiration
		time.Sleep(2 * time.Second)

		// Should be expired
		_, found = manager.getFromCache(key)
		if found {
			t.Error("Should not find expired cached value")
		}
	})

	t.Run("TestGetStats", func(t *testing.T) {
		stats := manager.GetStats()

		// Check required fields
		if stats["cache_enabled"] != config.EnableCaching {
			t.Error("Stats should include cache_enabled")
		}

		if stats["rate_limit_enabled"] != config.RateLimitEnabled {
			t.Error("Stats should include rate_limit_enabled")
		}

		if stats["package_id"] != config.PackageID {
			t.Error("Stats should include package_id")
		}
	})
}

func TestMarketplaceConfig(t *testing.T) {
	t.Run("TestDefaultConfig", func(t *testing.T) {
		config := configs.DefaultMarketplaceConfig()

		if config.SuiNodeURL == "" {
			t.Error("Default config should have SuiNodeURL")
		}

		if config.DefaultGasBudget == 0 {
			t.Error("Default config should have DefaultGasBudget")
		}

		if config.Module != "marketplace" {
			t.Error("Default config should have correct module name")
		}
	})

	t.Run("TestValidation", func(t *testing.T) {
		// Valid config
		validConfig := &configs.MarketplaceConfig{
			SuiNodeURL:          "https://test.sui.io",
			PackageID:           "0x123",
			MarketplaceObjectID: "0x456",
			DefaultGasBudget:    1000000,
		}

		if err := validConfig.Validate(); err != nil {
			t.Errorf("Valid config should pass validation: %v", err)
		}

		// Invalid config - missing required fields
		invalidConfig := &configs.MarketplaceConfig{}

		if err := invalidConfig.Validate(); err == nil {
			t.Error("Invalid config should fail validation")
		}
	})
}

// Benchmark tests
func BenchmarkRateLimit(b *testing.B) {
	config := configs.DefaultMarketplaceConfig()
	config.RateLimitEnabled = true
	config.RateLimitPerMin = 1000

	manager, _ := NewMarketplaceServiceManager(config)
	defer manager.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		manager.checkRateLimit("benchmark_user")
	}
}

func BenchmarkCache(b *testing.B) {
	config := configs.DefaultMarketplaceConfig()
	config.EnableCaching = true

	manager, _ := NewMarketplaceServiceManager(config)
	defer manager.Close()

	// Populate cache
	for i := 0; i < 1000; i++ {
		manager.setCache(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		manager.getFromCache(fmt.Sprintf("key_%d", i%1000))
	}
}
