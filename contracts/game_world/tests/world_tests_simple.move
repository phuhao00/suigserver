#[test_only]
module mmo_game::world_tests_simple {
    use mmo_game::world;
    use sui::test_scenario;
    use sui::coin;
    use sui::sui::SUI;
    use sui::clock;

    const ADMIN: address = @0xAD;
    const PLAYER1: address = @0x123;

    #[test]    fun test_basic_resource_node_creation() {
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
            let admin_cap = test_scenario::take_from_sender<world::AdminCap>(scenario);
            let mut registry = test_scenario::take_shared<world::WorldRegistry>(scenario);
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
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    fun test_basic_harvest() {
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
            let admin_cap = test_scenario::take_from_sender<world::AdminCap>(scenario);
            let mut registry = test_scenario::take_shared<world::WorldRegistry>(scenario);
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
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(registry);
        };
          // Test harvesting
        test_scenario::next_tx(scenario, PLAYER1);
        {
            let mut node = test_scenario::take_shared<world::ResourceNode>(scenario);
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
    }

    #[test]
    fun test_registry_initialization() {
        let mut scenario_val = test_scenario::begin(ADMIN);
        let scenario = &mut scenario_val;
          test_scenario::next_tx(scenario, ADMIN);
        {
            let ctx = test_scenario::ctx(scenario);
            world::setup_test_scenario(ADMIN, ctx);
        };
        
        test_scenario::next_tx(scenario, ADMIN);
        {
            let registry = test_scenario::take_shared<world::WorldRegistry>(scenario);
            
            // Test initial values
            assert!(world::get_total_regions(&registry) == 0, 0);
            assert!(world::get_total_nodes(&registry) == 0, 1);
            assert!(vector::length(&world::get_active_events(&registry)) == 0, 2);
            
            test_scenario::return_shared(registry);
        };
        
        test_scenario::end(scenario_val);
    }
}
