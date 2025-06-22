#[test_only]
module mmo_game::world_tests {
    use mmo_game::world::{Self, AdminCap, WorldRegistry, ResourceNode, WorldRegion, WorldEvent};
    use sui::test_scenario;
    use sui::coin;
    use sui::sui::SUI;    use sui::clock;
    use std::vector;

    const ADMIN: address = @0xAD;
    const PLAYER1: address = @0x123;

    #[test]
    fun test_comprehensive_resource_management() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
          // Setup
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            world::setup_test_scenario(ADMIN, ctx);
        };
        
        // Create resource node
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut registry = test_scenario::take_shared<WorldRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            world::create_resource_node(
                &admin_cap,
                &mut registry,
                1, // RESOURCE_GOLD
                100, 200, 50, // coordinates
                1000, // initial yield
                5000, // max yield
                100000000, // harvest cost (0.1 SUI)
                10, // regeneration rate
                true, // is public
                ctx
            );
            
            assert!(world::get_total_nodes(&registry) == 1, 0);
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(registry);
        };        // Test harvesting
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut node = test_scenario::take_shared<ResourceNode>(scenario);
            let mut clock = clock::create_for_testing(test_scenario::ctx(scenario));
            let ctx = test_scenario::ctx(scenario);
            
            // Advance clock to pass cooldown (1 hour = 3600 seconds = 3600000 ms)
            clock::increment_for_testing(&mut clock, 3600000);
            
            let payment = coin::mint_for_testing<SUI>(100000000, ctx); // 0.1 SUI
            world::harvest_resource(&mut node, payment, &clock, ctx);
            
            // Check that yield was reduced
            assert!(world::get_node_current_yield(&node) < 1000, 1);
            
            clock::destroy_for_testing(clock);
            test_scenario::return_shared(node);
        };
        
        test_scenario::end(scenario_val);
    }    #[test]
    fun test_region_management() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
          // Setup
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            world::setup_test_scenario(ADMIN, ctx);
        };
        
        // Create region
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut registry = test_scenario::take_shared<WorldRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            world::create_world_region(
                &admin_cap,
                &mut registry,
                b"Test Mountains",
                3, // REGION_MOUNTAINS
                0, 500, // x bounds
                0, 500, // y bounds
                100, 300, // z bounds
                b"Cold Mountain Climate",
                7, // danger level
                200, // resource multiplier
                1000000000, // control fee (1 SUI)
                ctx
            );
            
            assert!(world::get_total_regions(&registry) == 1, 0);
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(registry);
        };
          // Test region control
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut region = test_scenario::take_shared<WorldRegion>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            let payment = coin::mint_for_testing<SUI>(1000000000, ctx); // 1 SUI
            world::claim_region_control(&mut region, payment, ctx);
            
            assert!(world::get_region_controller(&region) == std::option::some(PLAYER1), 1);
            
            test_scenario::return_shared(region);
        };
        
        // Test releasing control
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut region = test_scenario::take_shared<WorldRegion>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            world::release_region_control(&mut region, ctx);
            
            assert!(std::option::is_none(&world::get_region_controller(&region)), 2);
            
            test_scenario::return_shared(region);
        };
        
        test_scenario::end(scenario_val);
    }    #[test]    fun test_world_events() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        // Setup
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            world::setup_test_scenario(ADMIN, ctx);
        };
        
        // Create world event
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut registry = test_scenario::take_shared<WorldRegistry>(scenario);
            let clock = clock::create_for_testing(test_scenario::ctx(scenario));
            let ctx = test_scenario::ctx(scenario);
            
            let reward_pool = coin::mint_for_testing<SUI>(10000000000, ctx); // 10 SUI
            world::trigger_world_event(
                &admin_cap,
                &mut registry,
                1, // EVENT_DRAGON_INVASION
                b"Dragon Attack",
                b"Dragons are attacking the realm!",
                24, // 24 hour duration
                vector::empty(), // affected regions
                150, // effect multiplier
                reward_pool,
                &clock,
                ctx
            );
            
            assert!(vector::length(&world::get_active_events(&registry)) == 1, 0);
            
            clock::destroy_for_testing(clock);
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(registry);
        };
          // Participate in event
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut event = test_scenario::take_shared<WorldEvent>(scenario);
            let clock = clock::create_for_testing(test_scenario::ctx(scenario));
            let ctx = test_scenario::ctx(scenario);
            
            world::participate_in_event(&mut event, &clock, ctx);
            
            assert!(vector::length(&world::get_event_participants(&event)) == 1, 1);
            assert!(world::is_event_active(&event), 2);
            
            clock::destroy_for_testing(clock);
            test_scenario::return_shared(event);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]    fun test_treasure_discovery() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        // Setup
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            world::setup_test_scenario(ADMIN, ctx);
        };
        
        // Create treasure location
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            world::create_treasure_location(
                &admin_cap,
                150, 300, 75, // coordinates
                b"Ancient Chest",
                5000, // value
                1000, // discovery reward
                ctx
            );
            
            test_scenario::return_to_sender(scenario, admin_cap);
        };
          // Discover treasure
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut treasure = test_scenario::take_shared<world::TreasureLocation>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            world::discover_treasure(&mut treasure, ctx);
            
            // Verify treasure was discovered
            let discoverer = world::get_treasure_discoverer(&treasure);
            assert!(std::option::is_some(&discoverer), 0);
            assert!(*std::option::borrow(&discoverer) == PLAYER1, 1);
            
            test_scenario::return_shared(treasure);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_node_ownership() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        // Setup
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            world::setup_test_scenario(ADMIN, ctx);
        };
          // Create private resource node
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut registry = test_scenario::take_shared<WorldRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            world::create_resource_node(
                &admin_cap,
                &mut registry,
                2, // RESOURCE_IRON
                50, 100, 25, // coordinates
                800, // initial yield
                4000, // max yield
                50000000, // harvest cost (0.05 SUI)
                5, // regeneration rate
                false, // not public
                ctx
            );
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(registry);
        };
          // Purchase ownership
        test_scenario::next_tx(scenario, ADMIN);
        {
            let mut node = test_scenario::take_from_sender<ResourceNode>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            // Calculate purchase cost and pay
            let purchase_cost = (4000 * 80) / 100; // 1% of total value
            let payment = coin::mint_for_testing<SUI>(purchase_cost, ctx);
            world::purchase_node_ownership(&mut node, payment, ctx);
            
            assert!(world::get_node_owner(&node) == std::option::some(ADMIN), 0);
            
            test_scenario::return_to_sender(scenario, node);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]    fun test_node_regeneration() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
        
        // Setup and create node
        test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            world::setup_test_scenario(ADMIN, ctx);
        };
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut registry = test_scenario::take_shared<WorldRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            world::create_resource_node(
                &admin_cap,
                &mut registry,
                3, // RESOURCE_GEMS
                75, 150, 30, // coordinates
                500, // initial yield
                2000, // max yield
                200000000, // harvest cost (0.2 SUI)
                20, // regeneration rate
                true, // is public
                ctx
            );
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(registry);
        };        // Harvest to reduce yield
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut node = test_scenario::take_shared<ResourceNode>(scenario);
            let mut clock = clock::create_for_testing(test_scenario::ctx(scenario));
            let ctx = test_scenario::ctx(scenario);
            
            // Advance clock to pass cooldown
            clock::increment_for_testing(&mut clock, 3600000);
            
            let payment = coin::mint_for_testing<SUI>(200000000, ctx);
            world::harvest_resource(&mut node, payment, &clock, ctx);
            
            let _yield_after_harvest = world::get_node_current_yield(&node);
            
            clock::destroy_for_testing(clock);
            test_scenario::return_shared(node);
        };
        
        // Test regeneration
        test_scenario::next_tx(scenario, ADMIN);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut node = test_scenario::take_shared<ResourceNode>(scenario);
            let mut clock = clock::create_for_testing(test_scenario::ctx(scenario));
            let ctx = test_scenario::ctx(scenario);
            
            // Advance time and regenerate
            clock::increment_for_testing(&mut clock, 3600000); // 1 hour
            world::regenerate_node(&admin_cap, &mut node, &clock, ctx);
            
            // Should have regenerated some yield
            assert!(world::get_node_current_yield(&node) > 0, 0);
            
            clock::destroy_for_testing(clock);
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(node);
        };
        
        test_scenario::end(scenario_val);
    }
}
