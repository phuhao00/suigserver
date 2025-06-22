#[test_only]
module mmo_game::combat_tests {
    use std::vector;
    use std::option;
    use std::string;
    use sui::test_scenario::{Self, Scenario};
    use sui::coin::{Self, mint_for_testing};
    use sui::sui::SUI;
    use mmo_game::combat::{Self, AdminCap, CombatSession, CombatRegistry};

    // Test addresses
    const ADMIN: address = @0xA;
    const PLAYER1: address = @0xB;
    const PLAYER2: address = @0xC;

    #[test]
    fun test_init_combat_system() {
        let mut scenario = test_scenario::begin(ADMIN);
        
        // Initialize the combat system
        combat::test_init(test_scenario::ctx(&mut scenario));
        
        test_scenario::next_tx(&mut scenario, ADMIN);
        
        // Check if AdminCap was created
        assert!(test_scenario::has_most_recent_for_sender<AdminCap>(&scenario), 0);
        
        // Check if CombatRegistry was shared
        assert!(test_scenario::has_most_recent_shared<CombatRegistry>(), 1);
        
        test_scenario::end(scenario);
    }

    #[test]
    fun test_initiate_combat() {
        let mut scenario = test_scenario::begin(ADMIN);
        
        // Initialize the combat system
        combat::test_init(test_scenario::ctx(&mut scenario));
        
        test_scenario::next_tx(&mut scenario, ADMIN);
        
        let admin_cap = test_scenario::take_from_sender<AdminCap>(&scenario);
        let participants = vector[PLAYER1, PLAYER2];
        let reward_coin = mint_for_testing<SUI>(1000, test_scenario::ctx(&mut scenario));
        
        // Initiate a combat session
        combat::initiate_combat(
            &admin_cap,
            1, // combat_id
            1, // combat_type (PVP)
            participants,
            reward_coin,
            b"test combat",
            test_scenario::ctx(&mut scenario)
        );
        
        test_scenario::next_tx(&mut scenario, ADMIN);
        
        // Check if CombatSession was created and shared
        assert!(test_scenario::has_most_recent_shared<CombatSession>(), 2);
        
        test_scenario::return_to_sender(&scenario, admin_cap);
        test_scenario::end(scenario);
    }

    #[test]
    fun test_start_and_finish_combat() {
        let mut scenario = test_scenario::begin(ADMIN);
        
        // Initialize the combat system
        combat::test_init(test_scenario::ctx(&mut scenario));
        
        test_scenario::next_tx(&mut scenario, ADMIN);
        
        let admin_cap = test_scenario::take_from_sender<AdminCap>(&scenario);
        let participants = vector[PLAYER1, PLAYER2];
        let reward_coin = mint_for_testing<SUI>(1000, test_scenario::ctx(&mut scenario));
        
        // Initiate a combat session
        combat::initiate_combat(
            &admin_cap,
            1,
            1,
            participants,
            reward_coin,
            b"test combat",
            test_scenario::ctx(&mut scenario)
        );
        
        test_scenario::next_tx(&mut scenario, ADMIN);
        
        let mut combat_session = test_scenario::take_shared<CombatSession>(&scenario);
        
        // Start the combat
        combat::start_combat(
            &admin_cap,
            &mut combat_session,
            test_scenario::ctx(&mut scenario)
        );
        
        // Record some damage
        combat::record_damage(
            &admin_cap,
            &mut combat_session,
            PLAYER1,
            PLAYER2,
            100,
            b"sword_attack",
            test_scenario::ctx(&mut scenario)
        );
        
        // Record some healing
        combat::record_healing(
            &admin_cap,
            &mut combat_session,
            PLAYER2,
            PLAYER2,
            50,
            test_scenario::ctx(&mut scenario)
        );
        
        // Finalize the combat
        let reward_distributions = vector::empty();
        combat::finalize_combat(
            &admin_cap,
            &mut combat_session,
            option::some(PLAYER1),
            option::some(PLAYER2),
            reward_distributions,
            test_scenario::ctx(&mut scenario)
        );
        
        // Check combat info
        let (combat_id, combat_type, status, participants_result) = combat::get_combat_info(&combat_session);
        assert!(combat_id == 1, 3);
        assert!(combat_type == 1, 4);
        assert!(status == 2, 5); // STATUS_FINISHED
        assert!(vector::length(&participants_result) == 2, 6);
        
        test_scenario::return_shared(combat_session);
        test_scenario::return_to_sender(&scenario, admin_cap);
        test_scenario::end(scenario);
    }

    #[test]
    fun test_cancel_combat() {
        let mut scenario = test_scenario::begin(ADMIN);
        
        // Initialize the combat system
        combat::test_init(test_scenario::ctx(&mut scenario));
        
        test_scenario::next_tx(&mut scenario, ADMIN);
        
        let admin_cap = test_scenario::take_from_sender<AdminCap>(&scenario);
        let participants = vector[PLAYER1, PLAYER2];
        let reward_coin = mint_for_testing<SUI>(1000, test_scenario::ctx(&mut scenario));
        
        // Initiate a combat session
        combat::initiate_combat(
            &admin_cap,
            2,
            1,
            participants,
            reward_coin,
            b"test combat",
            test_scenario::ctx(&mut scenario)
        );
        
        test_scenario::next_tx(&mut scenario, ADMIN);
        
        let mut combat_session = test_scenario::take_shared<CombatSession>(&scenario);
        
        // Cancel the combat
        combat::cancel_combat(
            &admin_cap,
            &mut combat_session,
            test_scenario::ctx(&mut scenario)
        );
        
        // Check that combat was cancelled
        let (_, _, status, _) = combat::get_combat_info(&combat_session);
        assert!(status == 3, 7); // STATUS_CANCELLED
        
        test_scenario::return_shared(combat_session);
        test_scenario::return_to_sender(&scenario, admin_cap);
        test_scenario::end(scenario);
    }

    #[test]
    fun test_detailed_combat_session() {
        let mut scenario = test_scenario::begin(ADMIN);
        
        // Initialize the combat system
        combat::test_init(test_scenario::ctx(&mut scenario));
        
        test_scenario::next_tx(&mut scenario, ADMIN);
        
        let admin_cap = test_scenario::take_from_sender<AdminCap>(&scenario);
        let mut registry = test_scenario::take_shared<CombatRegistry>(&scenario);
        
        let participants = vector[PLAYER1, PLAYER2];
        let health = vector[100, 150];
        let mana = vector[50, 75];
        let power = vector[20, 25];
        let reward_coin = mint_for_testing<SUI>(2000, test_scenario::ctx(&mut scenario));
        
        // Create detailed combat session
        combat::create_detailed_combat(
            &admin_cap,
            &mut registry,
            3,
            1, // PVP
            participants,
            health,
            mana,
            power,
            reward_coin,
            10, // max_turns
            b"detailed test combat",
            test_scenario::ctx(&mut scenario)
        );
        
        test_scenario::return_shared(registry);
        test_scenario::return_to_sender(&scenario, admin_cap);
        test_scenario::end(scenario);
    }
}
