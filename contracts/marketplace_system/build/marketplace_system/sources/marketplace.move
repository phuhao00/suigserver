// Marketplace System Contract for Sui MMO Game
// Enables buying, selling, and trading of in-game NFTs

#[allow(duplicate_alias, unused_use, unused_const, unused_field, lint(missing_key))]
module mmo_game::marketplace {
    use std::string::{Self, String};
    use std::option::{Self, Option};
    use sui::object::{Self, UID, ID};
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};
    use sui::event;    use sui::table::{Self, Table};
    use sui::clock::{Self, Clock};
    use sui::coin::{Self, Coin};
    use sui::sui::SUI;
    use sui::balance::{Self, Balance};
    use sui::dynamic_field;

    // Error codes
    const E_NOT_AUTHORIZED: u64 = 1;
    const E_LISTING_NOT_FOUND: u64 = 2;
    const E_INSUFFICIENT_PAYMENT: u64 = 3;
    const E_ITEM_NOT_AVAILABLE: u64 = 4;
    const E_INVALID_PRICE: u64 = 5;
    const E_MARKETPLACE_FEE_TOO_HIGH: u64 = 6;
    const E_ALREADY_LISTED: u64 = 7;

    // Constants
    const MAX_FEE_PERCENTAGE: u64 = 1000; // 10% max fee (basis points)
    const BASIS_POINTS: u64 = 10000; // 100% = 10000 basis points

    // Admin capability
    public struct AdminCap has key, store {
        id: UID,
    }

    /// Marketplace configuration and state
    public struct Marketplace has key {
        id: UID,
        fee_percentage: u64, // In basis points (100 = 1%)
        treasury: Balance<SUI>,
        active_listings: Table<ID, Listing>,
        listing_count: u64,
        admin: address,
    }

    /// Individual NFT listing
    public struct Listing has store {
        id: UID,
        seller: address,
        nft_id: ID,
        nft_type: String, // "player", "item", "guild_badge", etc.
        price: u64,
        currency: String, // "SUI" for now, could support other currencies
        created_at: u64,        expires_at: Option<u64>,
        description: String,
    }

    /// Purchase receipt
    public struct PurchaseReceipt has key, store {
        id: UID,
        buyer: address,
        seller: address,
        nft_id: ID,
        price_paid: u64,
        marketplace_fee: u64,
        purchase_time: u64,
    }

    // Events
    public struct ListingCreated has copy, drop {
        listing_id: ID,
        seller: address,
        nft_id: ID,
        nft_type: String,
        price: u64,
        currency: String,
        timestamp: u64,
    }

    public struct ListingCanceled has copy, drop {
        listing_id: ID,
        seller: address,
        nft_id: ID,
        timestamp: u64,
    }

    public struct NFTPurchased has copy, drop {
        listing_id: ID,
        buyer: address,
        seller: address,
        nft_id: ID,
        price_paid: u64,
        marketplace_fee: u64,
        timestamp: u64,
    }

    public struct ListingExpired has copy, drop {
        listing_id: ID,
        nft_id: ID,
        timestamp: u64,
    }

    /// Initialize the marketplace
    fun init(ctx: &mut TxContext) {
        let admin_cap = AdminCap {
            id: object::new(ctx),
        };

        let marketplace = Marketplace {
            id: object::new(ctx),
            fee_percentage: 250, // 2.5% default fee
            treasury: balance::zero(),
            active_listings: table::new(ctx),
            listing_count: 0,            admin: tx_context::sender(ctx),
        };
        
        transfer::transfer(admin_cap, tx_context::sender(ctx));
        transfer::share_object(marketplace);
    }

    /// List an NFT for sale
    public entry fun list_nft<T: key + store>(
        marketplace: &mut Marketplace,
        nft: T,
        nft_type: vector<u8>,
        price: u64,
        currency: vector<u8>,
        description: vector<u8>,
        mut duration_hours: Option<u64>,
        ctx: &mut TxContext
    ) {
        assert!(price > 0, E_INVALID_PRICE);
        
        let seller = tx_context::sender(ctx);
        let nft_id = object::id(&nft);
        
        // Check if NFT is already listed
        assert!(!table::contains(&marketplace.active_listings, nft_id), E_ALREADY_LISTED);

        let listing_uid = object::new(ctx);
        let listing_id = object::uid_to_inner(&listing_uid);
        let current_time = tx_context::epoch(ctx);
          let expires_at = if (option::is_some(&duration_hours)) {
            let hours = option::extract(&mut duration_hours);
            option::some(current_time + (hours * 3600)) // Convert hours to seconds
        } else {
            option::none()
        };
        
        let mut listing = Listing {
            id: listing_uid,
            seller,
            nft_id,
            nft_type: string::utf8(nft_type),
            price,
            currency: string::utf8(currency),
            created_at: current_time,
            expires_at,
            description: string::utf8(description),        };

        // Store the NFT in dynamic field
        sui::dynamic_field::add(&mut listing.id, b"nft", nft);
        
        // Add listing to marketplace
        table::add(&mut marketplace.active_listings, nft_id, listing);
        marketplace.listing_count = marketplace.listing_count + 1;

        event::emit(ListingCreated {
            listing_id,
            seller,
            nft_id,
            nft_type: string::utf8(nft_type),
            price,
            currency: string::utf8(currency),            timestamp: current_time,
        });
    }
      /// Purchase an NFT from the marketplace
    #[allow(lint(self_transfer))]
    public fun purchase_nft<T: key + store>(
        marketplace: &mut Marketplace,
        nft_id: ID,        payment: Coin<SUI>,
        ctx: &mut TxContext
    ): PurchaseReceipt {
        assert!(table::contains(&marketplace.active_listings, nft_id), E_LISTING_NOT_FOUND);
        
        let mut listing = table::remove(&mut marketplace.active_listings, nft_id);
        let buyer = tx_context::sender(ctx);
        let payment_amount = coin::value(&payment);
        
        // Verify payment amount
        assert!(payment_amount >= listing.price, E_INSUFFICIENT_PAYMENT);
        
        // Check if listing has expired
        if (option::is_some(&listing.expires_at)) {
            let expiry = option::extract(&mut listing.expires_at);
            assert!(tx_context::epoch(ctx) <= expiry, E_ITEM_NOT_AVAILABLE);
        };
        
        // Calculate marketplace fee
        let marketplace_fee = (listing.price * marketplace.fee_percentage) / BASIS_POINTS;
        let seller_payment = listing.price - marketplace_fee;
        
        // Handle payment distribution
        let mut payment_balance = coin::into_balance(payment);
        let fee_balance = balance::split(&mut payment_balance, marketplace_fee);
        let seller_balance = balance::split(&mut payment_balance, seller_payment);
        
        // Add fee to marketplace treasury
        balance::join(&mut marketplace.treasury, fee_balance);
        
        // Send payment to seller
        let seller_coin = coin::from_balance(seller_balance, ctx);
        transfer::public_transfer(seller_coin, listing.seller);
        
        // Return excess payment to buyer if any
        if (balance::value(&payment_balance) > 0) {
            let excess_coin = coin::from_balance(payment_balance, ctx);
            transfer::public_transfer(excess_coin, buyer);
        } else {
            balance::destroy_zero(payment_balance);
        };
        
        // Transfer NFT to buyer
        let nft: T = sui::dynamic_field::remove(&mut listing.id, b"nft");
        transfer::public_transfer(nft, buyer);
        
        // Update marketplace stats
        marketplace.listing_count = marketplace.listing_count - 1;
        
        // Create purchase receipt
        let receipt = PurchaseReceipt {
            id: object::new(ctx),
            buyer,
            seller: listing.seller,
            nft_id: listing.nft_id,
            price_paid: listing.price,
            marketplace_fee,
            purchase_time: tx_context::epoch(ctx),
        };

        event::emit(NFTPurchased {
            listing_id: object::uid_to_inner(&listing.id),
            buyer,
            seller: listing.seller,
            nft_id: listing.nft_id,
            price_paid: listing.price,
            marketplace_fee,
            timestamp: tx_context::epoch(ctx),
        });        // Clean up listing
        let Listing { id, seller: _, nft_id: _, nft_type: _, price: _, currency: _, created_at: _, expires_at: _, description: _ } = listing;
        object::delete(id);
        
        receipt
    }

    /// Entry function wrapper for purchasing an NFT (handles receipt automatically)
    public entry fun purchase_nft_entry<T: key + store>(
        marketplace: &mut Marketplace,
        nft_id: ID,
        payment: Coin<SUI>,
        ctx: &mut TxContext
    ) {
        let receipt = purchase_nft<T>(marketplace, nft_id, payment, ctx);
        transfer::public_transfer(receipt, tx_context::sender(ctx));
    }

    /// Cancel a listing and return the NFT
    public entry fun cancel_listing<T: key + store>(
        marketplace: &mut Marketplace,        nft_id: ID,
        ctx: &mut TxContext
    ) {
        assert!(table::contains(&marketplace.active_listings, nft_id), E_LISTING_NOT_FOUND);
        
        let mut listing = table::remove(&mut marketplace.active_listings, nft_id);
        let sender = tx_context::sender(ctx);
        
        // Only seller can cancel their own listing
        assert!(listing.seller == sender, E_NOT_AUTHORIZED);
        
        // Return NFT to seller
        let nft: T = sui::dynamic_field::remove(&mut listing.id, b"nft");
        transfer::public_transfer(nft, listing.seller);
        
        // Update marketplace stats
        marketplace.listing_count = marketplace.listing_count - 1;

        event::emit(ListingCanceled {
            listing_id: object::uid_to_inner(&listing.id),
            seller: listing.seller,
            nft_id: listing.nft_id,
            timestamp: tx_context::epoch(ctx),
        });

        // Clean up listing
        let Listing { id, seller: _, nft_id: _, nft_type: _, price: _, currency: _, created_at: _, expires_at: _, description: _ } = listing;
        object::delete(id);
    }

    /// Admin function to update marketplace fee
    public entry fun update_marketplace_fee(
        _admin_cap: &AdminCap,
        marketplace: &mut Marketplace,
        new_fee_percentage: u64,
        _ctx: &mut TxContext
    ) {
        assert!(new_fee_percentage <= MAX_FEE_PERCENTAGE, E_MARKETPLACE_FEE_TOO_HIGH);        marketplace.fee_percentage = new_fee_percentage;
    }
    
    /// Admin function to withdraw marketplace fees
    public entry fun withdraw_fees(
        _admin_cap: &AdminCap,
        marketplace: &mut Marketplace,
        amount: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {        assert!(balance::value(&marketplace.treasury) >= amount, E_INSUFFICIENT_PAYMENT);
        
        let withdrawal_balance = balance::split(&mut marketplace.treasury, amount);
        let withdrawal_coin = coin::from_balance(withdrawal_balance, ctx);
        transfer::public_transfer(withdrawal_coin, recipient);
    }

    /// Remove expired listing and return NFT to seller
    public entry fun remove_expired_listing<T: key + store>(
        marketplace: &mut Marketplace,
        nft_id: ID,
        ctx: &mut TxContext
    ) {
        assert!(table::contains(&marketplace.active_listings, nft_id), E_LISTING_NOT_FOUND);
        
        let listing = table::borrow(&marketplace.active_listings, nft_id);
        
        // Check if listing has expired
        if (option::is_some(&listing.expires_at)) {
            let expiry = *option::borrow(&listing.expires_at);
            assert!(tx_context::epoch(ctx) > expiry, E_ITEM_NOT_AVAILABLE);
        } else {
            // No expiry set, can't remove
            abort E_ITEM_NOT_AVAILABLE
        };
          // Remove from marketplace
        let mut listing = table::remove(&mut marketplace.active_listings, nft_id);
        
        // Return NFT to seller
        let nft: T = sui::dynamic_field::remove(&mut listing.id, b"nft");
        transfer::public_transfer(nft, listing.seller);
        
        // Update marketplace stats
        marketplace.listing_count = marketplace.listing_count - 1;

        event::emit(ListingExpired {
            listing_id: object::uid_to_inner(&listing.id),
            nft_id: listing.nft_id,
            timestamp: tx_context::epoch(ctx),
        });

        // Clean up listing
        let Listing { id, seller: _, nft_id: _, nft_type: _, price: _, currency: _, created_at: _, expires_at: _, description: _ } = listing;
        object::delete(id);
    }

    /// Update the price of an existing listing
    public entry fun update_listing_price(
        marketplace: &mut Marketplace,
        nft_id: ID,
        new_price: u64,
        ctx: &mut TxContext
    ) {
        assert!(table::contains(&marketplace.active_listings, nft_id), E_LISTING_NOT_FOUND);
        assert!(new_price > 0, E_INVALID_PRICE);
        
        let listing = table::borrow_mut(&mut marketplace.active_listings, nft_id);
        let sender = tx_context::sender(ctx);
        
        // Only seller can update their own listing
        assert!(listing.seller == sender, E_NOT_AUTHORIZED);
        
        // Check if listing hasn't expired
        if (option::is_some(&listing.expires_at)) {
            let expiry = *option::borrow(&listing.expires_at);
            assert!(tx_context::epoch(ctx) <= expiry, E_ITEM_NOT_AVAILABLE);
        };
        
        listing.price = new_price;
    }

    /// Extend the expiry time of a listing
    public entry fun extend_listing(
        marketplace: &mut Marketplace,
        nft_id: ID,
        additional_hours: u64,
        ctx: &mut TxContext
    ) {
        assert!(table::contains(&marketplace.active_listings, nft_id), E_LISTING_NOT_FOUND);
        
        let listing = table::borrow_mut(&mut marketplace.active_listings, nft_id);
        let sender = tx_context::sender(ctx);
        
        // Only seller can extend their own listing
        assert!(listing.seller == sender, E_NOT_AUTHORIZED);
        
        let current_time = tx_context::epoch(ctx);
        let additional_seconds = additional_hours * 3600;
        
        if (option::is_some(&listing.expires_at)) {
            let current_expiry = option::extract(&mut listing.expires_at);
            let new_expiry = if (current_expiry > current_time) {
                current_expiry + additional_seconds
            } else {
                current_time + additional_seconds
            };
            listing.expires_at = option::some(new_expiry);
        } else {
            // If no expiry was set, set one now
            listing.expires_at = option::some(current_time + additional_seconds);
        };
    }

    // === Query Functions ===

    /// Get marketplace info
    public fun get_marketplace_info(marketplace: &Marketplace): (u64, u64, u64) {
        (marketplace.fee_percentage, marketplace.listing_count, balance::value(&marketplace.treasury))
    }

    /// Get listing details
    public fun get_listing_info(marketplace: &Marketplace, nft_id: ID): (address, u64, String, String, u64, Option<u64>) {
        assert!(table::contains(&marketplace.active_listings, nft_id), E_LISTING_NOT_FOUND);
        let listing = table::borrow(&marketplace.active_listings, nft_id);
        (listing.seller, listing.price, listing.nft_type, listing.currency, listing.created_at, listing.expires_at)
    }    /// Check if NFT is listed
    public fun is_listed(marketplace: &Marketplace, nft_id: ID): bool {
        table::contains(&marketplace.active_listings, nft_id)
    }
    
    /// Get total fees collected
    public fun get_treasury_balance(marketplace: &Marketplace): u64 {
        balance::value(&marketplace.treasury)
    }

    /// Get purchase receipt details
    public fun get_receipt_info(receipt: &PurchaseReceipt): (address, address, ID, u64, u64, u64) {
        (receipt.buyer, receipt.seller, receipt.nft_id, receipt.price_paid, receipt.marketplace_fee, receipt.purchase_time)
    }

    // === Test Helper Functions ===
    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(ctx);
    }

    #[test_only]
    public fun test_create_admin_cap(ctx: &mut TxContext): AdminCap {
        AdminCap { id: object::new(ctx) }
    }
}
