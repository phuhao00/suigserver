# Marketplace System

A comprehensive NFT marketplace system for the Sui MMO Game, enabling players to buy, sell, and trade in-game NFTs.

## Features

### Core Functionality
- **List NFTs for Sale**: Players can list their NFTs with custom prices and descriptions
- **Purchase NFTs**: Buy listed NFTs with SUI tokens
- **Cancel Listings**: Sellers can cancel their listings and retrieve their NFTs
- **Expiring Listings**: Support for time-limited listings that automatically expire
- **Marketplace Fees**: Configurable marketplace fees with admin controls

### Advanced Features
- **Update Listing Price**: Sellers can update the price of their active listings
- **Extend Listing Duration**: Sellers can extend the expiry time of their listings
- **Remove Expired Listings**: Anyone can remove expired listings, returning NFTs to sellers
- **Admin Fee Management**: Admins can update marketplace fees and withdraw collected fees
- **Comprehensive Events**: All marketplace activities emit events for tracking

### Query Functions
- Get marketplace information (fee percentage, listing count, treasury balance)
- Get listing details for any NFT
- Check if an NFT is currently listed
- Get purchase receipt information

## Contract Structure

### Main Types

#### `Marketplace`
The main marketplace object that holds:
- Configuration (fee percentage, admin address)
- Treasury for collected fees
- Active listings table
- Listing statistics

#### `Listing`
Individual NFT listing containing:
- Seller information
- NFT details and type
- Price and currency
- Creation and expiry timestamps
- Description

#### `PurchaseReceipt`
Receipt issued after successful purchases containing:
- Buyer and seller addresses
- NFT information
- Price paid and marketplace fee
- Purchase timestamp

#### `AdminCap`
Administrative capability for marketplace management

### Error Codes
- `E_NOT_AUTHORIZED`: Unauthorized action attempt
- `E_LISTING_NOT_FOUND`: NFT listing doesn't exist
- `E_INSUFFICIENT_PAYMENT`: Payment amount too low
- `E_ITEM_NOT_AVAILABLE`: NFT not available (expired/sold)
- `E_INVALID_PRICE`: Invalid price (zero or negative)
- `E_MARKETPLACE_FEE_TOO_HIGH`: Fee exceeds maximum allowed
- `E_ALREADY_LISTED`: NFT already has an active listing

## Usage Examples

### Listing an NFT
```move
marketplace::list_nft(
    &mut marketplace,
    my_nft,
    b"player_character",
    1000000, // 1 SUI in MIST
    b"SUI",
    b"Rare level 50 warrior",
    option::some(168), // 7 days
    ctx
);
```

### Purchasing an NFT
```move
let payment = coin::split(&mut my_sui_coins, 1000000, ctx);
marketplace::purchase_nft_entry<PlayerNFT>(
    &mut marketplace,
    nft_id,
    payment,
    ctx
);
```

### Canceling a Listing
```move
marketplace::cancel_listing<PlayerNFT>(
    &mut marketplace,
    nft_id,
    ctx
);
```

### Updating Listing Price
```move
marketplace::update_listing_price(
    &mut marketplace,
    nft_id,
    2000000, // New price: 2 SUI
    ctx
);
```

## Security Features

1. **Authorization Checks**: Only sellers can modify their own listings
2. **Expiry Validation**: Automatic validation of listing expiry times
3. **Payment Verification**: Ensures sufficient payment before transfers
4. **Fee Limits**: Maximum marketplace fee of 10%
5. **Duplicate Prevention**: Prevents listing the same NFT multiple times

## Fee Structure

- Default marketplace fee: 2.5%
- Maximum allowed fee: 10%
- Fees are collected in the marketplace treasury
- Admin can withdraw fees and update fee percentage

## Testing

The contract includes comprehensive tests covering:
- Basic functionality (list, purchase, cancel)
- Admin operations (fee management, withdrawals)
- Error conditions and edge cases
- Authorization and security checks

Run tests with:
```bash
sui move test
```

## Deployment

1. Build the contract:
```bash
sui move build
```

2. Deploy to testnet/mainnet:
```bash
sui client publish --gas-budget 50000000
```

3. The deployment will create:
   - A shared `Marketplace` object
   - An `AdminCap` transferred to the deployer

## Integration

The marketplace can handle any NFT type that implements `key + store`, making it compatible with:
- Player character NFTs
- Item NFTs
- Guild badges
- Land parcels
- Any other game assets

## Events

All marketplace activities emit events for off-chain tracking:
- `ListingCreated`: When an NFT is listed
- `NFTPurchased`: When an NFT is purchased
- `ListingCanceled`: When a listing is canceled
- `ListingExpired`: When a listing expires

These events can be monitored to build marketplace UIs and analytics.
{
  "package_id": "0x...",
  "marketplace_object_id": "0x...",
  "admin_cap_id": "0x...",
  "sui_node_url": "https://fullnode.testnet.sui.io:443"
}
```

### 3. Run the Server

```bash
cd examples
go run marketplace_server.go
```

The server will start on `http://localhost:8080` with the following endpoints:

## API Endpoints

### List NFT for Sale
```http
POST /api/marketplace/list
Content-Type: application/json

{
  "seller_address": "0x...",
  "nft_id": "0x...",
  "nft_type": "player",
  "price": 1000,
  "currency": "SUI",
  "description": "Level 50 Warrior",
  "duration_hours": 168
}
```

### Purchase NFT
```http
POST /api/marketplace/purchase
Content-Type: application/json

{
  "buyer_address": "0x...",
  "nft_id": "0x...",
  "payment_coin_id": "0x..."
}
```

### Cancel Listing
```http
POST /api/marketplace/cancel
Content-Type: application/json

{
  "seller_address": "0x...",
  "nft_id": "0x..."
}
```

### Get Active Listings
```http
GET /api/marketplace/listings?limit=50&cursor=...
```

### Get Player NFTs
```http
GET /api/marketplace/player-nfts?address=0x...
```

### Get Marketplace Info
```http
GET /api/marketplace/info
```

### Get Marketplace Events
```http
GET /api/marketplace/events?type=ListingCreated&limit=50
```

### Get Service Statistics
```http
GET /api/marketplace/stats
```

## Go SDK Usage

### Basic Usage

```go
package main

import (
    "log"
    "github.com/phuhao00/suigserver/server/configs"
    "github.com/phuhao00/suigserver/server/internal/sui"
)

func main() {
    // Load configuration
    config, err := configs.LoadMarketplaceConfig("configs/marketplace.json")
    if err != nil {
        log.Fatal(err)
    }

    // Create marketplace manager
    manager, err := sui.NewMarketplaceServiceManager(config)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()

    // List an NFT
    listingID, err := manager.ListNFTForSale(
        "0x123...", // seller address
        "0xabc...", // NFT ID
        "player",   // NFT type
        1000,       // price in MIST (1 SUI = 1,000,000,000 MIST)
        "SUI",      // currency
        "Level 50 Warrior with rare sword",
        nil,        // no expiration
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Listed NFT with ID: %s", listingID)

    // Get marketplace info
    info, err := manager.GetMarketplaceInfo()
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Marketplace has %d active listings", info.ListingCount)
}
```

### Advanced Features

```go
// Get listings with pagination
listings, nextCursor, err := manager.GetListings(50, nil)
if err != nil {
    log.Fatal(err)
}

for _, listing := range listings {
    log.Printf("Listing: %s - %s for %d %s", 
        listing.NFTID, listing.NFTType, listing.Price, listing.Currency)
}

// Get player's NFTs
nfts, err := manager.GetPlayerNFTs("0x123...")
if err != nil {
    log.Fatal(err)
}

log.Printf("Player owns %d NFTs", len(nfts))

// Purchase an NFT
result, err := manager.PurchaseNFT(
    "0x456...", // buyer address
    "0xabc...", // NFT ID
    "0xdef...", // payment coin ID
)
if err != nil {
    log.Fatal(err)
}

log.Printf("Purchase completed: %s", result.TransactionDigest)
```

## Move Contract Reference

### Key Functions

#### `list_nft<T>`
Lists an NFT for sale on the marketplace.

**Parameters:**
- `marketplace: &mut Marketplace` - The marketplace object
- `nft: T` - The NFT to list (must have `key + store`)
- `nft_type: vector<u8>` - Type description (e.g., "player", "item")
- `price: u64` - Price in the smallest currency unit
- `currency: vector<u8>` - Currency type (e.g., "SUI")
- `description: vector<u8>` - Human-readable description
- `duration_hours: Option<u64>` - Optional expiration time

#### `purchase_nft<T>`
Purchases an NFT from the marketplace.

**Parameters:**
- `marketplace: &mut Marketplace` - The marketplace object
- `nft_id: ID` - ID of the NFT to purchase
- `payment: Coin<SUI>` - Payment coin

**Returns:** `PurchaseReceipt` - Receipt of the purchase

#### `cancel_listing<T>`
Cancels an NFT listing and returns the NFT to the seller.

**Parameters:**
- `marketplace: &mut Marketplace` - The marketplace object
- `nft_id: ID` - ID of the NFT to cancel

### Events

#### `ListingCreated`
Emitted when an NFT is listed for sale.

#### `NFTPurchased`
Emitted when an NFT is purchased.

#### `ListingCanceled`
Emitted when a listing is canceled.

### View Functions

#### `get_marketplace_info`
Returns marketplace statistics (fee percentage, listing count, treasury balance).

#### `get_listing_info`
Returns detailed information about a specific listing.

#### `is_listed`
Checks if an NFT is currently listed.

## Configuration

### Marketplace Configuration

```json
{
  "sui_node_url": "https://fullnode.testnet.sui.io:443",
  "package_id": "0x...",
  "marketplace_object_id": "0x...",
  "admin_cap_id": "0x...",
  "module": "marketplace",
  "default_gas_budget": 1000000,
  "max_listing_duration_hours": 168,
  "enable_caching": true,
  "cache_expiration_seconds": 300,
  "rate_limit_enabled": true,
  "rate_limit_per_minute": 100
}
```

### Configuration Options

- `sui_node_url`: Sui RPC endpoint URL
- `package_id`: Published marketplace package ID
- `marketplace_object_id`: Shared marketplace object ID
- `admin_cap_id`: Admin capability object ID
- `default_gas_budget`: Default gas budget for transactions
- `max_listing_duration_hours`: Maximum allowed listing duration
- `enable_caching`: Enable response caching
- `cache_expiration_seconds`: Cache expiration time
- `rate_limit_enabled`: Enable rate limiting
- `rate_limit_per_minute`: Requests per minute per user

## Testing

### Run Move Tests

```bash
cd contracts/marketplace_system
sui move test
```

### Run Go Tests

```bash
cd server/internal/sui
go test -v
```

### Integration Testing

The test suite includes comprehensive tests for:
- Marketplace initialization
- NFT listing and cancellation
- Purchase transactions
- Fee collection and withdrawal
- Error handling and edge cases
- Rate limiting and caching

## Security Considerations

1. **Access Control**: Only NFT owners can list their NFTs
2. **Atomic Transactions**: All marketplace operations are atomic
3. **Fee Protection**: Marketplace fees are automatically collected
4. **Expiration Handling**: Expired listings cannot be purchased
5. **Rate Limiting**: Prevents spam and abuse
6. **Input Validation**: All inputs are validated before processing

## Performance Optimization

1. **Caching**: Frequently accessed data is cached to reduce blockchain queries
2. **Event Indexing**: Events are indexed for efficient querying
3. **Batch Operations**: Multiple NFT operations can be batched
4. **Gas Optimization**: Smart contract is optimized for minimal gas usage

## Deployment Checklist

1. ✅ Deploy marketplace Move contract
2. ✅ Initialize marketplace with admin capability
3. ✅ Configure marketplace fees (default: 2.5%)
4. ✅ Set up monitoring for marketplace events
5. ✅ Configure backend service with contract addresses
6. ✅ Test all marketplace operations on testnet
7. ✅ Set up rate limiting and caching
8. ✅ Deploy HTTP API server
9. ✅ Monitor marketplace statistics and treasury

## Support

For questions, issues, or contributions, please refer to the main project documentation or open an issue in the repository.

## License

This marketplace system is part of the MMO game project and follows the same licensing terms.
