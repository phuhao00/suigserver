package sui

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// MarketplaceConfig holds marketplace contract configuration
type MarketplaceConfig struct {
	PackageID    string `json:"package_id"`
	MarketplaceObjectID string `json:"marketplace_object_id"`
	Module       string `json:"module"`
}

// ListingInfo represents marketplace listing information
type ListingInfo struct {
	ID          string `json:"id"`
	Seller      string `json:"seller"`
	NFTID       string `json:"nft_id"`
	NFTType     string `json:"nft_type"`
	Price       uint64 `json:"price"`
	Currency    string `json:"currency"`
	CreatedAt   uint64 `json:"created_at"`
	ExpiresAt   *uint64 `json:"expires_at,omitempty"`
	Description string `json:"description"`
}

// MarketplaceInfo represents marketplace statistics
type MarketplaceInfo struct {
	FeePercentage uint64 `json:"fee_percentage"`
	ListingCount  uint64 `json:"listing_count"`
	TreasuryBalance uint64 `json:"treasury_balance"`
}

// PurchaseResult represents the result of a purchase transaction
type PurchaseResult struct {
	TransactionDigest string `json:"transaction_digest"`
	ListingID         string `json:"listing_id"`
	Buyer             string `json:"buyer"`
	Seller            string `json:"seller"`
	NFTID             string `json:"nft_id"`
	PricePaid         uint64 `json:"price_paid"`
	MarketplaceFee    uint64 `json:"marketplace_fee"`
	PurchaseTime      uint64 `json:"purchase_time"`
}

// MarketSuiService interacts with the Marketplace contract on the Sui blockchain
type MarketSuiService struct {
	client *SuiClient
	config MarketplaceConfig
}

// NewMarketSuiService creates a new MarketSuiService
func NewMarketSuiService(client *SuiClient, config MarketplaceConfig) *MarketSuiService {
	log.Println("Initializing Market Sui Service...")
	
	if config.Module == "" {
		config.Module = "marketplace"
	}
	
	return &MarketSuiService{
		client: client,
		config: config,
	}
}

// ListNFTForSale lists an NFT for sale on the marketplace
func (s *MarketSuiService) ListNFTForSale(sellerAddress, nftID, nftType string, price uint64, currency, description string, durationHours *uint64) (string, error) {
	log.Printf("Listing NFT %s for sale by %s at %d %s", nftID, sellerAddress, price, currency)

	// Prepare arguments for the Move function
	arguments := []interface{}{
		s.config.MarketplaceObjectID, // marketplace object
		nftID,                        // NFT object to list
		[]byte(nftType),             // NFT type as bytes
		strconv.FormatUint(price, 10), // price
		[]byte(currency),            // currency as bytes
		[]byte(description),         // description as bytes
	}

	// Add duration if specified
	if durationHours != nil {
		arguments = append(arguments, map[string]interface{}{
			"Some": strconv.FormatUint(*durationHours, 10),
		})
	} else {
		arguments = append(arguments, map[string]interface{}{
			"None": nil,
		})
	}

	// Execute the Move call
	result, err := s.client.MoveCall(
		sellerAddress,
		s.config.PackageID,
		s.config.Module,
		"list_nft",
		[]string{}, // Type arguments will be filled based on NFT type
		arguments,
		"",      // Gas object (let Sui auto-select)
		1000000, // Gas budget
	)

	if err != nil {
		return "", fmt.Errorf("failed to list NFT: %w", err)
	}

	// Extract listing ID from the transaction result
	// In a real implementation, you would parse the events or object changes
	// to get the actual listing ID created
	txDigest := result.Get("digest").String()
	listingID := fmt.Sprintf("listing_%s_%s", nftID, txDigest[:8])

	log.Printf("Successfully listed NFT %s with listing ID: %s", nftID, listingID)
	return listingID, nil
}

// PurchaseNFT buys an NFT from the marketplace
func (s *MarketSuiService) PurchaseNFT(buyerAddress, nftID, paymentCoinID string) (*PurchaseResult, error) {
	log.Printf("Player %s purchasing NFT %s", buyerAddress, nftID)

	// Prepare arguments for the Move function
	arguments := []interface{}{
		s.config.MarketplaceObjectID, // marketplace object
		nftID,                        // NFT ID to purchase
		paymentCoinID,               // Payment coin object
	}

	// Execute the Move call
	result, err := s.client.MoveCall(
		buyerAddress,
		s.config.PackageID,
		s.config.Module,
		"purchase_nft",
		[]string{}, // Type arguments will be filled based on NFT type
		arguments,
		"",      // Gas object
		2000000, // Gas budget (higher for purchase)
	)

	if err != nil {
		return nil, fmt.Errorf("failed to purchase NFT: %w", err)
	}

	// Parse the transaction result to extract purchase details
	// In a real implementation, you would parse the events to get the actual data
	txDigest := result.Get("digest").String()
	
	purchaseResult := &PurchaseResult{
		TransactionDigest: txDigest,
		ListingID:         fmt.Sprintf("listing_%s", nftID),
		Buyer:             buyerAddress,
		NFTID:             nftID,
		PurchaseTime:      uint64(result.Get("timestampMs").Int()),
	}

	log.Printf("Successfully purchased NFT %s, transaction: %s", nftID, txDigest)
	return purchaseResult, nil
}

// CancelListing cancels an NFT listing
func (s *MarketSuiService) CancelListing(sellerAddress, nftID string) error {
	log.Printf("Player %s canceling listing for NFT %s", sellerAddress, nftID)

	// Prepare arguments for the Move function
	arguments := []interface{}{
		s.config.MarketplaceObjectID, // marketplace object
		nftID,                        // NFT ID to cancel
	}

	// Execute the Move call
	result, err := s.client.MoveCall(
		sellerAddress,
		s.config.PackageID,
		s.config.Module,
		"cancel_listing",
		[]string{}, // Type arguments will be filled based on NFT type
		arguments,
		"",      // Gas object
		1000000, // Gas budget
	)

	if err != nil {
		return fmt.Errorf("failed to cancel listing: %w", err)
	}

	txDigest := result.Get("digest").String()
	log.Printf("Successfully canceled listing for NFT %s, transaction: %s", nftID, txDigest)
	return nil
}

// GetListings retrieves active listings from the marketplace
func (s *MarketSuiService) GetListings(limit int, cursor *string) ([]ListingInfo, *string, error) {
	log.Println("Fetching active market listings from Sui")

	// Query events for ListingCreated to get active listings
	query := map[string]interface{}{
		"MoveModule": map[string]interface{}{
			"package": s.config.PackageID,
			"module":  s.config.Module,
		},
	}

	limitPtr := uint64(limit)
	if limit <= 0 {
		limitPtr = 50 // default
	}

	result, err := s.client.QueryEvents(query, cursor, &limitPtr, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query marketplace events: %w", err)
	}

	var listings []ListingInfo
	var nextCursor *string

	// Parse events to extract listing information
	data := result.Get("data")
	if data.IsArray() {
		for _, event := range data.Array() {
			eventType := event.Get("type").String()
			if strings.Contains(eventType, "ListingCreated") {
				parsedJSON := event.Get("parsedJson")
				
				listing := ListingInfo{
					ID:          parsedJSON.Get("listing_id").String(),
					Seller:      parsedJSON.Get("seller").String(),
					NFTID:       parsedJSON.Get("nft_id").String(),
					NFTType:     parsedJSON.Get("nft_type").String(),
					Price:       parsedJSON.Get("price").Uint(),
					Currency:    parsedJSON.Get("currency").String(),
					CreatedAt:   parsedJSON.Get("timestamp").Uint(),
				}
				listings = append(listings, listing)
			}
		}
	}

	// Check if there's a next cursor
	if result.Get("nextCursor").Exists() {
		cursorStr := result.Get("nextCursor").String()
		nextCursor = &cursorStr
	}

	log.Printf("Retrieved %d active listings", len(listings))
	return listings, nextCursor, nil
}

// GetListingInfo retrieves detailed information about a specific listing
func (s *MarketSuiService) GetListingInfo(nftID string) (*ListingInfo, error) {
	log.Printf("Fetching listing info for NFT %s", nftID)

	// This would call the get_listing_info view function
	// For now, we'll simulate with a query
	
	// In a real implementation, you would call:
	// sui_devInspectTransactionBlock with a transaction that calls get_listing_info

	return &ListingInfo{
		ID:          fmt.Sprintf("listing_%s", nftID),
		NFTID:       nftID,
		NFTType:     "unknown",
		Price:       0,
		Currency:    "SUI",
	}, nil
}

// GetMarketplaceInfo retrieves marketplace statistics
func (s *MarketSuiService) GetMarketplaceInfo() (*MarketplaceInfo, error) {
	log.Println("Fetching marketplace information")

	// Get marketplace object details
	result, err := s.client.GetObject(s.config.MarketplaceObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get marketplace object: %w", err)
	}

	// Parse marketplace data
	content := result.Get("data.content")
	if !content.Exists() {
		return nil, fmt.Errorf("marketplace object not found or no content")
	}

	fields := content.Get("fields")
	
	info := &MarketplaceInfo{
		FeePercentage:   fields.Get("fee_percentage").Uint(),
		ListingCount:    fields.Get("listing_count").Uint(),
		TreasuryBalance: fields.Get("treasury.fields.balance").Uint(),
	}

	log.Printf("Marketplace info: fee=%d%%, listings=%d, treasury=%d", 
		info.FeePercentage, info.ListingCount, info.TreasuryBalance)
	
	return info, nil
}

// GetPlayerNFTs retrieves NFTs owned by a player that can be listed
func (s *MarketSuiService) GetPlayerNFTs(playerAddress string) ([]map[string]interface{}, error) {
	log.Printf("Fetching NFTs for player %s", playerAddress)

	// Get all objects owned by the player
	result, err := s.client.GetOwnedObjects(playerAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get owned objects: %w", err)
	}

	var nfts []map[string]interface{}
	
	data := result.Get("data")
	if data.IsArray() {
		for _, obj := range data.Array() {
			objData := obj.Get("data")
			objectType := objData.Get("type").String()
			
			// Filter for game NFTs (player, item, etc.)
			if strings.Contains(objectType, "player") || 
			   strings.Contains(objectType, "item") ||
			   strings.Contains(objectType, "guild") {
				
				nft := map[string]interface{}{
					"object_id": objData.Get("objectId").String(),
					"type":      objectType,
					"version":   objData.Get("version").Int(),
					"digest":    objData.Get("digest").String(),
				}
				
				// Add content if available
				if objData.Get("content").Exists() {
					nft["content"] = objData.Get("content").Value()
				}
				
				nfts = append(nfts, nft)
			}
		}
	}

	log.Printf("Found %d NFTs for player %s", len(nfts), playerAddress)
	return nfts, nil
}

// IsNFTListed checks if an NFT is currently listed on the marketplace
func (s *MarketSuiService) IsNFTListed(nftID string) (bool, error) {
	log.Printf("Checking if NFT %s is listed", nftID)

	// This would call the is_listed view function
	// For now, we'll return false as a placeholder
	
	return false, nil
}

// Helper function to parse Sui events into structured data
func (s *MarketSuiService) parseMarketplaceEvent(event gjson.Result) map[string]interface{} {
	parsedJSON := event.Get("parsedJson")
	eventType := event.Get("type").String()
	
	result := map[string]interface{}{
		"event_type":     eventType,
		"transaction_digest": event.Get("id.txDigest").String(),
		"timestamp":      event.Get("timestampMs").String(),
		"sender":         event.Get("sender").String(),
	}

	// Add parsed JSON data
	parsedJSON.ForEach(func(key, value gjson.Result) bool {
		result[key.String()] = value.Value()
		return true
	})

	return result
}

// GetMarketplaceEvents retrieves marketplace events (purchases, listings, cancellations)
func (s *MarketSuiService) GetMarketplaceEvents(eventType string, limit int, cursor *string) ([]map[string]interface{}, *string, error) {
	log.Printf("Fetching marketplace events of type: %s", eventType)

	query := map[string]interface{}{
		"MoveModule": map[string]interface{}{
			"package": s.config.PackageID,
			"module":  s.config.Module,
		},
	}

	limitPtr := uint64(limit)
	if limit <= 0 {
		limitPtr = 50
	}

	result, err := s.client.QueryEvents(query, cursor, &limitPtr, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query events: %w", err)
	}

	var events []map[string]interface{}
	var nextCursor *string

	data := result.Get("data")
	if data.IsArray() {
		for _, event := range data.Array() {
			evtType := event.Get("type").String()
			
			// Filter by event type if specified
			if eventType == "" || strings.Contains(evtType, eventType) {
				events = append(events, s.parseMarketplaceEvent(event))
			}
		}
	}

	if result.Get("nextCursor").Exists() {
		cursorStr := result.Get("nextCursor").String()
		nextCursor = &cursorStr
	}

	log.Printf("Retrieved %d marketplace events", len(events))
	return events, nextCursor, nil
}
