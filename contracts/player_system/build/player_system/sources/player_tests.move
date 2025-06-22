#[test_only]
#[allow(duplicate_alias, unused_use)]
module mmo_game::player_tests {
    use std::string;
    use std::vector;
    use sui::test_scenario;
    use sui::object;
    use mmo_game::player::{Self, PlayerNFT, PlayerRegistry, AdminCap};

    // Test constants
    const ADMIN: address = @0xABCD;
    const PLAYER1: address = @0x1111;
    const PLAYER2: address = @0x2222;    #[test]
    fun test_player_system_initialization() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;

        // Initialize the player system
        test_scenario::next_tx(scenario, ADMIN);
        {
            player::test_init(test_scenario::ctx(scenario));
        };

        // Check if AdminCap and PlayerRegistry were created
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            let (total_players, active_players) = player::get_registry_info(&registry);
            assert!(total_players == 0, 0);
            assert!(active_players == 0, 1);
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(registry);
        };

        test_scenario::end(scenario_val);
    }    #[test]
    fun test_create_warrior_player() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;

        // Initialize system
        test_scenario::next_tx(scenario, ADMIN);
        {
            player::test_init(test_scenario::ctx(scenario));
        };

        // Create a warrior player
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            player::create_player(
                &mut registry,
                b"TestWarrior",
                1, // CLASS_WARRIOR
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared(registry);
        };

        // Verify player was created
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let player_nft = test_scenario::take_from_sender<PlayerNFT>(scenario);
            let registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            let (name, class, level, experience, stat_points) = player::get_player_info(&player_nft);
            assert!(name == string::utf8(b"TestWarrior"), 0);
            assert!(class == 1, 1);
            assert!(level == 1, 2);
            assert!(experience == 0, 3);
            assert!(stat_points == 0, 4);
            
            let stats = player::get_player_stats(&player_nft);
            assert!(player::get_strength(stats) == 15, 5); // Warrior base strength
            assert!(player::get_vitality(stats) == 15, 6); // Warrior base vitality
            
            let (total_players, active_players) = player::get_registry_info(&registry);
            assert!(total_players == 1, 7);
            assert!(active_players == 1, 8);
            
            test_scenario::return_to_sender(scenario, player_nft);
            test_scenario::return_shared(registry);
        };

        test_scenario::end(scenario_val);
    }    #[test]
    fun test_create_mage_player() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;

        // Initialize system
        test_scenario::next_tx(scenario, ADMIN);
        {
            player::test_init(test_scenario::ctx(scenario));
        };

        // Create a mage player
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            player::create_player(
                &mut registry,
                b"TestMage",
                2, // CLASS_MAGE
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared(registry);
        };

        // Verify mage stats
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let player_nft = test_scenario::take_from_sender<PlayerNFT>(scenario);
            
            let stats = player::get_player_stats(&player_nft);
            assert!(player::get_intelligence(stats) == 20, 0); // Mage base intelligence
            assert!(player::get_strength(stats) == 5, 1); // Mage base strength
            assert!(player::get_max_mana(stats) > player::get_max_health(stats), 2); // Mages should have more mana
            
            test_scenario::return_to_sender(scenario, player_nft);
        };

        test_scenario::end(scenario_val);
    }    #[test]
    fun test_experience_and_level_up() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;

        // Initialize system
        test_scenario::next_tx(scenario, ADMIN);
        {
            player::test_init(test_scenario::ctx(scenario));
        };

        // Create player
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            player::create_player(
                &mut registry,
                b"TestPlayer",
                1,
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared(registry);
        };

        // Grant experience to trigger level up
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut player_nft = test_scenario::take_from_address<PlayerNFT>(scenario, PLAYER1);
            
            player::grant_experience(
                &admin_cap,
                &mut player_nft,
                2000, // Enough for level 2
                b"quest_completion",
                test_scenario::ctx(scenario)
            );
            
            let (_, _, level, experience, stat_points) = player::get_player_info(&player_nft);
            assert!(level == 2, 0);
            assert!(experience == 2000, 1);
            assert!(stat_points == 5, 2); // Should get 5 stat points per level
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_to_address(PLAYER1, player_nft);
        };

        test_scenario::end(scenario_val);
    }    #[test]
    fun test_stat_allocation() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;

        // Initialize and create player
        test_scenario::next_tx(scenario, ADMIN);
        {
            player::test_init(test_scenario::ctx(scenario));
        };

        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            player::create_player(
                &mut registry,
                b"TestPlayer",
                1,
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared(registry);
        };

        // Grant experience for level up
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut player_nft = test_scenario::take_from_address<PlayerNFT>(scenario, PLAYER1);
            
            player::grant_experience(
                &admin_cap,
                &mut player_nft,
                2000,
                b"test",
                test_scenario::ctx(scenario)
            );
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_to_address(PLAYER1, player_nft);
        };

        // Allocate stats
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut player_nft = test_scenario::take_from_sender<PlayerNFT>(scenario);
            
            let stats_before = player::get_player_stats(&player_nft);
            let strength_before = player::get_strength(stats_before);
            let agility_before = player::get_agility(stats_before);
            
            player::allocate_stats(
                &mut player_nft,
                3, // strength
                2, // agility
                0, // intelligence
                0, // vitality
                0, // luck
                test_scenario::ctx(scenario)
            );
            
            let (_, _, _, _, remaining_points) = player::get_player_info(&player_nft);
            assert!(remaining_points == 0, 0); // All 5 points should be used
            
            let stats_after = player::get_player_stats(&player_nft);
            assert!(player::get_strength(stats_after) == strength_before + 3, 1);
            assert!(player::get_agility(stats_after) == agility_before + 2, 2);
            
            test_scenario::return_to_sender(scenario, player_nft);
        };

        test_scenario::end(scenario_val);
    }    #[test]
    fun test_multiple_players_different_classes() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;

        test_scenario::next_tx(scenario, ADMIN);
        {
            player::test_init(test_scenario::ctx(scenario));
        };

        // Create warrior
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            player::create_player(
                &mut registry,
                b"Warrior1",
                1,
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared(registry);
        };

        // Create mage
        test_scenario::next_tx(scenario, PLAYER2);
        {
            let mut registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            player::create_player(
                &mut registry,
                b"Mage1",
                2,
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared(registry);
        };

        // Verify registry updated
        test_scenario::next_tx(scenario, ADMIN);
        {
            let registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            let (total_players, active_players) = player::get_registry_info(&registry);
            assert!(total_players == 2, 0);
            assert!(active_players == 2, 1);
            
            // Check name availability
            assert!(!player::is_name_available(&registry, string::utf8(b"Warrior1")), 2);
            assert!(!player::is_name_available(&registry, string::utf8(b"Mage1")), 3);
            assert!(player::is_name_available(&registry, string::utf8(b"Available")), 4);
            
            test_scenario::return_shared(registry);
        };

        test_scenario::end(scenario_val);
    }    #[test]
    fun test_achievement_system() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;

        // Initialize and create player
        test_scenario::next_tx(scenario, ADMIN);
        {
            player::test_init(test_scenario::ctx(scenario));
        };

        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            player::create_player(
                &mut registry,
                b"TestPlayer",
                1,
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared(registry);
        };

        // Unlock achievement
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut player_nft = test_scenario::take_from_address<PlayerNFT>(scenario, PLAYER1);
            
            player::unlock_achievement(
                &admin_cap,
                &mut player_nft,
                101, // First achievement ID
                test_scenario::ctx(scenario)
            );
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_to_address(PLAYER1, player_nft);
        };

        test_scenario::end(scenario_val);
    }#[test]
    #[expected_failure(abort_code = player::E_NAME_TOO_SHORT)]
    fun test_name_too_short() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;

        test_scenario::next_tx(scenario, ADMIN);
        {
            player::test_init(test_scenario::ctx(scenario));
        };

        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            player::create_player(
                &mut registry,
                b"AB", // Too short (minimum is 3)
                1,
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared(registry);
        };

        test_scenario::end(scenario_val);
    }

    #[test]
    #[expected_failure(abort_code = player::E_INVALID_CLASS)]
    fun test_invalid_class() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;

        test_scenario::next_tx(scenario, ADMIN);
        {
            player::test_init(test_scenario::ctx(scenario));
        };

        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut registry = test_scenario::take_shared<PlayerRegistry>(scenario);
            
            player::create_player(
                &mut registry,
                b"TestPlayer",
                99, // Invalid class
                test_scenario::ctx(scenario)
            );

            test_scenario::return_shared(registry);
        };

        test_scenario::end(scenario_val);
    }    #[test]
    fun test_class_base_stats() {
        // Test warrior stats
        let warrior_stats = player::test_get_class_base_stats(1);
        assert!(player::get_strength(&warrior_stats) == 15, 0);
        assert!(player::get_vitality(&warrior_stats) == 15, 1);
        
        // Test mage stats
        let mage_stats = player::test_get_class_base_stats(2);
        assert!(player::get_intelligence(&mage_stats) == 20, 2);
        assert!(player::get_strength(&mage_stats) == 5, 3);
        
        // Test archer stats
        let archer_stats = player::test_get_class_base_stats(3);
        assert!(player::get_agility(&archer_stats) == 18, 4);
        
        // Test rogue stats
        let rogue_stats = player::test_get_class_base_stats(4);
        assert!(player::get_agility(&rogue_stats) == 15, 5);
        assert!(player::get_luck(&rogue_stats) == 7, 6);
    }
}
