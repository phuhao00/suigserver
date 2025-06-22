#[test_only]
module mmo_game::guild_tests {
    use mmo_game::guild::{Self, Guild, GuildRegistry};
    use sui::test_scenario;
    use sui::coin;
    use sui::sui::SUI;
    use std::string;
    use std::vector;

    const ADMIN: address = @0xAD;
    const LEADER: address = @0x123;
    const MEMBER1: address = @0x456;
    const MEMBER2: address = @0x789;
    const MEMBER3: address = @0xABC;

    fun setup_guild(scenario: &mut test_scenario::Scenario): Guild {
        // Create registry
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            let registry = guild::create_registry_for_testing(ctx);
            test_scenario::share_object(registry);
        };

        // Create guild
        test_scenario::next_tx(scenario, LEADER);
        {
            let registry = test_scenario::take_shared<GuildRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            guild::create_guild(
                &mut registry,
                b"Test Guild",
                b"A comprehensive test guild",
                ctx
            );
            
            test_scenario::return_shared(registry);
        };

        test_scenario::next_tx(scenario, LEADER);
        test_scenario::take_from_sender<Guild>(scenario)
    }

    #[test]
    fun test_comprehensive_guild_management() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        // Test initial state
        assert!(guild::get_guild_name(&guild) == string::utf8(b"Test Guild"), 0);
        assert!(guild::get_guild_leader(&guild) == LEADER, 1);
        assert!(guild::get_member_count(&guild) == 1, 2);
        assert!(guild::get_officer_count(&guild) == 0, 3);
        assert!(guild::get_guild_level(&guild) == 1, 4);
        assert!(guild::get_max_members(&guild) == 25, 5);
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_member_lifecycle() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        // Add multiple members
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_member(&mut guild, MEMBER1, ctx);
            guild::add_member(&mut guild, MEMBER2, ctx);
            guild::add_member(&mut guild, MEMBER3, ctx);
        };
        
        assert!(guild::get_member_count(&guild) == 4, 0);
        assert!(guild::is_member(&guild, MEMBER1), 1);
        assert!(guild::is_member(&guild, MEMBER2), 2);
        assert!(guild::is_member(&guild, MEMBER3), 3);
        
        // Test member roles
        assert!(guild::get_member_role(&guild, LEADER) == 0, 4); // ROLE_LEADER
        assert!(guild::get_member_role(&guild, MEMBER1) == 2, 5); // ROLE_MEMBER
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_officer_management() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        // Add a member and promote to officer
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_member(&mut guild, MEMBER1, ctx);
            guild::promote_to_officer(&mut guild, MEMBER1, ctx);
        };
        
        assert!(guild::is_officer(&guild, MEMBER1), 0);
        assert!(guild::get_officer_count(&guild) == 1, 1);
        assert!(guild::get_member_role(&guild, MEMBER1) == 1, 2); // ROLE_OFFICER
        
        // Test officer can manage members
        test_scenario::next_tx(scenario, MEMBER1);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_member(&mut guild, MEMBER2, ctx);
        };
        
        assert!(guild::is_member(&guild, MEMBER2), 3);
        
        // Demote officer
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::demote_officer(&mut guild, MEMBER1, ctx);
        };
        
        assert!(!guild::is_officer(&guild, MEMBER1), 4);
        assert!(guild::get_member_role(&guild, MEMBER1) == 2, 5); // ROLE_MEMBER
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_leadership_transfer() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        // Add a member
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_member(&mut guild, MEMBER1, ctx);
        };
        
        // Transfer leadership
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::transfer_leadership(&mut guild, MEMBER1, ctx);
        };
        
        assert!(guild::get_guild_leader(&guild) == MEMBER1, 0);
        assert!(guild::is_leader(&guild, MEMBER1), 1);
        assert!(!guild::is_leader(&guild, LEADER), 2);
        assert!(guild::get_member_role(&guild, MEMBER1) == 0, 3); // ROLE_LEADER
        assert!(guild::get_member_role(&guild, LEADER) == 2, 4); // ROLE_MEMBER
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_leave_guild() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        // Add members
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_member(&mut guild, MEMBER1, ctx);
            guild::add_member(&mut guild, MEMBER2, ctx);
        };
        
        let initial_count = guild::get_member_count(&guild);
        
        // Member leaves guild
        test_scenario::next_tx(scenario, MEMBER1);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::leave_guild(&mut guild, ctx);
        };
        
        assert!(guild::get_member_count(&guild) == initial_count - 1, 0);
        assert!(!guild::is_member(&guild, MEMBER1), 1);
        assert!(guild::is_member(&guild, MEMBER2), 2);
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_treasury_management() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        // Add member
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_member(&mut guild, MEMBER1, ctx);
        };
        
        // Member deposits to treasury
        test_scenario::next_tx(scenario, MEMBER1);
        {
            let ctx = test_scenario::ctx(scenario);
            let payment = coin::mint_for_testing<SUI>(2000000000, ctx); // 2 SUI
            guild::deposit_to_treasury(&mut guild, payment, ctx);
        };
        
        assert!(guild::get_treasury_balance(&guild) == 2000000000, 0);
        
        // Leader deposits more
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            let payment = coin::mint_for_testing<SUI>(1000000000, ctx); // 1 SUI
            guild::deposit_to_treasury(&mut guild, payment, ctx);
        };
        
        assert!(guild::get_treasury_balance(&guild) == 3000000000, 1);
        
        // Leader withdraws
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::withdraw_from_treasury(&mut guild, 1500000000, ctx); // 1.5 SUI
        };
        
        assert!(guild::get_treasury_balance(&guild) == 1500000000, 2);
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_guild_settings() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        // Update MOTD
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::update_motd(&mut guild, b"New message of the day!", ctx);
        };
        
        assert!(guild::get_guild_motd(&guild) == string::utf8(b"New message of the day!"), 0);
        
        // Update description
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::update_description(&mut guild, b"Updated guild description", ctx);
        };
        
        assert!(guild::get_guild_description(&guild) == string::utf8(b"Updated guild description"), 1);
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_guild_experience_and_leveling() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        let initial_level = guild::get_guild_level(&guild);
        let initial_exp = guild::get_guild_experience(&guild);
        
        // Add experience
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_guild_experience(&mut guild, 500, ctx);
        };
        
        assert!(guild::get_guild_experience(&guild) == initial_exp + 500, 0);
        
        // Add enough experience to level up
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_guild_experience(&mut guild, 1000, ctx); // Should level up
        };
        
        assert!(guild::get_guild_level(&guild) == initial_level + 1, 1);
        assert!(guild::get_guild_experience(&guild) == 500, 2); // Remaining after level up
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_member_info_tracking() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        // Add member
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_member(&mut guild, MEMBER1, ctx);
        };
        
        // Check member info
        let (role, joined_at, contribution) = guild::get_member_info(&guild, MEMBER1);
        assert!(role == 2, 0); // ROLE_MEMBER
        assert!(joined_at > 0, 1); // Should have a join timestamp
        assert!(contribution == 0, 2); // Initial contribution
        
        // Member makes contribution
        test_scenario::next_tx(scenario, MEMBER1);
        {
            let ctx = test_scenario::ctx(scenario);
            let payment = coin::mint_for_testing<SUI>(1000000000, ctx); // 1 SUI
            guild::deposit_to_treasury(&mut guild, payment, ctx);
        };
        
        // Check updated contribution
        let (_, _, new_contribution) = guild::get_member_info(&guild, MEMBER1);
        assert!(new_contribution == 1000000000, 3);
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_guild_vectors() {
        let scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        let guild = setup_guild(scenario);
        
        // Add members and officers
        test_scenario::next_tx(scenario, LEADER);
        {
            let ctx = test_scenario::ctx(scenario);
            guild::add_member(&mut guild, MEMBER1, ctx);
            guild::add_member(&mut guild, MEMBER2, ctx);
            guild::promote_to_officer(&mut guild, MEMBER1, ctx);
        };
        
        let members = guild::get_members(&guild);
        let officers = guild::get_officers(&guild);
        
        assert!(vector::length(&members) == 3, 0); // Leader + 2 members
        assert!(vector::length(&officers) == 1, 1); // 1 officer
        assert!(vector::contains(&members, &LEADER), 2);
        assert!(vector::contains(&members, &MEMBER1), 3);
        assert!(vector::contains(&members, &MEMBER2), 4);
        assert!(vector::contains(&officers, &MEMBER1), 5);
        
        test_scenario::return_to_sender(scenario, guild);
        test_scenario::end(scenario_val);
    }
}
