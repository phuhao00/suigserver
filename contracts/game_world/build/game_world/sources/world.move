// Sui Move Game World Contract
// Manages aspects of the game world that are on-chain.
// This includes definitions of regions, control over dynamic world events,
// and ownership of certain persistent world objects/resources.

module mmo_game::world {    use sui::object::{Self, UID, ID};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer;
    use sui::event;
    use std::string::{Self, String};
    use std::vector;
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, Coin};
    use sui::sui::SUI;
    use sui::dynamic_field;
    use sui::clock::{Self, Clock};
    use std::option;

    // Error constants
    const E_NOT_ADMIN: u64 = 0;
    const E_RESOURCE_EXHAUSTED: u64 = 1;
    const E_COOLDOWN_ACTIVE: u64 = 2;
    const E_INVALID_COORDINATES: u64 = 3;
    const E_REGION_NOT_FOUND: u64 = 4;
    const E_INSUFFICIENT_PAYMENT: u64 = 5;
    const E_EVENT_NOT_ACTIVE: u64 = 6;
    const E_INVALID_RESOURCE_TYPE: u64 = 7;
    const E_NODE_ALREADY_EXISTS: u64 = 8;

    // Resource type constants
    const RESOURCE_GOLD: u8 = 1;
    const RESOURCE_IRON: u8 = 2;
    const RESOURCE_GEMS: u8 = 3;
    const RESOURCE_MITHRIL: u8 = 4;
    const RESOURCE_CRYSTALS: u8 = 5;

    // Region type constants
    const REGION_PLAINS: u8 = 1;
    const REGION_FOREST: u8 = 2;
    const REGION_MOUNTAINS: u8 = 3;
    const REGION_DESERT: u8 = 4;
    const REGION_SWAMP: u8 = 5;
    const REGION_ARCTIC: u8 = 6;

    // World event type constants
    const EVENT_DRAGON_INVASION: u8 = 1;
    const EVENT_RESOURCE_BOOM: u8 = 2;
    const EVENT_MERCHANT_CARAVAN: u8 = 3;
    const EVENT_MYSTICAL_FOG: u8 = 4;
    const EVENT_TREASURE_HUNT: u8 = 5;

    // Admin capability for privileged functions
    public struct AdminCap has key, store {
        id: UID,
    }    // World registry - global state manager
    public struct WorldRegistry has key {
        id: UID,
        total_regions: u64,
        total_nodes: u64,
        active_events: vector<u64>,
        next_event_id: u64, // Counter for generating numeric event IDs
        world_seed: u64, // For procedural generation
    }

    // Persistent world resource node
    public struct ResourceNode has key, store {
        id: UID,
        resource_type: u8,
        coordinates: Coordinates,
        last_harvested_epoch: u64,
        yield_per_harvest: u64,
        max_yield: u64,
        current_yield: u64,
        harvest_cost: u64, // Cost in SUI to harvest
        regeneration_rate: u64, // How much yield regenerates per epoch
        owner: option::Option<address>, // Can be owned by players/guilds
        is_public: bool,
    }

    // World region definition
    public struct WorldRegion has key, store {
        id: UID,
        name: String,
        region_type: u8,
        coordinates: RegionBounds,
        climate: String,
        danger_level: u8, // 1-10
        resource_multiplier: u64, // Affects resource yield in this region
        special_properties: vector<String>,
        controlled_by: option::Option<address>, // Guild or player control
        control_fee: u64, // Fee to control this region
    }

    // Coordinate system
    public struct Coordinates has store, copy, drop {
        x: u64,
        y: u64,
        z: u64, // For vertical positioning
    }

    // Region boundaries
    public struct RegionBounds has store, copy, drop {
        min_x: u64,
        max_x: u64,
        min_y: u64,
        max_y: u64,
        min_z: u64,
        max_z: u64,
    }    // Dynamic world event
    public struct WorldEvent has key, store {
        id: UID,
        event_id: u64, // Numeric ID for tracking
        event_type: u8,
        name: String,
        description: String,
        start_epoch: u64,
        end_epoch: u64,
        affected_regions: vector<ID>,
        effect_multiplier: u64,
        reward_pool: Balance<SUI>,
        participants: vector<address>,
        is_active: bool,
    }

    // Treasure locations for events
    public struct TreasureLocation has key, store {
        id: UID,
        coordinates: Coordinates,
        treasure_type: String,
        value: u64,
        discovered_by: option::Option<address>,
        discovery_reward: u64,
    }

    // Events
    public struct WorldEventTriggered has copy, drop {
        event_id: u64,
        event_type: u8,
        name: String,
        description: String,
        affected_regions: vector<ID>,
    }

    public struct ResourceHarvested has copy, drop {
        node_id: ID,
        harvester: address,
        resource_type: u8,
        amount: u64,
        coordinates: Coordinates,
    }

    public struct RegionControlChanged has copy, drop {
        region_id: ID,
        old_controller: option::Option<address>,
        new_controller: option::Option<address>,
        control_fee: u64,
    }

    public struct TreasureDiscovered has copy, drop {
        treasure_id: ID,
        discoverer: address,
        coordinates: Coordinates,
        value: u64,
    }

    public struct NodeCreated has copy, drop {
        node_id: ID,
        resource_type: u8,
        coordinates: Coordinates,
        initial_yield: u64,
    }

    // --- Initialization ---

    // Initialize the world system (call once during deployment)
    fun init(ctx: &mut TxContext) {
        let admin_cap = AdminCap {
            id: object::new(ctx),
        };
          let registry = WorldRegistry {
            id: object::new(ctx),
            total_regions: 0,
            total_nodes: 0,
            active_events: vector::empty(),
            next_event_id: 1, // Start event IDs from 1
            world_seed: tx_context::epoch(ctx),
        };

        transfer::transfer(admin_cap, tx_context::sender(ctx));
        transfer::share_object(registry);
    }

    // Example: Initialize a part of the world or a specific resource
    // public entry fun initialize_resource_node(resource_type: u8, ctx: &mut TxContext) {
    //     let node = ResourceNode {
    //         id: object::new(ctx),
    //         resource_type: resource_type,
    //         last_harvested_epoch: 0, // Or current epoch from ctx
    //     };
    //     // Transfer to a world admin/system address or make it a shared object
    //     transfer::share_object(node);
    // }

    // Example: A function that could be called by the server to trigger an on-chain event
    // public entry fun trigger_world_event(
    //     _admin_cap: &AdminCap, // Requires server capability
    //     event_id: u64,
    //     description: vector<u8>,
    //     _ctx: &mut TxContext
    // ) {
    //     // Emit an event that off-chain systems or other contracts can react to
    //     event::emit(WorldEventTriggered {
    //         event_id,
    //         description,
    //     });
    //     // Potentially modify some on-chain state related to the event
    // }

    // AdminCap would be defined as in other modules, if needed for privileged functions.
    // struct AdminCap has key, store { id: UID }

    // --- Resource Node Management ---

    // Initialize a resource node in the world
    public entry fun create_resource_node(
        _admin_cap: &AdminCap,
        registry: &mut WorldRegistry,
        resource_type: u8,
        x: u64,
        y: u64,
        z: u64,
        initial_yield: u64,
        max_yield: u64,
        harvest_cost: u64,
        regeneration_rate: u64,
        is_public: bool,
        ctx: &mut TxContext
    ) {
        assert!(resource_type >= 1 && resource_type <= 5, E_INVALID_RESOURCE_TYPE);

        let coordinates = Coordinates { x, y, z };
        let node_id = object::new(ctx);
        let node_id_copy = object::uid_to_inner(&node_id);

        let node = ResourceNode {
            id: node_id,
            resource_type,
            coordinates,
            last_harvested_epoch: 0,
            yield_per_harvest: initial_yield / 10, // 10% of max per harvest
            max_yield,
            current_yield: initial_yield,
            harvest_cost,
            regeneration_rate,
            owner: option::none(),
            is_public,
        };

        registry.total_nodes = registry.total_nodes + 1;

        event::emit(NodeCreated {
            node_id: node_id_copy,
            resource_type,
            coordinates,
            initial_yield,
        });

        if (is_public) {
            transfer::share_object(node);
        } else {
            transfer::transfer(node, tx_context::sender(ctx));
        };
    }

    // Harvest resources from a node
    public entry fun harvest_resource(
        node: &mut ResourceNode,
        payment: Coin<SUI>,
        clock: &Clock,
        ctx: &mut TxContext
    ) {
        let harvester = tx_context::sender(ctx);
        let current_epoch = clock::timestamp_ms(clock) / 1000; // Convert to seconds
        
        // Check if harvest cost is met
        assert!(coin::value(&payment) >= node.harvest_cost, E_INSUFFICIENT_PAYMENT);
        
        // Check if cooldown period has passed (1 hour = 3600 seconds)
        assert!(current_epoch >= node.last_harvested_epoch + 3600, E_COOLDOWN_ACTIVE);
        
        // Check if node has resources
        assert!(node.current_yield > 0, E_RESOURCE_EXHAUSTED);

        // Calculate harvest amount
        let harvest_amount = if (node.current_yield >= node.yield_per_harvest) {
            node.yield_per_harvest
        } else {
            node.current_yield
        };        // Update node state
        node.current_yield = node.current_yield - harvest_amount;
        node.last_harvested_epoch = current_epoch;

        // Simple payment handling - in real implementation, this would go to a treasury
        // For now, we'll just consume the payment by transferring it back to sender
        // (effectively a refund in testing, but represents payment processing)
        transfer::public_transfer(payment, harvester);

        event::emit(ResourceHarvested {
            node_id: object::uid_to_inner(&node.id),
            harvester,
            resource_type: node.resource_type,
            amount: harvest_amount,
            coordinates: node.coordinates,
        });
    }

    // Regenerate resources in a node (called periodically by system)
    public entry fun regenerate_node(
        _admin_cap: &AdminCap,
        node: &mut ResourceNode,
        clock: &Clock,
        _ctx: &mut TxContext
    ) {
        let current_epoch = clock::timestamp_ms(clock) / 1000;
        let epochs_passed = current_epoch - node.last_harvested_epoch;
        
        if (epochs_passed > 0 && node.current_yield < node.max_yield) {
            let regeneration = epochs_passed * node.regeneration_rate;
            let new_yield = node.current_yield + regeneration;
            
            node.current_yield = if (new_yield > node.max_yield) {
                node.max_yield
            } else {
                new_yield
            };
        };
    }

    // Purchase ownership of a resource node
    public entry fun purchase_node_ownership(
        node: &mut ResourceNode,
        payment: Coin<SUI>,
        ctx: &mut TxContext
    ) {
        let buyer = tx_context::sender(ctx);
        
        // Calculate purchase cost based on node value
        let purchase_cost = (node.max_yield * node.yield_per_harvest) / 100; // 1% of total value
        assert!(coin::value(&payment) >= purchase_cost, E_INSUFFICIENT_PAYMENT);        // Transfer ownership
        node.owner = option::some(buyer);
        
        // Handle payment - in real implementation, this would go to previous owner or treasury
        transfer::public_transfer(payment, buyer);
    }

    // --- World Region Management ---

    // Create a new world region
    public entry fun create_world_region(
        _admin_cap: &AdminCap,
        registry: &mut WorldRegistry,
        name: vector<u8>,
        region_type: u8,
        min_x: u64, max_x: u64,
        min_y: u64, max_y: u64,
        min_z: u64, max_z: u64,
        climate: vector<u8>,
        danger_level: u8,
        resource_multiplier: u64,
        control_fee: u64,
        ctx: &mut TxContext
    ) {
        assert!(region_type >= 1 && region_type <= 6, E_INVALID_COORDINATES);
        assert!(danger_level >= 1 && danger_level <= 10, E_INVALID_COORDINATES);

        let region_id = object::new(ctx);
        
        let region = WorldRegion {
            id: region_id,
            name: string::utf8(name),
            region_type,
            coordinates: RegionBounds {
                min_x, max_x,
                min_y, max_y,
                min_z, max_z,
            },
            climate: string::utf8(climate),
            danger_level,
            resource_multiplier,
            special_properties: vector::empty(),
            controlled_by: option::none(),
            control_fee,
        };

        registry.total_regions = registry.total_regions + 1;
        transfer::share_object(region);
    }

    // Claim control of a region
    public entry fun claim_region_control(
        region: &mut WorldRegion,
        payment: Coin<SUI>,
        ctx: &mut TxContext
    ) {
        let claimer = tx_context::sender(ctx);
        assert!(coin::value(&payment) >= region.control_fee, E_INSUFFICIENT_PAYMENT);        let old_controller = region.controlled_by;
        region.controlled_by = option::some(claimer);

        // Handle payment - in real implementation, could go to treasury or previous controller
        transfer::public_transfer(payment, claimer);

        event::emit(RegionControlChanged {
            region_id: object::uid_to_inner(&region.id),
            old_controller,
            new_controller: option::some(claimer),
            control_fee: region.control_fee,
        });
    }

    // Release control of a region
    public entry fun release_region_control(
        region: &mut WorldRegion,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(region.controlled_by == option::some(sender), E_NOT_ADMIN);

        let old_controller = region.controlled_by;
        region.controlled_by = option::none();

        event::emit(RegionControlChanged {
            region_id: object::uid_to_inner(&region.id),
            old_controller,
            new_controller: option::none(),
            control_fee: 0,
        });
    }

    // Add special property to a region
    public entry fun add_region_property(
        _admin_cap: &AdminCap,
        region: &mut WorldRegion,
        property: vector<u8>,
        _ctx: &mut TxContext
    ) {
        vector::push_back(&mut region.special_properties, string::utf8(property));
    }

    // --- World Events Management ---

    // Trigger a world event
    public entry fun trigger_world_event(
        _admin_cap: &AdminCap,
        registry: &mut WorldRegistry,
        event_type: u8,
        name: vector<u8>,
        description: vector<u8>,
        duration_epochs: u64,
        affected_regions: vector<ID>,
        effect_multiplier: u64,
        reward_pool: Coin<SUI>,
        clock: &Clock,
        ctx: &mut TxContext    ) {
        let start_epoch = clock::timestamp_ms(clock) / 1000;
        let end_epoch = start_epoch + (duration_epochs * 3600);

        // Generate numeric event ID
        let numeric_event_id = registry.next_event_id;
        registry.next_event_id = registry.next_event_id + 1;

        let world_event = WorldEvent {
            id: object::new(ctx),
            event_id: numeric_event_id,
            event_type,
            name: string::utf8(name),
            description: string::utf8(description),
            start_epoch,
            end_epoch,
            affected_regions,
            effect_multiplier,
            reward_pool: coin::into_balance(reward_pool),
            participants: vector::empty(),
            is_active: true,
        };

        vector::push_back(&mut registry.active_events, numeric_event_id);

        event::emit(WorldEventTriggered {
            event_id: numeric_event_id,
            event_type,
            name: string::utf8(name),
            description: string::utf8(description),
            affected_regions,
        });

        transfer::share_object(world_event);
    }

    // Participate in a world event
    public entry fun participate_in_event(
        world_event: &mut WorldEvent,
        clock: &Clock,
        ctx: &mut TxContext
    ) {
        let participant = tx_context::sender(ctx);
        let current_time = clock::timestamp_ms(clock) / 1000;

        assert!(world_event.is_active, E_EVENT_NOT_ACTIVE);
        assert!(current_time >= world_event.start_epoch && current_time <= world_event.end_epoch, E_EVENT_NOT_ACTIVE);
        assert!(!vector::contains(&world_event.participants, &participant), E_EVENT_NOT_ACTIVE);

        vector::push_back(&mut world_event.participants, participant);
    }

    // End a world event and distribute rewards
    public entry fun end_world_event(
        _admin_cap: &AdminCap,
        registry: &mut WorldRegistry,
        world_event: &mut WorldEvent,
        clock: &Clock,
        ctx: &mut TxContext
    ) {
        let current_time = clock::timestamp_ms(clock) / 1000;
        assert!(current_time >= world_event.end_epoch, E_EVENT_NOT_ACTIVE);        world_event.is_active = false;

        // Remove from active events using numeric event ID
        let event_id = world_event.event_id;
        let (found, index) = vector::index_of(&registry.active_events, &event_id);
        if (found) {
            vector::remove(&mut registry.active_events, index);
        };

        // Distribute rewards to participants
        let participant_count = vector::length(&world_event.participants);        if (participant_count > 0) {
            let reward_per_participant = balance::value(&world_event.reward_pool) / participant_count;
            
            let mut i = 0;
            while (i < participant_count) {
                let participant = *vector::borrow(&world_event.participants, i);
                let reward = coin::take(&mut world_event.reward_pool, reward_per_participant, ctx);
                transfer::public_transfer(reward, participant);
                i = i + 1;
            };
        };
    }

    // Create a treasure location during events
    public entry fun create_treasure_location(
        _admin_cap: &AdminCap,
        x: u64, y: u64, z: u64,
        treasure_type: vector<u8>,
        value: u64,
        discovery_reward: u64,
        ctx: &mut TxContext
    ) {
        let treasure_id = object::new(ctx);
        
        let treasure = TreasureLocation {
            id: treasure_id,
            coordinates: Coordinates { x, y, z },
            treasure_type: string::utf8(treasure_type),
            value,
            discovered_by: option::none(),
            discovery_reward,
        };

        transfer::share_object(treasure);
    }

    // Discover a treasure location
    public entry fun discover_treasure(
        treasure: &mut TreasureLocation,
        ctx: &mut TxContext
    ) {
        let discoverer = tx_context::sender(ctx);
        assert!(option::is_none(&treasure.discovered_by), E_EVENT_NOT_ACTIVE);

        treasure.discovered_by = option::some(discoverer);        event::emit(TreasureDiscovered {
            treasure_id: object::uid_to_inner(&treasure.id),
            discoverer,
            coordinates: treasure.coordinates,
            value: treasure.value,
        });
    }

    // --- Getter Functions ---

    // Resource Node getters
    public fun get_node_resource_type(node: &ResourceNode): u8 { node.resource_type }
    public fun get_node_coordinates(node: &ResourceNode): Coordinates { node.coordinates }
    public fun get_node_current_yield(node: &ResourceNode): u64 { node.current_yield }
    public fun get_node_max_yield(node: &ResourceNode): u64 { node.max_yield }
    public fun get_node_harvest_cost(node: &ResourceNode): u64 { node.harvest_cost }
    public fun get_node_owner(node: &ResourceNode): option::Option<address> { node.owner }
    public fun is_node_public(node: &ResourceNode): bool { node.is_public }

    // Region getters
    public fun get_region_name(region: &WorldRegion): String { region.name }
    public fun get_region_type(region: &WorldRegion): u8 { region.region_type }
    public fun get_region_danger_level(region: &WorldRegion): u8 { region.danger_level }
    public fun get_region_controller(region: &WorldRegion): option::Option<address> { region.controlled_by }    // Event getters
    public fun get_event_id(event: &WorldEvent): u64 { event.event_id }
    public fun get_event_name(event: &WorldEvent): String { event.name }
    public fun get_event_type(event: &WorldEvent): u8 { event.event_type }
    public fun is_event_active(event: &WorldEvent): bool { event.is_active }
    public fun get_event_participants(event: &WorldEvent): vector<address> { event.participants }

    // Registry getters
    public fun get_total_regions(registry: &WorldRegistry): u64 { registry.total_regions }
    public fun get_total_nodes(registry: &WorldRegistry): u64 { registry.total_nodes }
    public fun get_active_events(registry: &WorldRegistry): vector<u64> { registry.active_events }

    // --- Test Helper Functions ---
    #[test_only]
    public fun create_admin_cap_for_testing(ctx: &mut TxContext): AdminCap {
        AdminCap { id: object::new(ctx) }
    }

    #[test_only] 
    public fun setup_test_scenario(admin: address, ctx: &mut TxContext) {
        let admin_cap = create_admin_cap_for_testing(ctx);
        let registry = create_registry_for_testing(ctx);
        
        transfer::public_transfer(admin_cap, admin);
        transfer::share_object(registry);
    }

    #[test_only]
    public fun create_registry_for_testing(ctx: &mut TxContext): WorldRegistry {
        WorldRegistry {
            id: object::new(ctx),
            total_regions: 0,
            total_nodes: 0,
            active_events: vector::empty(),
            next_event_id: 1,
            world_seed: 12345,
        }
    }

    #[test_only]
    public fun get_treasure_discoverer(treasure: &TreasureLocation): option::Option<address> {
        treasure.discovered_by
    }

    // === Test Functions ===
    #[test_only]
    use sui::test_scenario;    #[test]
    public fun test_create_resource_node() {
        let admin = @0xAD;
        let mut scenario_val = test_scenario::begin(admin);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, admin);
        {
            let ctx = test_scenario::ctx(scenario);
            let admin_cap = AdminCap { id: object::new(ctx) };
            let registry = WorldRegistry {
                id: object::new(ctx),
                total_regions: 0,
                total_nodes: 0,
                active_events: vector::empty(),
                next_event_id: 1,
                world_seed: 12345,
            };
              transfer::share_object(registry);
            transfer::transfer(admin_cap, admin);
        };
        
        test_scenario::next_tx(scenario, admin);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut registry = test_scenario::take_shared<WorldRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            create_resource_node(
                &admin_cap,
                &mut registry,
                RESOURCE_GOLD,
                100, 200, 50, // coordinates
                1000, // initial yield
                5000, // max yield
                100, // harvest cost
                10, // regeneration rate
                true, // is public
                ctx
            );
            
            assert!(get_total_nodes(&registry) == 1, 0);
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(registry);
        };
        
        test_scenario::end(scenario_val);
    }    #[test]
    public fun test_create_world_region() {
        let admin = @0xAD;
        let mut scenario_val = test_scenario::begin(admin);
        let scenario = &mut scenario_val;
        
        test_scenario::next_tx(scenario, admin);
        {
            let ctx = test_scenario::ctx(scenario);
            let admin_cap = AdminCap { id: object::new(ctx) };
            let registry = WorldRegistry {
                id: object::new(ctx),
                total_regions: 0,
                total_nodes: 0,
                active_events: vector::empty(),
                next_event_id: 1,
                world_seed: 12345,
            };
            
            transfer::share_object(registry);
            transfer::transfer(admin_cap, admin);
        };
          test_scenario::next_tx(scenario, admin);
        {
            let admin_cap = test_scenario::take_from_sender<AdminCap>(scenario);
            let mut registry = test_scenario::take_shared<WorldRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            create_world_region(
                &admin_cap,
                &mut registry,
                b"Test Forest",
                REGION_FOREST,
                0, 1000, // x bounds
                0, 1000, // y bounds
                0, 100,  // z bounds
                b"Temperate",
                3, // danger level
                150, // resource multiplier
                1000, // control fee
                ctx
            );
            
            assert!(get_total_regions(&registry) == 1, 0);
            
            test_scenario::return_to_sender(scenario, admin_cap);
            test_scenario::return_shared(registry);
        };
        
        test_scenario::end(scenario_val);
    }
}
