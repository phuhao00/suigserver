#[test_only]
module mmo_game::marketplace_tests {
    use std::option;
    use std::string;
    use sui::test_scenario::{Self as test, next_tx, ctx};
    use sui::coin::{Self, Coin};
    use sui::sui::SUI;
    use sui::object::{Self, UID, ID};
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};
    
    use mmo_game::marketplace::{Self, Marketplace, AdminCap, PurchaseReceipt};

    // Test NFT struct for testing purposes
    public struct TestNFT has key, store {
        id: UID,
        name: string::String,
        description: string::String,
    }

    const ADMIN: address = @0xAD;
    const SELLER: address = @0x1;
    const BUYER: address = @0x2;
    const OTHER_USER: address = @0x3;

    // Helper function to create a test NFT
    fun create_test_nft(ctx: &mut TxContext): TestNFT {
        TestNFT {
            id: object::new(ctx),
            name: string::utf8(b"Test NFT"),
            description: string::utf8(b"A test NFT for marketplace testing"),
        }
    }

    #[test]
    fun test_marketplace_initialization() {
        let mut scenario = test::begin(ADMIN);
        {
            marketplace::test_init(ctx(&mut scenario));
        };
        
        next_tx(&mut scenario, ADMIN);
        {
            let marketplace = test::take_shared<Marketplace>(&scenario);
            let admin_cap = test::take_from_sender<AdminCap>(&scenario);
            
            // Check initial state
            let (fee_percentage, listing_count, treasury_balance) = marketplace::get_marketplace_info(&marketplace);
            assert!(fee_percentage == 250, 0); // 2.5%
            assert!(listing_count == 0, 1);
            assert!(treasury_balance == 0, 2);
            
            test::return_shared(marketplace);
            test::return_to_sender(&scenario, admin_cap);
        };
        
        test::end(scenario);
    }

    #[test]
    fun test_list_nft_successfully() {
        let mut scenario = test::begin(ADMIN);
        {
            marketplace::test_init(ctx(&mut scenario));
        };
        
        next_tx(&mut scenario, SELLER);
        {
            let mut marketplace = test::take_shared<Marketplace>(&scenario);
            let test_nft = create_test_nft(ctx(&mut scenario));
            let nft_id = object::id(&test_nft);
            
            marketplace::list_nft(
                &mut marketplace,
                test_nft,
                b"test_nft",
                1000, // 1000 MIST
                b"SUI",
                b"A beautiful test NFT",
                option::some(24), // 24 hours
                ctx(&mut scenario)
            );
            
            // Check listing was created
            assert!(marketplace::is_listed(&marketplace, nft_id), 0);
            let (fee_percentage, listing_count, treasury_balance) = marketplace::get_marketplace_info(&marketplace);
            assert!(listing_count == 1, 1);
            
            test::return_shared(marketplace);
        };
        
        test::end(scenario);
    }

    #[test]
    fun test_purchase_nft_successfully() {
        let mut scenario = test::begin(ADMIN);
        {
            marketplace::test_init(ctx(&mut scenario));
        };
        
        let mut nft_id: ID;
        
        // Seller lists NFT
        next_tx(&mut scenario, SELLER);
        {
            let mut marketplace = test::take_shared<Marketplace>(&scenario);
            let test_nft = create_test_nft(ctx(&mut scenario));
            nft_id = object::id(&test_nft);
            
            marketplace::list_nft(
                &mut marketplace,
                test_nft,
                b"test_nft",
                1000,
                b"SUI",
                b"A beautiful test NFT",
                option::none(),
                ctx(&mut scenario)
            );
            
            test::return_shared(marketplace);
        };
        
        // Buyer purchases NFT
        next_tx(&mut scenario, BUYER);
        {
            let mut marketplace = test::take_shared<Marketplace>(&scenario);
            let payment = coin::mint_for_testing<SUI>(1000, ctx(&mut scenario));
            
            marketplace::purchase_nft_entry<TestNFT>(
                &mut marketplace,
                nft_id,
                payment,
                ctx(&mut scenario)
            );
            
            // Check listing was removed
            assert!(!marketplace::is_listed(&marketplace, nft_id), 0);
            let (fee_percentage, listing_count, treasury_balance) = marketplace::get_marketplace_info(&marketplace);
            assert!(listing_count == 0, 1);
            assert!(treasury_balance == 25, 2); // 2.5% of 1000
            
            test::return_shared(marketplace);
        };
        
        // Check buyer received NFT and receipt
        next_tx(&mut scenario, BUYER);
        {
            let nft = test::take_from_sender<TestNFT>(&scenario);
            let receipt = test::take_from_sender<PurchaseReceipt>(&scenario);
            
            assert!(object::id(&nft) == nft_id, 0);
            let (buyer, seller, receipt_nft_id, price_paid, marketplace_fee, purchase_time) = 
                marketplace::get_receipt_info(&receipt);
            assert!(buyer == BUYER, 1);
            assert!(seller == SELLER, 2);
            assert!(receipt_nft_id == nft_id, 3);
            assert!(price_paid == 1000, 4);
            assert!(marketplace_fee == 25, 5);
            
            test::return_to_sender(&scenario, nft);
            test::return_to_sender(&scenario, receipt);
        };
        
        test::end(scenario);
    }

    #[test]
    #[expected_failure(abort_code = mmo_game::marketplace::E_INVALID_PRICE)]
    fun test_list_nft_with_zero_price_fails() {
        let mut scenario = test::begin(ADMIN);
        {
            marketplace::test_init(ctx(&mut scenario));
        };
        
        next_tx(&mut scenario, SELLER);
        {
            let mut marketplace = test::take_shared<Marketplace>(&scenario);
            let test_nft = create_test_nft(ctx(&mut scenario));
            
            marketplace::list_nft(
                &mut marketplace,
                test_nft,
                b"test_nft",
                0, // Invalid price
                b"SUI",
                b"A test NFT",
                option::none(),
                ctx(&mut scenario)
            );
            
            test::return_shared(marketplace);
        };
        
        test::end(scenario);
    }
}
