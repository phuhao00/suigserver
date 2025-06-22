package sui

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/phuhao00/suigserver/server/configs"
)

// MarketplaceServiceManager manages the marketplace service with advanced features
type MarketplaceServiceManager struct {
	marketService *MarketSuiService
	client        *SuiClient
	config        *configs.MarketplaceConfig

	// Caching
	cache       map[string]interface{}
	cacheMutex  sync.RWMutex
	cacheExpiry map[string]time.Time

	// Rate limiting
	rateLimiter map[string][]time.Time
	rateMutex   sync.RWMutex
}

// NewMarketplaceServiceManager creates a new marketplace service manager
func NewMarketplaceServiceManager(config *configs.MarketplaceConfig) (*MarketplaceServiceManager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create Sui client
	client := NewSuiClient(config.SuiNodeURL)

	// Create marketplace service config
	marketConfig := MarketplaceConfig{
		PackageID:           config.PackageID,
		MarketplaceObjectID: config.MarketplaceObjectID,
		Module:              config.Module,
	}

	// Create marketplace service
	marketService := NewMarketSuiService(client, marketConfig)

	manager := &MarketplaceServiceManager{
		marketService: marketService,
		client:        client,
		config:        config,
		cache:         make(map[string]interface{}),
		cacheExpiry:   make(map[string]time.Time),
		rateLimiter:   make(map[string][]time.Time),
	}

	// Start cache cleanup routine
	if config.EnableCaching {
		go manager.cacheCleanupRoutine()
	}

	log.Println("Marketplace Service Manager initialized successfully")
	return manager, nil
}

// cacheCleanupRoutine periodically cleans expired cache entries
func (m *MarketplaceServiceManager) cacheCleanupRoutine() {
	ticker := time.NewTicker(time.Minute * 5) // Clean every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		m.cleanExpiredCache()
	}
}

// cleanExpiredCache removes expired cache entries
func (m *MarketplaceServiceManager) cleanExpiredCache() {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	now := time.Now()
	for key, expiry := range m.cacheExpiry {
		if now.After(expiry) {
			delete(m.cache, key)
			delete(m.cacheExpiry, key)
		}
	}
}

// getFromCache retrieves data from cache
func (m *MarketplaceServiceManager) getFromCache(key string) (interface{}, bool) {
	if !m.config.EnableCaching {
		return nil, false
	}

	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	// Check if key exists and hasn't expired
	expiry, exists := m.cacheExpiry[key]
	if !exists || time.Now().After(expiry) {
		return nil, false
	}

	value, exists := m.cache[key]
	return value, exists
}

// setCache stores data in cache
func (m *MarketplaceServiceManager) setCache(key string, value interface{}) {
	if !m.config.EnableCaching {
		return
	}

	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	m.cache[key] = value
	m.cacheExpiry[key] = time.Now().Add(time.Second * time.Duration(m.config.CacheExpiration))
}

// checkRateLimit checks if the operation is rate limited for a user
func (m *MarketplaceServiceManager) checkRateLimit(userID string) bool {
	if !m.config.RateLimitEnabled {
		return true // Allow if rate limiting is disabled
	}

	m.rateMutex.Lock()
	defer m.rateMutex.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-time.Minute)

	// Get user's recent requests
	requests, exists := m.rateLimiter[userID]
	if !exists {
		requests = []time.Time{}
	}

	// Filter out requests older than 1 minute
	var recentRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(oneMinuteAgo) {
			recentRequests = append(recentRequests, reqTime)
		}
	}

	// Check if under rate limit
	if len(recentRequests) >= m.config.RateLimitPerMin {
		return false
	}

	// Add current request
	recentRequests = append(recentRequests, now)
	m.rateLimiter[userID] = recentRequests

	return true
}

// ListNFTForSale lists an NFT for sale with rate limiting and caching
func (m *MarketplaceServiceManager) ListNFTForSale(sellerAddress, nftID, nftType string, price uint64, currency, description string, durationHours *uint64) (string, error) {
	// Check rate limit
	if !m.checkRateLimit(sellerAddress) {
		return "", fmt.Errorf("rate limit exceeded for user %s", sellerAddress)
	}

	// Validate duration
	if durationHours != nil && *durationHours > m.config.MaxListingDuration {
		return "", fmt.Errorf("listing duration exceeds maximum allowed (%d hours)", m.config.MaxListingDuration)
	}

	// Call marketplace service
	listingID, err := m.marketService.ListNFTForSale(sellerAddress, nftID, nftType, price, currency, description, durationHours)
	if err != nil {
		return "", err
	}

	// Invalidate cache for listings
	m.invalidateListingsCache()

	return listingID, nil
}

// PurchaseNFT purchases an NFT with rate limiting
func (m *MarketplaceServiceManager) PurchaseNFT(buyerAddress, nftID, paymentCoinID string) (*PurchaseResult, error) {
	// Check rate limit
	if !m.checkRateLimit(buyerAddress) {
		return nil, fmt.Errorf("rate limit exceeded for user %s", buyerAddress)
	}

	// Call marketplace service
	result, err := m.marketService.PurchaseNFT(buyerAddress, nftID, paymentCoinID)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	m.invalidateListingsCache()
	m.invalidateNFTCache(nftID)

	return result, nil
}

// CancelListing cancels a listing with rate limiting
func (m *MarketplaceServiceManager) CancelListing(sellerAddress, nftID string) error {
	// Check rate limit
	if !m.checkRateLimit(sellerAddress) {
		return fmt.Errorf("rate limit exceeded for user %s", sellerAddress)
	}

	// Call marketplace service
	err := m.marketService.CancelListing(sellerAddress, nftID)
	if err != nil {
		return err
	}

	// Invalidate cache
	m.invalidateListingsCache()
	m.invalidateNFTCache(nftID)

	return nil
}

// GetListings retrieves listings with caching
func (m *MarketplaceServiceManager) GetListings(limit int, cursor *string) ([]ListingInfo, *string, error) {
	cacheKey := fmt.Sprintf("listings_%d_%v", limit, cursor)

	// Try cache first
	if cached, found := m.getFromCache(cacheKey); found {
		if result, ok := cached.(struct {
			Listings   []ListingInfo
			NextCursor *string
		}); ok {
			return result.Listings, result.NextCursor, nil
		}
	}

	// Fetch from blockchain
	listings, nextCursor, err := m.marketService.GetListings(limit, cursor)
	if err != nil {
		return nil, nil, err
	}

	// Cache the result
	m.setCache(cacheKey, struct {
		Listings   []ListingInfo
		NextCursor *string
	}{
		Listings:   listings,
		NextCursor: nextCursor,
	})

	return listings, nextCursor, nil
}

// GetMarketplaceInfo retrieves marketplace info with caching
func (m *MarketplaceServiceManager) GetMarketplaceInfo() (*MarketplaceInfo, error) {
	cacheKey := "marketplace_info"

	// Try cache first
	if cached, found := m.getFromCache(cacheKey); found {
		if info, ok := cached.(*MarketplaceInfo); ok {
			return info, nil
		}
	}

	// Fetch from blockchain
	info, err := m.marketService.GetMarketplaceInfo()
	if err != nil {
		return nil, err
	}

	// Cache the result
	m.setCache(cacheKey, info)

	return info, nil
}

// GetPlayerNFTs retrieves player NFTs with caching
func (m *MarketplaceServiceManager) GetPlayerNFTs(playerAddress string) ([]map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("player_nfts_%s", playerAddress)

	// Try cache first
	if cached, found := m.getFromCache(cacheKey); found {
		if nfts, ok := cached.([]map[string]interface{}); ok {
			return nfts, nil
		}
	}

	// Fetch from blockchain
	nfts, err := m.marketService.GetPlayerNFTs(playerAddress)
	if err != nil {
		return nil, err
	}

	// Cache the result (shorter expiration for player data)
	m.cacheMutex.Lock()
	m.cache[cacheKey] = nfts
	m.cacheExpiry[cacheKey] = time.Now().Add(time.Second * 60) // 1 minute for player data
	m.cacheMutex.Unlock()

	return nfts, nil
}

// GetMarketplaceEvents retrieves marketplace events
func (m *MarketplaceServiceManager) GetMarketplaceEvents(eventType string, limit int, cursor *string) ([]map[string]interface{}, *string, error) {
	return m.marketService.GetMarketplaceEvents(eventType, limit, cursor)
}

// invalidateListingsCache invalidates cached listings
func (m *MarketplaceServiceManager) invalidateListingsCache() {
	if !m.config.EnableCaching {
		return
	}

	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	// Remove all listings cache entries
	for key := range m.cache {
		if key == "marketplace_info" ||
			key[:9] == "listings_" {
			delete(m.cache, key)
			delete(m.cacheExpiry, key)
		}
	}
}

// invalidateNFTCache invalidates cache for a specific NFT
func (m *MarketplaceServiceManager) invalidateNFTCache(nftID string) {
	if !m.config.EnableCaching {
		return
	}

	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	cacheKey := fmt.Sprintf("nft_info_%s", nftID)
	delete(m.cache, cacheKey)
	delete(m.cacheExpiry, cacheKey)
}

// GetStats returns service statistics
func (m *MarketplaceServiceManager) GetStats() map[string]interface{} {
	m.cacheMutex.RLock()
	cacheSize := len(m.cache)
	m.cacheMutex.RUnlock()

	m.rateMutex.RLock()
	rateLimitEntries := len(m.rateLimiter)
	m.rateMutex.RUnlock()

	return map[string]interface{}{
		"cache_enabled":         m.config.EnableCaching,
		"cache_size":            cacheSize,
		"rate_limit_enabled":    m.config.RateLimitEnabled,
		"rate_limit_entries":    rateLimitEntries,
		"sui_node_url":          m.config.SuiNodeURL,
		"package_id":            m.config.PackageID,
		"marketplace_object_id": m.config.MarketplaceObjectID,
	}
}

// Close gracefully shuts down the service manager
func (m *MarketplaceServiceManager) Close() error {
	log.Println("Shutting down Marketplace Service Manager...")

	// Clear caches
	m.cacheMutex.Lock()
	m.cache = make(map[string]interface{})
	m.cacheExpiry = make(map[string]time.Time)
	m.cacheMutex.Unlock()

	// Clear rate limiter
	m.rateMutex.Lock()
	m.rateLimiter = make(map[string][]time.Time)
	m.rateMutex.Unlock()

	return nil
}
