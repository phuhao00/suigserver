package sui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/block-vision/sui-go-sdk/models" // Added for TransactionBlockResponse
	"github.com/phuhao00/suigserver/server/configs"
	"github.com/phuhao00/suigserver/server/internal/utils" // For logging
)

// MarketplaceServiceManager manages the marketplace service, adding features like caching,
// rate limiting, and potentially orchestrating transaction signing and execution.
// For now, it primarily adapts the new MarketSuiService interface.
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

	utils.LogInfo("Marketplace Service Manager initialized successfully")
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

// PrepareListNFTForSale prepares a transaction to list an NFT for sale.
// This manager method handles rate limiting, validation, and then calls the underlying service.
// It returns the TransactionBlockResponse which contains TxBytes for signing.
// The actual signing and execution must be handled by the caller or a subsequent step.
// gasObjectID must be for a gas coin owned by sellerAddress.
func (m *MarketplaceServiceManager) PrepareListNFTForSale(
	sellerAddress string,
	nftID string,
	nftType string, // Fully qualified type of the NFT, e.g., "0xPACKAGE::module::NftName"
	price uint64,
	currencyCoinType string, // Fully qualified type of the coin for pricing, e.g., "0x2::sui::SUI"
	description string,
	durationHours *uint64,
	gasObjectID string, // Specific gas object ID for this transaction
) (models.TxnMetaData, error) {
	if !m.checkRateLimit(sellerAddress) {
		return models.TxnMetaData{}, fmt.Errorf("rate limit exceeded for user %s", sellerAddress)
	}

	if durationHours != nil && *durationHours > m.config.MaxListingDuration {
		return models.TxnMetaData{}, fmt.Errorf("listing duration exceeds maximum allowed (%d hours)", m.config.MaxListingDuration)
	}
	if gasObjectID == "" {
		return models.TxnMetaData{}, fmt.Errorf("gasObjectID is required for PrepareListNFTForSale")
	}

	// Call marketplace service - note the new signature
	txBlockResp, err := m.marketService.ListNFTForSale(
		sellerAddress, nftID, nftType, price, currencyCoinType, description, durationHours,
		gasObjectID, m.config.DefaultGasBudget, // Using default gas budget from config
	)
	if err != nil {
		return models.TxnMetaData{}, err // Error already logged by service
	}

	// Invalidate relevant caches if the operation implies a state change that affects cached data.
	// For preparing a transaction, this might be premature. Caches should be invalidated *after*
	// successful execution of the transaction.
	// For now, we'll assume invalidation happens elsewhere post-execution.
	// m.invalidateListingsCache() // Potentially move this to an "after execution success" handler

	return txBlockResp, nil
}

// PreparePurchaseNFT prepares a transaction to purchase an NFT.
// Returns TransactionBlockResponse containing TxBytes for signing by the buyer.
// buyerGasObjectID must be for a gas coin owned by buyerAddress.
// paymentCoinID is the specific Coin object used for payment.
func (m *MarketplaceServiceManager) PreparePurchaseNFT(
	buyerAddress string,
	listingObjectID string,
	paymentCoinID string,
	nftType string, // Fully qualified type of the NFT being purchased
	coinType string, // Fully qualified type of the coin being used for payment
	buyerGasObjectID string,
) (models.TxnMetaData, error) {
	if !m.checkRateLimit(buyerAddress) {
		return models.TxnMetaData{}, fmt.Errorf("rate limit exceeded for user %s", buyerAddress)
	}
	if buyerGasObjectID == "" || paymentCoinID == "" || listingObjectID == "" {
		return models.TxnMetaData{}, fmt.Errorf("buyerGasObjectID, paymentCoinID, and listingObjectID are required")
	}

	txBlockResp, err := m.marketService.PurchaseNFT(
		buyerAddress, listingObjectID, paymentCoinID,
		nftType, coinType, // Pass NFT and Coin types for generics
		buyerGasObjectID, m.config.DefaultGasBudget,
	)
	if err != nil {
		return models.TxnMetaData{}, err
	}

	// Invalidation would happen after successful execution.
	// m.invalidateListingsCache()
	// m.invalidateNFTCache(nftID) // Need the actual NFT ID here, which might be part of listingObjectID or fetched.

	return txBlockResp, nil
}

// PrepareCancelListing prepares a transaction to cancel an NFT listing.
// Returns TransactionBlockResponse containing TxBytes for signing by the seller.
// sellerGasObjectID must be for a gas coin owned by sellerAddress.
func (m *MarketplaceServiceManager) PrepareCancelListing(
	sellerAddress string,
	listingObjectID string,
	nftType string, // The type of NFT that was listed
	coinType string, // The type of Coin that was expected
	sellerGasObjectID string,
) (models.TxnMetaData, error) {
	if !m.checkRateLimit(sellerAddress) {
		return models.TxnMetaData{}, fmt.Errorf("rate limit exceeded for user %s", sellerAddress)
	}
	if sellerGasObjectID == "" || listingObjectID == "" {
		return models.TxnMetaData{}, fmt.Errorf("sellerGasObjectID and listingObjectID are required")
	}

	txBlockResp, err := m.marketService.CancelListing(
		sellerAddress, listingObjectID,
		nftType, coinType, // Pass NFT and Coin types for generics
		sellerGasObjectID, m.config.DefaultGasBudget,
	)
	if err != nil {
		return models.TxnMetaData{}, err
	}

	// Invalidation would happen after successful execution.
	// m.invalidateListingsCache()
	// m.invalidateNFTCache(nftID) // Need actual NFT ID

	return txBlockResp, nil
}

// GetListings retrieves listings with caching. The eventType is the fully qualified
// type of the event that creates listings (e.g., "0xPKG::market::ListingCreated").
func (m *MarketplaceServiceManager) GetListings(eventType string, limit int, cursor *string) ([]ListingInfo, *string, error) {
	// Note: Caching key might need to include eventType if it can vary for "listings"
	cacheKey := fmt.Sprintf("listings_%s_%d_%v", eventType, limit, cursor)

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
	// Ensure marketService.GetListings is called with eventType
	listings, nextCursor, err := m.marketService.GetListings(eventType, limit, cursor)
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
	// The marketService.GetPlayerNFTs method takes an optional specificNftType filter.
	// This manager method currently doesn't expose that, so passing nil.
	nfts, err := m.marketService.GetPlayerNFTs(playerAddress, nil)
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

	// Remove all listings cache entries (could be more specific if keys are well-defined)
	// Example: iterate and check prefix "listings_" and "marketplace_info"
	for key := range m.cache {
		// A more robust way would be to have predefined cache key prefixes
		if strings.HasPrefix(key, "listings_") || key == "marketplace_info" {
			delete(m.cache, key)
			delete(m.cacheExpiry, key)
			utils.LogDebugf("MarketplaceManager: Invalidated cache key: %s", key)
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

	cacheKeyListingInfo := fmt.Sprintf("listing_info_%s", nftID) // Assuming nftID here means listingObjectID for GetListingInfo
	delete(m.cache, cacheKeyListingInfo)
	delete(m.cacheExpiry, cacheKeyListingInfo)
	utils.LogDebugf("MarketplaceManager: Invalidated cache key: %s", cacheKeyListingInfo)

	// If nftID is also used for other specific NFT details (not just listings)
	// cacheKeyNftDetails := fmt.Sprintf("nft_details_%s", nftID)
	// delete(m.cache, cacheKeyNftDetails)
	// delete(m.cacheExpiry, cacheKeyNftDetails)
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
	utils.LogInfo("Shutting down Marketplace Service Manager...")

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
