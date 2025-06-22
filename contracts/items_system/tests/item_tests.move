#[test_only]
module mmo_game::item_tests {
    use mmo_game::item::{Self, ItemNFT};
    use sui::test_scenario;
    use std::string;

    const ADMIN: address = @0xAD;
    const PLAYER1: address = @0x123;
    const PLAYER2: address = @0x456;

    #[test]
    fun test_create_weapon() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            item::create_weapon(
                1, // item_type_id
                b"Steel Sword", // name
                b"A sharp steel sword", // description
                2, // rarity (uncommon)
                5, // level
                50, // attack_bonus
                PLAYER1,
                ctx
            );
        };
        
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let weapon = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            assert!(item::get_name(&weapon) == string::utf8(b"Steel Sword"), 0);
            assert!(item::get_item_type(&weapon) == 1, 1); // ITEM_TYPE_WEAPON
            assert!(item::get_attack_bonus(&weapon) == 50, 2);
            assert!(item::get_defense_bonus(&weapon) == 0, 3);
            assert!(item::is_equipment(&weapon), 4);
            assert!(!item::is_consumable(&weapon), 5);
            
            test_scenario::return_to_sender(scenario, weapon);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_create_armor() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            item::create_armor(
                2, // item_type_id
                b"Iron Chestplate", // name
                b"Heavy iron armor", // description
                3, // rarity (rare)
                3, // level
                75, // defense_bonus
                PLAYER1,
                ctx
            );
        };
        
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let armor = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            assert!(item::get_name(&armor) == string::utf8(b"Iron Chestplate"), 0);
            assert!(item::get_item_type(&armor) == 2, 1); // ITEM_TYPE_ARMOR
            assert!(item::get_attack_bonus(&armor) == 0, 2);
            assert!(item::get_defense_bonus(&armor) == 75, 3);
            assert!(item::is_equipment(&armor), 4);
            
            test_scenario::return_to_sender(scenario, armor);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_create_consumable() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            item::create_consumable(
                3, // item_type_id
                b"Mana Potion", // name
                b"Restores mana", // description
                1, // rarity (common)
                10, // charges
                2, // effect_type (EFFECT_MANA_RESTORE)
                30, // effect_value
                PLAYER1,
                ctx
            );
        };
        
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let potion = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            assert!(item::get_name(&potion) == string::utf8(b"Mana Potion"), 0);
            assert!(item::get_item_type(&potion) == 3, 1); // ITEM_TYPE_CONSUMABLE
            assert!(item::get_charges(&potion) == 10, 2);
            assert!(item::get_effect_type(&potion) == 2, 3);
            assert!(item::get_effect_value(&potion) == 30, 4);
            assert!(item::is_consumable(&potion), 5);
            assert!(item::has_charges(&potion), 6);
            
            test_scenario::return_to_sender(scenario, potion);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_use_consumable() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            item::create_consumable(
                4, // item_type_id
                b"Health Potion", // name
                b"Restores health", // description
                1, // rarity (common)
                3, // charges
                1, // effect_type (EFFECT_HEALING)
                50, // effect_value
                PLAYER1,
                ctx
            );
        };
        
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let potion = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            assert!(item::get_charges(&potion) == 3, 0);
            
            // Use the potion once
            item::use_consumable(&mut potion);
            assert!(item::get_charges(&potion) == 2, 1);
            
            // Use it again
            item::use_consumable(&mut potion);
            assert!(item::get_charges(&potion) == 1, 2);
            
            // Use the last charge
            item::use_consumable(&mut potion);
            assert!(item::get_charges(&potion) == 0, 3);
            assert!(!item::has_charges(&potion), 4);
            
            test_scenario::return_to_sender(scenario, potion);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_upgrade_item() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            item::create_weapon(
                5, // item_type_id
                b"Basic Sword", // name
                b"A simple sword", // description
                1, // rarity (common)
                1, // level
                10, // attack_bonus
                PLAYER1,
                ctx
            );
        };
        
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let sword = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            assert!(item::get_level(&sword) == 1, 0);
            assert!(item::get_attack_bonus(&sword) == 10, 1);
            
            // Upgrade by 3 levels
            item::upgrade_item(&mut sword, 3);
            
            assert!(item::get_level(&sword) == 4, 2);
            assert!(item::get_attack_bonus(&sword) == 16, 3); // 10 + (3 * 2)
            
            test_scenario::return_to_sender(scenario, sword);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_repair_consumable() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            item::create_consumable(
                6, // item_type_id
                b"Magic Scroll", // name
                b"Casts a spell", // description
                2, // rarity (uncommon)
                1, // charges
                3, // effect_type (EFFECT_BUFF_ATTACK)
                20, // effect_value
                PLAYER1,
                ctx
            );
        };
        
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let scroll = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            assert!(item::get_charges(&scroll) == 1, 0);
            
            // Use the scroll
            item::use_consumable(&mut scroll);
            assert!(item::get_charges(&scroll) == 0, 1);
            
            // Repair it (add 5 charges)
            item::repair_consumable(&mut scroll, 5);
            assert!(item::get_charges(&scroll) == 5, 2);
            assert!(item::has_charges(&scroll), 3);
            
            test_scenario::return_to_sender(scenario, scroll);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_combat_power_calculation() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            item::create_accessory(
                7, // item_type_id
                b"Magic Ring", // name
                b"A powerful ring", // description
                4, // rarity (epic)
                5, // level
                20, // attack_bonus
                15, // defense_bonus
                PLAYER1,
                ctx
            );
        };
        
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let ring = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            let combat_power = item::get_combat_power(&ring);
            // Expected: (20 + 15) * 5 + 4 * 10 = 175 + 40 = 215
            assert!(combat_power == 215, 0);
            
            assert!(item::can_upgrade(&ring), 1); // Epic items can upgrade to level 75
            
            let upgrade_cost = item::get_upgrade_cost(&ring);
            // Expected: 100 + 5 * 50 * 4 = 100 + 1000 = 1100
            assert!(upgrade_cost == 1100, 2);
            
            test_scenario::return_to_sender(scenario, ring);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test] 
    fun test_item_compatibility() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            
            // Create two similar weapons
            item::create_weapon(
                8, // item_type_id
                b"Iron Sword 1", // name
                b"First iron sword", // description
                2, // rarity (uncommon)
                3, // level
                25, // attack_bonus
                PLAYER1,
                ctx
            );
        };
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            
            item::create_weapon(
                8, // same item_type_id
                b"Iron Sword 2", // name
                b"Second iron sword", // description
                2, // same rarity
                5, // different level
                30, // different attack_bonus
                PLAYER2,
                ctx
            );
        };
        
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let sword1 = test_scenario::take_from_sender<ItemNFT>(scenario);
            
            test_scenario::next_tx(scenario, PLAYER2);
            {
                let sword2 = test_scenario::take_from_sender<ItemNFT>(scenario);
                
                // These items should be combinable (same type_id, type, and rarity)
                assert!(item::can_combine_items(&sword1, &sword2), 0);
                
                test_scenario::return_to_sender(scenario, sword2);
            };
            
            test_scenario::return_to_sender(scenario, sword1);
        };
        
        test_scenario::end(scenario_val);
    }
}
