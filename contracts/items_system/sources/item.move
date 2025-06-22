// Sui Move Items System Contract
// Defines item NFTs, equipment, consumables, etc.

module mmo_game::item {    use sui::object::{Self, UID, ID};
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};
    use std::string::{Self, String};
    use std::vector;
    use std::option;

    // Error constants
    const E_NOT_CONSUMABLE: u64 = 0;
    const E_NO_CHARGES: u64 = 1;
    const E_INVALID_ITEM_TYPE: u64 = 2;
    const E_NOT_EQUIPMENT: u64 = 3;

    // Item type constants
    const ITEM_TYPE_WEAPON: u8 = 1;
    const ITEM_TYPE_ARMOR: u8 = 2;
    const ITEM_TYPE_CONSUMABLE: u8 = 3;
    const ITEM_TYPE_ACCESSORY: u8 = 4;

    // Effect type constants for consumables
    const EFFECT_HEALING: u8 = 1;
    const EFFECT_MANA_RESTORE: u8 = 2;
    const EFFECT_BUFF_ATTACK: u8 = 3;
    const EFFECT_BUFF_DEFENSE: u8 = 4;

    // Generic Item NFT structure
    public struct ItemNFT has key, store {
        id: UID,
        item_type_id: u64, // An ID referring to an off-chain or on-chain definition of the item type
        name: String,
        description: String,
        item_type: u8, // ITEM_TYPE_* constants
        rarity: u8, // 1-5 (common to legendary)
        level: u64,
        // For equipment:
        attack_bonus: u64,
        defense_bonus: u64,
        // For consumables:
        charges: u8,
        effect_type: u8, // EFFECT_* constants
        effect_value: u64, // amount of healing, mana, etc.
    }

    // Equipment slot structure for organizing equipped items
    public struct EquipmentSlot has store {
        slot_type: u8, // 1=weapon, 2=armor, 3=accessory
        item_id: option::Option<ID>,
    }    // --- Public Functions ---

    // Function to mint a new item NFT
    public fun mint_item(
        item_type_id: u64,
        name: vector<u8>,
        description: vector<u8>,
        item_type: u8,
        rarity: u8,
        level: u64,
        attack_bonus: u64,
        defense_bonus: u64,
        charges: u8,
        effect_type: u8,
        effect_value: u64,
        ctx: &mut TxContext
    ): ItemNFT {
        assert!(item_type >= 1 && item_type <= 4, E_INVALID_ITEM_TYPE);
        
        ItemNFT {
            id: object::new(ctx),
            item_type_id,
            name: string::utf8(name),
            description: string::utf8(description),
            item_type,
            rarity,
            level,
            attack_bonus,
            defense_bonus,
            charges,
            effect_type,
            effect_value,
        }
    }

    // Function to mint and transfer a new item NFT
    public entry fun mint_and_transfer_item(
        item_type_id: u64,
        name: vector<u8>,
        description: vector<u8>,
        item_type: u8,
        rarity: u8,
        level: u64,
        attack_bonus: u64,
        defense_bonus: u64,
        charges: u8,
        effect_type: u8,
        effect_value: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {
        let item = mint_item(
            item_type_id,
            name,
            description,
            item_type,
            rarity,
            level,
            attack_bonus,
            defense_bonus,
            charges,
            effect_type,
            effect_value,
            ctx
        );
        transfer::transfer(item, recipient);
    }

    // Function to use a consumable item
    public entry fun use_consumable(item_nft: &mut ItemNFT) {
        assert!(item_nft.item_type == ITEM_TYPE_CONSUMABLE, E_NOT_CONSUMABLE);
        assert!(item_nft.charges > 0, E_NO_CHARGES);
        
        item_nft.charges = item_nft.charges - 1;
        
        // If no charges left, the item becomes unusable but not deleted
        // (Player can choose to delete it manually or keep as collectible)
    }

    // Function to upgrade an item's level
    public entry fun upgrade_item(item_nft: &mut ItemNFT, levels: u64) {
        item_nft.level = item_nft.level + levels;
        
        // Increase bonuses based on item type and levels gained
        if (item_nft.item_type == ITEM_TYPE_WEAPON || item_nft.item_type == ITEM_TYPE_ACCESSORY) {
            item_nft.attack_bonus = item_nft.attack_bonus + (levels * 2);
        };
        
        if (item_nft.item_type == ITEM_TYPE_ARMOR || item_nft.item_type == ITEM_TYPE_ACCESSORY) {
            item_nft.defense_bonus = item_nft.defense_bonus + (levels * 2);
        };
    }

    // Function to repair a consumable item (restore charges)
    public entry fun repair_consumable(item_nft: &mut ItemNFT, charges_to_add: u8) {
        assert!(item_nft.item_type == ITEM_TYPE_CONSUMABLE, E_NOT_CONSUMABLE);
        item_nft.charges = item_nft.charges + charges_to_add;
    }

    // --- Helper Functions for Common Item Creation ---

    // Create a weapon with specified stats
    public entry fun create_weapon(
        item_type_id: u64,
        name: vector<u8>,
        description: vector<u8>,
        rarity: u8,
        level: u64,
        attack_bonus: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {
        mint_and_transfer_item(
            item_type_id,
            name,
            description,
            ITEM_TYPE_WEAPON,
            rarity,
            level,
            attack_bonus,
            0, // defense_bonus
            0, // charges
            0, // effect_type
            0, // effect_value
            recipient,
            ctx
        );
    }

    // Create armor with specified stats
    public entry fun create_armor(
        item_type_id: u64,
        name: vector<u8>,
        description: vector<u8>,
        rarity: u8,
        level: u64,
        defense_bonus: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {
        mint_and_transfer_item(
            item_type_id,
            name,
            description,
            ITEM_TYPE_ARMOR,
            rarity,
            level,
            0, // attack_bonus
            defense_bonus,
            0, // charges
            0, // effect_type
            0, // effect_value
            recipient,
            ctx
        );
    }

    // Create a consumable item
    public entry fun create_consumable(
        item_type_id: u64,
        name: vector<u8>,
        description: vector<u8>,
        rarity: u8,
        charges: u8,
        effect_type: u8,
        effect_value: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {
        mint_and_transfer_item(
            item_type_id,
            name,
            description,
            ITEM_TYPE_CONSUMABLE,
            rarity,
            1, // level (consumables start at level 1)
            0, // attack_bonus
            0, // defense_bonus
            charges,
            effect_type,
            effect_value,
            recipient,
            ctx
        );
    }

    // Create an accessory with balanced stats
    public entry fun create_accessory(
        item_type_id: u64,
        name: vector<u8>,
        description: vector<u8>,
        rarity: u8,
        level: u64,
        attack_bonus: u64,
        defense_bonus: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {
        mint_and_transfer_item(
            item_type_id,
            name,
            description,
            ITEM_TYPE_ACCESSORY,
            rarity,
            level,
            attack_bonus,
            defense_bonus,
            0, // charges
            0, // effect_type
            0, // effect_value
            recipient,
            ctx
        );
    }

    // --- Utility Functions ---

    // Calculate total combat power of an item
    public fun get_combat_power(item: &ItemNFT): u64 {
        let base_power = item.attack_bonus + item.defense_bonus;
        let level_multiplier = item.level;
        let rarity_bonus = (item.rarity as u64) * 10;
        
        base_power * level_multiplier + rarity_bonus
    }

    // Check if item can be upgraded (based on rarity and current level)
    public fun can_upgrade(item: &ItemNFT): bool {
        let max_level = match (item.rarity) {
            1 => 10, // Common: max level 10
            2 => 25, // Uncommon: max level 25
            3 => 50, // Rare: max level 50
            4 => 75, // Epic: max level 75
            5 => 100, // Legendary: max level 100
            _ => 1,
        };
        item.level < max_level
    }

    // Get upgrade cost based on current level and rarity
    public fun get_upgrade_cost(item: &ItemNFT): u64 {
        let base_cost = 100;
        let level_cost = item.level * 50;
        let rarity_multiplier = (item.rarity as u64);
        
        base_cost + level_cost * rarity_multiplier
    }

    // Check if two items are the same type and can be combined
    public fun can_combine_items(item1: &ItemNFT, item2: &ItemNFT): bool {
        item1.item_type_id == item2.item_type_id &&
        item1.item_type == item2.item_type &&
        item1.rarity == item2.rarity
    }

    // Get item description with stats
    public fun get_full_description(item: &ItemNFT): String {
        let mut desc = item.description;
        
        if (is_equipment(item)) {
            string::append_utf8(&mut desc, b" | ATK: ");
            // Note: In a real implementation, you'd need a number-to-string conversion
            string::append_utf8(&mut desc, b" DEF: ");
        };
        
        if (is_consumable(item)) {
            string::append_utf8(&mut desc, b" | Charges: ");
            string::append_utf8(&mut desc, b" Effect: ");
        };
        
        desc
    }

    // --- Getter Functions ---

    public fun get_item_id(item: &ItemNFT): ID {
        object::uid_to_inner(&item.id)
    }

    public fun get_item_type_id(item: &ItemNFT): u64 {
        item.item_type_id
    }

    public fun get_name(item: &ItemNFT): String {
        item.name
    }

    public fun get_description(item: &ItemNFT): String {
        item.description
    }

    public fun get_item_type(item: &ItemNFT): u8 {
        item.item_type
    }

    public fun get_rarity(item: &ItemNFT): u8 {
        item.rarity
    }

    public fun get_level(item: &ItemNFT): u64 {
        item.level
    }

    public fun get_attack_bonus(item: &ItemNFT): u64 {
        item.attack_bonus
    }

    public fun get_defense_bonus(item: &ItemNFT): u64 {
        item.defense_bonus
    }

    public fun get_charges(item: &ItemNFT): u8 {
        item.charges
    }

    public fun get_effect_type(item: &ItemNFT): u8 {
        item.effect_type
    }

    public fun get_effect_value(item: &ItemNFT): u64 {
        item.effect_value
    }

    public fun is_equipment(item: &ItemNFT): bool {
        item.item_type == ITEM_TYPE_WEAPON || 
        item.item_type == ITEM_TYPE_ARMOR || 
        item.item_type == ITEM_TYPE_ACCESSORY
    }

    public fun is_consumable(item: &ItemNFT): bool {
        item.item_type == ITEM_TYPE_CONSUMABLE
    }

    public fun has_charges(item: &ItemNFT): bool {
        item.charges > 0
    }    // --- Test Functions ---
    #[test_only]
    use sui::test_scenario;

    #[test]
    public fun test_mint_item() {
        let admin = @0xAD;
        let player = @0x123;
        
        let scenario_val = test_scenario::begin(admin);
        let scenario = &mut scenario_val;
        
        // Test minting a weapon
        test_scenario::next_tx(scenario, admin);
        {
            let ctx = test_scenario::ctx(scenario);
            mint_and_transfer_item(
                1, // item_type_id
                b"Excalibur", // name
                b"A legendary sword", // description
                ITEM_TYPE_WEAPON, // item_type
                5, // rarity (legendary)
                1, // level
                100, // attack_bonus
                0, // defense_bonus
                0, // charges (not applicable for weapon)
                0, // effect_type (not applicable)
                0, // effect_value (not applicable)
                player,
                ctx
            );
        };
        
        // Test the weapon was created correctly
        test_scenario::next_tx(scenario, player);
        {
            let item = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            assert!(get_name(&item) == string::utf8(b"Excalibur"), 0);
            assert!(get_item_type(&item) == ITEM_TYPE_WEAPON, 1);
            assert!(get_attack_bonus(&item) == 100, 2);
            assert!(is_equipment(&item), 3);
            assert!(!is_consumable(&item), 4);
            
            test_scenario::return_to_sender(scenario, item);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    public fun test_consumable_item() {
        let admin = @0xAD;
        let player = @0x123;
        
        let scenario_val = test_scenario::begin(admin);
        let scenario = &mut scenario_val;
        
        // Test minting a consumable
        test_scenario::next_tx(scenario, admin);
        {
            let ctx = test_scenario::ctx(scenario);
            mint_and_transfer_item(
                2, // item_type_id
                b"Health Potion", // name
                b"Restores 50 HP", // description
                ITEM_TYPE_CONSUMABLE, // item_type
                1, // rarity (common)
                1, // level
                0, // attack_bonus (not applicable)
                0, // defense_bonus (not applicable)
                5, // charges
                EFFECT_HEALING, // effect_type
                50, // effect_value
                player,
                ctx
            );
        };
        
        // Test using the consumable
        test_scenario::next_tx(scenario, player);
        {
            let item = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            assert!(get_charges(&item) == 5, 0);
            assert!(is_consumable(&item), 1);
            assert!(has_charges(&item), 2);
            
            // Use the item
            use_consumable(&mut item);
            assert!(get_charges(&item) == 4, 3);
            
            test_scenario::return_to_sender(scenario, item);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    public fun test_item_upgrade() {
        let admin = @0xAD;
        let player = @0x123;
        
        let scenario_val = test_scenario::begin(admin);
        let scenario = &mut scenario_val;
        
        // Test minting and upgrading a weapon
        test_scenario::next_tx(scenario, admin);
        {
            let ctx = test_scenario::ctx(scenario);
            mint_and_transfer_item(
                3, // item_type_id
                b"Iron Sword", // name
                b"A basic iron sword", // description
                ITEM_TYPE_WEAPON, // item_type
                1, // rarity (common)
                1, // level
                10, // attack_bonus
                0, // defense_bonus
                0, // charges
                0, // effect_type
                0, // effect_value
                player,
                ctx
            );
        };
        
        // Test upgrading the weapon
        test_scenario::next_tx(scenario, player);
        {
            let item = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            assert!(get_level(&item) == 1, 0);
            assert!(get_attack_bonus(&item) == 10, 1);
            
            // Upgrade by 5 levels
            upgrade_item(&mut item, 5);
            
            assert!(get_level(&item) == 6, 2);
            assert!(get_attack_bonus(&item) == 20, 3); // 10 + (5 * 2)
            
            test_scenario::return_to_sender(scenario, item);
        };
        
        test_scenario::end(scenario_val);
    }
}
