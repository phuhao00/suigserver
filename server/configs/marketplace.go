package configs

import (
	"encoding/json"
	"fmt"
	"os"
)

// MarketplaceConfig holds configuration for the marketplace service
type MarketplaceConfig struct {
	// Sui network configuration
	SuiNodeURL string `json:"sui_node_url"`
	
	// Marketplace contract configuration
	PackageID           string `json:"package_id"`
	MarketplaceObjectID string `json:"marketplace_object_id"`
	AdminCapID          string `json:"admin_cap_id"`
	Module              string `json:"module"`
	
	// Service configuration
	DefaultGasBudget uint64 `json:"default_gas_budget"`
	MaxListingDuration uint64 `json:"max_listing_duration_hours"`
	
	// Cache settings
	EnableCaching     bool   `json:"enable_caching"`
	CacheExpiration   int    `json:"cache_expiration_seconds"`
	
	// Rate limiting
	RateLimitEnabled  bool   `json:"rate_limit_enabled"`
	RateLimitPerMin   int    `json:"rate_limit_per_minute"`
}

// DefaultMarketplaceConfig returns default configuration
func DefaultMarketplaceConfig() *MarketplaceConfig {
	return &MarketplaceConfig{
		SuiNodeURL:           "https://fullnode.testnet.sui.io:443",
		Module:               "marketplace",
		DefaultGasBudget:     1000000,
		MaxListingDuration:   168, // 7 days
		EnableCaching:        true,
		CacheExpiration:      300, // 5 minutes
		RateLimitEnabled:     true,
		RateLimitPerMin:      100,
	}
}

// LoadMarketplaceConfig loads configuration from file
func LoadMarketplaceConfig(configPath string) (*MarketplaceConfig, error) {
	config := DefaultMarketplaceConfig()
	
	if configPath == "" {
		return config, nil
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return default config
			return config, nil
		}
		return nil, err
	}
	
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	
	return config, nil
}

// SaveMarketplaceConfig saves configuration to file
func (c *MarketplaceConfig) SaveToFile(configPath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

// Validate checks if the configuration is valid
func (c *MarketplaceConfig) Validate() error {
	if c.SuiNodeURL == "" {
		return fmt.Errorf("sui_node_url is required")
	}
	
	if c.PackageID == "" {
		return fmt.Errorf("package_id is required")
	}
	
	if c.MarketplaceObjectID == "" {
		return fmt.Errorf("marketplace_object_id is required")
	}
	
	if c.DefaultGasBudget == 0 {
		return fmt.Errorf("default_gas_budget must be greater than 0")
	}
	
	return nil
}
