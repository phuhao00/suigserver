package sui

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	// "github.com/tidwall/gjson" // Will be removed
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/phuhao00/suigserver/server/internal/utils"
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
func NewMarketSuiService(suiClient *SuiClient, config MarketplaceConfig) *MarketSuiService {
	utils.LogInfo("Initializing Market Sui Service...") // Changed to utils.LogInfo
	if suiClient == nil {
		utils.LogPanic("MarketSuiService: SuiClient cannot be nil") // Changed to utils.LogPanic
	}
	if config.PackageID == "" {
		utils.LogPanic("MarketSuiService: PackageID must be provided in config")
	}
	if config.MarketplaceObjectID == "" {
		utils.LogPanic("MarketSuiService: MarketplaceObjectID must be provided in config")
	}
	if config.Module == "" {
		utils.LogWarn("MarketSuiService: Module not specified in config, defaulting to 'marketplace'")
		config.Module = "marketplace"
	}

	return &MarketSuiService{
		client: suiClient, // Ensure this uses the passed suiClient
		config: config,
	}
}

// ListNFTForSale prepares a transaction to list an NFT for sale on the marketplace.
// It returns the transaction bytes that need to be signed and executed.
// A specific gas object ID owned by the sellerAddress must be provided for the transaction.
func (s *MarketSuiService) ListNFTForSale(
	sellerAddress string,
	nftID string,
	nftType string, // Consider if this is still needed if NFT type is part of NFT object, or used for generic type T
	price uint64,
	currency string, // e.g., "0x2::sui::SUI" or custom coin type
	description string,
	durationHours *uint64,
	gasObjectID string, // Specific gas object ID for the transaction
	gasBudget uint64,
) (models.TransactionBlockResponse, error) { // Return type changed
	utils.LogInfof("MarketSuiService: Preparing to list NFT %s for sale by %s at %d %s. GasObject: %s, GasBudget: %d",
		nftID, sellerAddress, price, currency, gasObjectID, gasBudget)

	if gasObjectID == "" {
		utils.LogError("MarketSuiService: gasObjectID must be provided for ListNFTForSale")
		return models.TransactionBlockResponse{}, fmt.Errorf("gasObjectID must be provided for ListNFTForSale")
	}

	// Prepare arguments for the Move function
	// The exact arguments depend on the 'list_nft' function signature in your Move contract.
	// Assuming the function is generic: list_nft<NFT_GENERIC_TYPE, COIN_GENERIC_TYPE>(marketplace_obj, nft_obj_to_list, price, description, option_duration_hours)
	arguments := []interface{}{
		s.config.MarketplaceObjectID,  // marketplace object
		nftID,                         // NFT object to list
		strconv.FormatUint(price, 10), // price
		description,                   // description as string
	}

	// Add duration if specified (Move contract must support Option<u64> for duration)
	// The sui-go-sdk might require specific formatting for Option types.
	// Assuming the Move function takes duration as a direct u64 or Option<u64>.
	// If it's Option<u64>, the representation might be more complex or require helper functions.
	// For simplicity, if durationHours is nil, we might need to call a different Move function
	// or the Move function itself handles a default/no expiry.
	// This example assumes duration is a u64 and 0 means no expiry, or contract handles Option.
	if durationHours != nil {
		arguments = append(arguments, *durationHours) // Or however Option<u64> is represented
	} else {
		// If the function expects Option<vector<u64>> for optional args, or similar complex structure:
		// arguments = append(arguments, []interface{}{}) // Represents None for an Option<vector<T>>
		// Or if the argument is simply omitted or a specific value like 0 indicates no expiry.
		// This needs to match the Move contract. For now, let's assume it's an optional last arg or handled by value.
		// arguments = append(arguments, uint64(0)) // Placeholder for "no specific duration" if contract expects a u64
	}
	// Type arguments - specify the NFT type and Coin type if the Move function is generic.
	// Example: list_nft<MyNFT, CoinType>(...)
	// This depends heavily on the contract definition.
	typeArgs := []string{nftType, currency} // Assuming nftType is full struct type, currency is full coin struct type

	txBlockResponse, err := s.client.MoveCall(
		sellerAddress,
		s.config.PackageID,
		s.config.Module,
		"list_nft", // Ensure this is the correct function name in your Move module
		typeArgs,
		arguments,
		gasObjectID, // Crucial: Use the provided gas object ID
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("MarketSuiService: MoveCall for ListNFTForSale failed for NFT %s: %v", nftID, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for ListNFTForSale (NFT: %s): %w", nftID, err)
	}

	// The transaction is prepared. Signing and execution are handled by the caller.
	// The txBlockResponse.TxBytes contains the bytes to be signed.
	// txBlockResponse.Effects can be examined after execution to get created objects (e.g., listing object ID).
	utils.LogInfof("MarketSuiService: ListNFTForSale transaction prepared for NFT %s. TxBytes: %s", nftID, txBlockResponse.TxBytes)
	// Returning the full response; the caller can extract TxBytes or other info.
	return txBlockResponse, nil
}

// PurchaseNFT prepares a transaction to buy an NFT from the marketplace.
// Returns the transaction bytes. Requires buyer's gas object and payment coin object.
func (s *MarketSuiService) PurchaseNFT(
	buyerAddress string,
	listingObjectID string, // ID of the listing object on the marketplace
	paymentCoinID string,   // ID of the coin object used for payment
	nftType string,         // Fully qualified type of the NFT being purchased
	coinType string,        // Fully qualified type of the coin being used for payment
	gasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) { // Return type changed
	utils.LogInfof("MarketSuiService: Player %s purchasing from listing %s with coin %s. GasObject: %s, GasBudget: %d",
		buyerAddress, listingObjectID, paymentCoinID, gasObjectID, gasBudget)

	if gasObjectID == "" || paymentCoinID == "" || listingObjectID == "" {
		errMsg := "gasObjectID, paymentCoinID, and listingObjectID must be provided for PurchaseNFT"
		utils.LogError("MarketSuiService: " + errMsg)
		return models.TransactionBlockResponse{}, fmt.Errorf(errMsg)
	}
	// Prepare arguments for the Move function. This depends on your 'purchase_nft' contract function.
		s.config.MarketplaceObjectID, // marketplace object (shared or owned by contract)
		listingObjectID,              // The ID of the listing object
		paymentCoinID,                // Payment coin object ID
	}

	// Type arguments - specify the NFT type and Coin type if the Move function is generic.
	// Example: purchase_nft<MyNFT, CoinType>(...)
	typeArgs := []string{nftType, coinType} // nftType is the actual type of the NFT being bought, coinType is the currency

	txBlockResponse, err := s.client.MoveCall(
		buyerAddress,
		s.config.PackageID,
		s.config.Module,
		"purchase_nft", // Ensure this is the correct function name
		typeArgs,
		arguments,
		gasObjectID, // Buyer's gas object
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("MarketSuiService: MoveCall for PurchaseNFT failed for listing %s by %s: %v", listingObjectID, buyerAddress, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for PurchaseNFT (listing: %s): %w", listingObjectID, err)
	}

	// Transaction is prepared. Caller handles signing & execution.
	// Effects of this transaction (after execution) would confirm transfer of NFT and coin.
	// A PurchaseResult struct might be populated by parsing events from SuiExecuteTransactionBlockResponse.
	utils.LogInfof("MarketSuiService: PurchaseNFT transaction prepared for listing %s by %s. TxBytes: %s",
		listingObjectID, buyerAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// CancelListing prepares a transaction to cancel an NFT listing.
// Returns transaction bytes. Requires seller's gas object.
// nftType and coinType are the generic types used when the item was listed.
func (s *MarketSuiService) CancelListing(
	sellerAddress string,
	listingObjectID string, // ID of the listing object to cancel
	nftType string,         // Fully qualified type of the NFT that was listed
	coinType string,        // Fully qualified type of the coin that was expected
	gasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) { // Return type changed
	utils.LogInfof("MarketSuiService: Player %s canceling listing %s. GasObject: %s, GasBudget: %d",
		sellerAddress, listingObjectID, gasObjectID, gasBudget)

	if gasObjectID == "" || listingObjectID == "" {
		errMsg := "gasObjectID and listingObjectID must be provided for CancelListing"
		utils.LogError("MarketSuiService: " + errMsg)
		return models.TransactionBlockResponse{}, fmt.Errorf(errMsg)
	}

	// Prepare arguments for the Move function. Depends on 'cancel_listing' contract function.
	arguments := []interface{}{
		s.config.MarketplaceObjectID, // marketplace object
		listingObjectID,              // Listing object ID to cancel
	}

	// Type arguments if cancel_listing is generic
	typeArgs := []string{nftType, coinType}

	txBlockResponse, err := s.client.MoveCall(
		sellerAddress,
		s.config.PackageID,
		s.config.Module,
		"cancel_listing", // Ensure this is the correct function name
		typeArgs,
		arguments,
		gasObjectID, // Seller's gas object
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("MarketSuiService: MoveCall for CancelListing failed for listing %s: %v", listingObjectID, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for CancelListing (listing: %s): %w", listingObjectID, err)
	}

	utils.LogInfof("MarketSuiService: CancelListing transaction prepared for listing %s. TxBytes: %s",
		listingObjectID, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// GetListings retrieves active listings from the marketplace by querying events.
// This is a conceptual implementation. A robust solution might involve a dedicated indexer or
// querying shared Listing objects if they are structured that way.
// The eventType should be the fully qualified type of the ListingCreated event.
func (s *MarketSuiService) GetListings(eventType string, limit int, cursor *string) ([]ListingInfo, *string, error) {
	utils.LogInfo("MarketSuiService: Fetching active market listings from Sui events")

	query := models.EventFilter{
		MoveEventType: &eventType, // Example: "0xPACKAGE::marketplace::ListingCreatedEvent"
	}

	actualLimit := uint64(50) // Default limit
	if limit > 0 {
		actualLimit = uint64(limit)
	}
	limitPtr := &actualLimit

	// QueryEvents now returns models.SuiQueryEventsResponse
	sdkResponse, err := s.client.QueryEvents(query, cursor, limitPtr, true) // true for descending (newest first)
	if err != nil {
		utils.LogErrorf("MarketSuiService: Failed to query marketplace events type %s: %v", eventType, err)
		return nil, nil, fmt.Errorf("failed to query marketplace events type %s: %w", eventType, err)
	}

	var listings []ListingInfo
	for _, event := range sdkResponse.Data {
		// The structure of event.ParsedJson depends on your Move event definition.
		// Example: if your event has fields listing_id, seller, nft_id, price, currency_type, created_at
		// Need to safely extract and parse these fields.
		// For this example, we assume ParsedJson is a map[string]interface{} after JSON unmarshal.
		parsedJSON, ok := event.ParsedJson.(map[string]interface{})
		if !ok {
			utils.LogWarnf("MarketSuiService: Could not parse event JSON for event ID %s:%d", event.Id.TxDigest, event.Id.EventSeq)
			continue
		}

		var listing ListingInfo
		if id, ok := parsedJSON["listing_id"].(string); ok {
			listing.ID = id
		}
		if seller, ok := parsedJSON["seller"].(string); ok {
			listing.Seller = seller
		}
		if nftID, ok := parsedJSON["nft_id"].(string); ok {
			listing.NFTID = nftID
		}
		if nftType, ok := parsedJSON["nft_type"].(string); ok { // Assuming nft_type is part of event
			listing.NFTType = nftType
		}
		if priceStr, ok := parsedJSON["price"].(string); ok { // Price might be string in event
			if p, err := strconv.ParseUint(priceStr, 10, 64); err == nil {
				listing.Price = p
			}
		}
		if currency, ok := parsedJSON["currency"].(string); ok { // Assuming currency is part of event
			listing.Currency = currency
		}
		if createdAtStr, ok := parsedJSON["created_at"].(string); ok { // Timestamp might be string
			if ts, err := strconv.ParseUint(createdAtStr, 10, 64); err == nil {
				listing.CreatedAt = ts
			}
		}
		// Description and ExpiresAt might not be in simple ListingCreatedEvent, could be on Listing object itself.
		listings = append(listings, listing)
	}

	var nextCursorStr *string
	if sdkResponse.HasNextPage && sdkResponse.NextCursor != nil {
		// Construct string cursor for the next call, if our client.QueryEvents expects string
		// Or pass sdkResponse.NextCursor directly if client.QueryEvents is updated
		strCursor := fmt.Sprintf("%s:%d", sdkResponse.NextCursor.TxDigest, sdkResponse.NextCursor.EventSeq)
		nextCursorStr = &strCursor
	}

	utils.LogInfof("MarketSuiService: Retrieved %d active listings from events.", len(listings))
	return listings, nextCursorStr, nil
}

// GetListingInfo retrieves detailed information about a specific listing object.
func (s *MarketSuiService) GetListingInfo(listingObjectID string) (*ListingInfo, error) {
	utils.LogInfof("MarketSuiService: Fetching listing info for object ID %s", listingObjectID)

	objectResponse, err := s.client.GetObject(listingObjectID)
	if err != nil {
		utils.LogErrorf("MarketSuiService: Failed to get listing object %s: %v", listingObjectID, err)
		return nil, fmt.Errorf("failed to get listing object %s: %w", listingObjectID, err)
	}

	if objectResponse.Data == nil || objectResponse.Data.Content == nil {
		utils.LogWarnf("MarketSuiService: Listing object %s not found or has no content.", listingObjectID)
		return nil, fmt.Errorf("listing object %s not found or has no content", listingObjectID)
	}

	fields, ok := objectResponse.Data.Content.Fields.(map[string]interface{})
	if !ok {
		utils.LogWarnf("MarketSuiService: Could not parse fields for listing object %s.", listingObjectID)
		return nil, fmt.Errorf("could not parse fields for listing object %s", listingObjectID)
	}

	// Assuming the Listing object on-chain has fields like: id (ObjectID), seller, nft_id, nft_type, price, currency_type etc.
	// This mapping needs to match your Move struct for the Listing.
	listing := &ListingInfo{
		ID: listingObjectID, // The object ID itself is the listing ID
	}
	if seller, ok := fields["seller"].(string); ok {
		listing.Seller = seller
	}
	if nftID, ok := fields["nft_id"].(string); ok {
		listing.NFTID = nftID
	}
	if nftType, ok := fields["nft_type"].(string); ok { // Actual NFT type string
		listing.NFTType = nftType
	}
	if priceStr, ok := fields["price"].(string); ok { // Price might be string in object fields
		if p, err := strconv.ParseUint(priceStr, 10, 64); err == nil {
			listing.Price = p
		}
	}
	if currency, ok := fields["currency_type"].(string); ok { // Full currency type string
		listing.Currency = currency
	}
	if createdAtStr, ok := fields["created_at_ms"].(string); ok { // Example field name
		if ts, err := strconv.ParseUint(createdAtStr, 10, 64); err == nil {
			listing.CreatedAt = ts
		}
	}
	// Populate ExpiresAt and Description if they exist in fields

	return listing, nil
}

// GetMarketplaceInfo retrieves marketplace statistics from the shared marketplace object.
func (s *MarketSuiService) GetMarketplaceInfo() (*MarketplaceInfo, error) {
	utils.LogInfo("MarketSuiService: Fetching marketplace information")

	objectResponse, err := s.client.GetObject(s.config.MarketplaceObjectID)
	if err != nil {
		utils.LogErrorf("MarketSuiService: Failed to get marketplace object %s: %v", s.config.MarketplaceObjectID, err)
		return nil, fmt.Errorf("failed to get marketplace object %s: %w", s.config.MarketplaceObjectID, err)
	}

	if objectResponse.Data == nil || objectResponse.Data.Content == nil {
		utils.LogWarnf("MarketSuiService: Marketplace object %s not found or has no content.", s.config.MarketplaceObjectID)
		return nil, fmt.Errorf("marketplace object %s not found or has no content", s.config.MarketplaceObjectID)
	}

	fields, ok := objectResponse.Data.Content.Fields.(map[string]interface{})
	if !ok {
		utils.LogWarnf("MarketSuiService: Could not parse fields for marketplace object %s.", s.config.MarketplaceObjectID)
		return nil, fmt.Errorf("could not parse fields for marketplace object %s", s.config.MarketplaceObjectID)
	}

	// Assuming fields like 'fee_percentage', 'listing_count', 'treasury' (which itself is an object with a 'balance' field)
	info := &MarketplaceInfo{}
	if feeStr, ok := fields["fee_percentage"].(string); ok { // Assuming it's a string needing parsing
		if fee, err := strconv.ParseUint(feeStr, 10, 64); err == nil {
			info.FeePercentage = fee
		}
	}
	if countStr, ok := fields["listing_count"].(string); ok { // Assuming it's a string
		if count, err := strconv.ParseUint(countStr, 10, 64); err == nil {
			info.ListingCount = count
		}
	}
	if treasuryMap, ok := fields["treasury"].(map[string]interface{}); ok {
		if balanceMap, ok := treasuryMap["fields"].(map[string]interface{}); ok {
			if balanceStr, ok := balanceMap["balance"].(string); ok { // Assuming balance is string in Coin object
				if bal, err := strconv.ParseUint(balanceStr, 10, 64); err == nil {
					info.TreasuryBalance = bal
				}
			}
		}
	}

	utils.LogInfof("MarketSuiService: Marketplace info: fee=%d%%, listings=%d, treasury=%d",
		info.FeePercentage, info.ListingCount, info.TreasuryBalance)

	return info, nil
}

// GetPlayerNFTs retrieves NFTs owned by a player.
// This is a general utility; filtering for listable NFTs might require more specific type checks.
func (s *MarketSuiService) GetPlayerNFTs(playerAddress string, specificNftType *string) ([]map[string]interface{}, error) {
	utils.LogInfof("MarketSuiService: Fetching NFTs for player %s (Type filter: %v)", playerAddress, specificNftType)

	// GetOwnedObjects already returns models.SuiGetOwnedObjectsResponse
	sdkResponse, err := s.client.GetOwnedObjects(playerAddress, specificNftType)
	if err != nil {
		utils.LogErrorf("MarketSuiService: Failed to get owned objects for player %s: %v", playerAddress, err)
		return nil, fmt.Errorf("failed to get owned objects for player %s: %w", playerAddress, err)
	}

	var nfts []map[string]interface{}
	for _, objInfo := range sdkResponse.Data {
		if objInfo.Data == nil {
			continue
		}
		// Basic info readily available in objInfo.Data
		nft := map[string]interface{}{
			"object_id": objInfo.Data.ObjectId,
			"type":      *objInfo.Data.Type, // Type is a pointer string
			"version":   objInfo.Data.Version.String(), // Version is models.SequenceNumber (string)
			"digest":    objInfo.Data.Digest,
		}

		// Add content if available and needed (already parsed by SDK if ShowContent is true)
		if objInfo.Data.Content != nil && objInfo.Data.Content.Fields != nil {
			// The Fields interface{} can be complex. For simplicity, just assign it.
			// Or, you could try to cast to map[string]interface{} if you know the structure.
			nft["content_fields"] = objInfo.Data.Content.Fields
		}
		nfts = append(nfts, nft)
	}

	utils.LogInfof("MarketSuiService: Found %d NFTs for player %s", len(nfts), playerAddress)
	return nfts, nil
}

// IsNFTListed checks if an NFT is currently part of an active listing.
// This is a conceptual placeholder. A real implementation would need to:
// 1. Query a global index of listings by NFT ID (if such an index exists as a dynamic field on a shared object).
// 2. Or, if the NFT itself holds a capability/receipt when listed, check for that.
// 3. Or, iterate through all Listing objects (inefficient) or events (potentially slow/incomplete for current state).
func (s *MarketSuiService) IsNFTListed(nftID string) (bool, error) {
	utils.LogInfof("MarketSuiService: Checking if NFT %s is listed (placeholder implementation)", nftID)
	// Placeholder - this requires specific contract design for efficient lookup.
	// For example, the marketplace contract might store a mapping from NFT ID to Listing ID.
	// Or one might query for Listing objects that have a field `nft_id == nftID`.
	// This cannot be reliably implemented without knowing the contract's query patterns.
	return false, fmt.Errorf("IsNFTListed is a placeholder and not implemented functionally")
}

// Helper function to parse Sui events into structured data
// This function needs to be adapted based on the actual structure of your Move events.
func (s *MarketSuiService) parseMarketplaceEvent(event models.SuiEvent) (map[string]interface{}, error) {
	// event.ParsedJson should already be a map[string]interface{} or similar parsed structure
	// from the sui-go-sdk if the event content is valid JSON.
	parsedData, ok := event.ParsedJson.(map[string]interface{})
	if !ok {
		utils.LogWarnf("MarketSuiService: Could not assert ParsedJson to map[string]interface{} for event type %s", event.Type)
		// Try to marshal and unmarshal if it's a different map type, e.g. map[interface{}]interface{}
		// This is a fallback, direct assertion is better.
		jsonData, err := json.Marshal(event.ParsedJson)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ParsedJson for event type %s: %w", event.Type, err)
		}
		err = json.Unmarshal(jsonData, &parsedData)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal ParsedJson into map for event type %s: %w", event.Type, err)
		}
	}

	result := map[string]interface{}{
		"event_type":         event.Type,
		"package_id":         event.PackageId,
		"transaction_module": event.TransactionModule,
		"sender":             event.Sender,
		"timestamp_ms":       event.TimestampMs, // TimestampMs is already a string
		"tx_digest":          event.Id.TxDigest,
		"event_seq":          event.Id.EventSeq.String(), // EventSeq is uint64
	}

	// Add all fields from parsedData
	for key, value := range parsedData {
		result[key] = value
	}
	return result, nil
}

// GetMarketplaceEvents retrieves marketplace events (e.g., purchases, listings, cancellations).
// eventTypeFilter is the fully qualified event type, e.g., "0xPACKAGE::marketplace::ListingCreatedEvent"
func (s *MarketSuiService) GetMarketplaceEvents(eventTypeFilter string, limit int, cursor *string) ([]map[string]interface{}, *string, error) {
	utils.LogInfof("MarketSuiService: Fetching marketplace events of type: %s", eventTypeFilter)

	query := models.EventFilter{}
	if eventTypeFilter != "" {
		query.MoveEventType = &eventTypeFilter
	} else {
		// If no specific eventTypeFilter, query for all events from the module
		moduleFilter := fmt.Sprintf("%s::%s", s.config.PackageID, s.config.Module)
		query.MoveModule = &models.MoveModule{Package: s.config.PackageID, Module: s.config.Module}
		utils.LogInfof("MarketSuiService: No specific event type provided, querying for all events from module %s", moduleFilter)
	}


	actualLimit := uint64(50) // Default limit
	if limit > 0 {
		actualLimit = uint64(limit)
	}
	limitPtr := &actualLimit

	sdkResponse, err := s.client.QueryEvents(query, cursor, limitPtr, true) // true for descending
	if err != nil {
		utils.LogErrorf("MarketSuiService: Failed to query events (filter: %s): %v", eventTypeFilter, err)
		return nil, nil, fmt.Errorf("failed to query events (filter: %s): %w", eventTypeFilter, err)
	}

	var parsedEvents []map[string]interface{}
	for _, eventData := range sdkResponse.Data {
		parsedEvent, err := s.parseMarketplaceEvent(eventData)
		if err != nil {
			utils.LogWarnf("MarketSuiService: Could not parse event data for event ID %s:%d: %v", eventData.Id.TxDigest, eventData.Id.EventSeq, err)
			continue // Skip this event
		}
		parsedEvents = append(parsedEvents, parsedEvent)
	}

	var nextCursorStr *string
	if sdkResponse.HasNextPage && sdkResponse.NextCursor != nil {
		strCursor := fmt.Sprintf("%s:%d", sdkResponse.NextCursor.TxDigest, sdkResponse.NextCursor.EventSeq)
		nextCursorStr = &strCursor
		utils.LogDebugf("MarketSuiService: Next cursor for events: %s", strCursor)
	}

	utils.LogInfof("MarketSuiService: Retrieved %d marketplace events (filter: %s).", len(parsedEvents), eventTypeFilter)
	return parsedEvents, nextCursorStr, nil
}
