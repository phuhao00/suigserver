// Sui Move Guild System Contract
// Defines guild NFTs/objects, membership, roles, treasury, etc.

#[allow(duplicate_alias)]
module mmo_game::guild {
    use std::string::{Self, String};
    use sui::object::{Self, ID, UID};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer;
    use std::vector;
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, Coin};
    use sui::sui::SUI;
    use sui::dynamic_field;
    use sui::event;

    // Error codes
    const E_GUILD_ALREADY_EXISTS: u64 = 0;
    const E_NOT_GUILD_LEADER: u64 = 1;
    const E_PLAYER_ALREADY_IN_GUILD: u64 = 2;
    const E_PLAYER_NOT_IN_GUILD: u64 = 3;
    const E_INSUFFICIENT_PERMISSIONS: u64 = 4;
    const E_GUILD_FULL: u64 = 5;
    const E_INVALID_ROLE: u64 = 6;
    const E_CANNOT_REMOVE_LEADER: u64 = 7;
    const E_INSUFFICIENT_FUNDS: u64 = 8;
    const E_INVALID_MEMBER_COUNT: u64 = 9;

    // Role constants
    const ROLE_LEADER: u8 = 0;
    const ROLE_OFFICER: u8 = 1;
    const ROLE_MEMBER: u8 = 2;

    // Guild size limits
    const MAX_GUILD_MEMBERS: u64 = 100;
    const MIN_GUILD_MEMBERS_FOR_UPGRADE: u64 = 10;

    // Guild data structure
    public struct Guild has key, store {
        id: UID,
        name: String,
        leader: address,
        members: vector<address>,
        officers: vector<address>,
        treasury: Balance<SUI>,
        motd: String, // Message of the day
        level: u64,
        experience: u64,
        created_at: u64,
        max_members: u64,
        description: String,
    }    // Member role information stored as dynamic field
    public struct MemberInfo has store, drop {
        role: u8,
        joined_at: u64,
        contribution: u64,
    }

    // Guild registry for global state management
    public struct GuildRegistry has key {
        id: UID,
        guild_names: vector<String>, // Track unique guild names
        total_guilds: u64,
    }

    // Events
    public struct GuildCreated has copy, drop {
        guild_id: ID,
        name: String,
        leader: address,
    }

    public struct MemberJoined has copy, drop {
        guild_id: ID,
        member: address,
        role: u8,
    }

    public struct MemberLeft has copy, drop {
        guild_id: ID,
        member: address,
    }

    public struct RoleChanged has copy, drop {
        guild_id: ID,
        member: address,
        old_role: u8,
        new_role: u8,
    }

    public struct TreasuryDeposit has copy, drop {
        guild_id: ID,
        depositor: address,
        amount: u64,
    }

    public struct TreasuryWithdraw has copy, drop {
        guild_id: ID,
        withdrawer: address,
        amount: u64,
    }    // --- Initialization ---

    // Initialize the guild registry (call once during deployment)
    fun init(ctx: &mut TxContext) {
        let registry = GuildRegistry {
            id: object::new(ctx),
            guild_names: vector::empty(),
            total_guilds: 0,
        };
        transfer::share_object(registry);
    }

    // --- Public Functions ---

    // Create a new guild
    public entry fun create_guild(
        registry: &mut GuildRegistry,
        name: vector<u8>,
        description: vector<u8>,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        let guild_name = string::utf8(name);
        
        // Check if guild name is unique
        assert!(!vector::contains(&registry.guild_names, &guild_name), E_GUILD_ALREADY_EXISTS);
          let guild_id = object::new(ctx);
        let guild_id_copy = object::uid_to_inner(&guild_id);
        
        let mut guild = Guild {
            id: guild_id,
            name: guild_name,
            leader: sender,
            members: vector[sender],
            officers: vector::empty(),
            treasury: balance::zero(),
            motd: string::utf8(b"Welcome to our guild!"),
            level: 1,
            experience: 0,
            created_at: tx_context::epoch(ctx),
            max_members: 25, // Starting capacity
            description: string::utf8(description),
        };

        // Add member info for the leader
        let leader_info = MemberInfo {
            role: ROLE_LEADER,
            joined_at: tx_context::epoch(ctx),
            contribution: 0,
        };
        dynamic_field::add(&mut guild.id, sender, leader_info);

        // Update registry
        vector::push_back(&mut registry.guild_names, guild_name);
        registry.total_guilds = registry.total_guilds + 1;

        // Emit event
        event::emit(GuildCreated {
            guild_id: guild_id_copy,
            name: guild_name,
            leader: sender,
        });

        transfer::transfer(guild, sender);
    }

    // Add a member to the guild
    public entry fun add_member(
        guild: &mut Guild,
        player_address: address,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(can_manage_members(guild, sender), E_INSUFFICIENT_PERMISSIONS);
        assert!(vector::length(&guild.members) < guild.max_members, E_GUILD_FULL);
        assert!(!vector::contains(&guild.members, &player_address), E_PLAYER_ALREADY_IN_GUILD);

        vector::push_back(&mut guild.members, player_address);

        let member_info = MemberInfo {
            role: ROLE_MEMBER,
            joined_at: tx_context::epoch(ctx),
            contribution: 0,
        };
        dynamic_field::add(&mut guild.id, player_address, member_info);

        event::emit(MemberJoined {
            guild_id: object::uid_to_inner(&guild.id),
            member: player_address,
            role: ROLE_MEMBER,
        });
    }

    // Remove a member from the guild
    public entry fun remove_member(
        guild: &mut Guild,
        player_address: address,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(can_manage_members(guild, sender), E_INSUFFICIENT_PERMISSIONS);
        assert!(player_address != guild.leader, E_CANNOT_REMOVE_LEADER);
        assert!(vector::contains(&guild.members, &player_address), E_PLAYER_NOT_IN_GUILD);

        // Remove from members vector
        let (found, index) = vector::index_of(&guild.members, &player_address);
        if (found) {
            vector::remove(&mut guild.members, index);
        };

        // Remove from officers if applicable
        let (found_officer, officer_index) = vector::index_of(&guild.officers, &player_address);
        if (found_officer) {
            vector::remove(&mut guild.officers, officer_index);
        };

        // Remove member info
        if (dynamic_field::exists_(&guild.id, player_address)) {
            let _: MemberInfo = dynamic_field::remove(&mut guild.id, player_address);
        };

        event::emit(MemberLeft {
            guild_id: object::uid_to_inner(&guild.id),
            member: player_address,
        });
    }

    // Leave guild (member can leave voluntarily)
    public entry fun leave_guild(guild: &mut Guild, ctx: &mut TxContext) {
        let sender = tx_context::sender(ctx);
        assert!(sender != guild.leader, E_CANNOT_REMOVE_LEADER); // Leader cannot leave
        assert!(vector::contains(&guild.members, &sender), E_PLAYER_NOT_IN_GUILD);

        // Remove from members vector
        let (found, index) = vector::index_of(&guild.members, &sender);
        if (found) {
            vector::remove(&mut guild.members, index);
        };

        // Remove from officers if applicable
        let (found_officer, officer_index) = vector::index_of(&guild.officers, &sender);
        if (found_officer) {
            vector::remove(&mut guild.officers, officer_index);
        };

        // Remove member info
        if (dynamic_field::exists_(&guild.id, sender)) {
            let _: MemberInfo = dynamic_field::remove(&mut guild.id, sender);
        };

        event::emit(MemberLeft {
            guild_id: object::uid_to_inner(&guild.id),
            member: sender,
        });
    }

    // Promote a member to officer
    public entry fun promote_to_officer(
        guild: &mut Guild,
        player_address: address,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(sender == guild.leader, E_NOT_GUILD_LEADER);
        assert!(vector::contains(&guild.members, &player_address), E_PLAYER_NOT_IN_GUILD);
        assert!(!vector::contains(&guild.officers, &player_address), E_INVALID_ROLE);

        vector::push_back(&mut guild.officers, player_address);

        // Update member info
        if (dynamic_field::exists_(&guild.id, player_address)) {
            let member_info = dynamic_field::borrow_mut<address, MemberInfo>(&mut guild.id, player_address);
            let old_role = member_info.role;
            member_info.role = ROLE_OFFICER;

            event::emit(RoleChanged {
                guild_id: object::uid_to_inner(&guild.id),
                member: player_address,
                old_role,
                new_role: ROLE_OFFICER,
            });
        };
    }

    // Demote an officer to member
    public entry fun demote_officer(
        guild: &mut Guild,
        player_address: address,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(sender == guild.leader, E_NOT_GUILD_LEADER);
        assert!(vector::contains(&guild.officers, &player_address), E_INVALID_ROLE);

        // Remove from officers
        let (found, index) = vector::index_of(&guild.officers, &player_address);
        if (found) {
            vector::remove(&mut guild.officers, index);
        };

        // Update member info
        if (dynamic_field::exists_(&guild.id, player_address)) {
            let member_info = dynamic_field::borrow_mut<address, MemberInfo>(&mut guild.id, player_address);
            let old_role = member_info.role;
            member_info.role = ROLE_MEMBER;

            event::emit(RoleChanged {
                guild_id: object::uid_to_inner(&guild.id),
                member: player_address,
                old_role,
                new_role: ROLE_MEMBER,
            });
        };
    }

    // Transfer guild leadership
    public entry fun transfer_leadership(
        guild: &mut Guild,
        new_leader: address,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(sender == guild.leader, E_NOT_GUILD_LEADER);
        assert!(vector::contains(&guild.members, &new_leader), E_PLAYER_NOT_IN_GUILD);

        let old_leader = guild.leader;
        guild.leader = new_leader;

        // Update old leader's role to member
        if (dynamic_field::exists_(&guild.id, old_leader)) {
            let old_leader_info = dynamic_field::borrow_mut<address, MemberInfo>(&mut guild.id, old_leader);
            old_leader_info.role = ROLE_MEMBER;
        };

        // Update new leader's role
        if (dynamic_field::exists_(&guild.id, new_leader)) {
            let new_leader_info = dynamic_field::borrow_mut<address, MemberInfo>(&mut guild.id, new_leader);
            new_leader_info.role = ROLE_LEADER;
        };

        // Remove new leader from officers if they were one
        let (found_officer, officer_index) = vector::index_of(&guild.officers, &new_leader);
        if (found_officer) {
            vector::remove(&mut guild.officers, officer_index);
        };

        event::emit(RoleChanged {
            guild_id: object::uid_to_inner(&guild.id),
            member: new_leader,
            old_role: ROLE_MEMBER, // or ROLE_OFFICER
            new_role: ROLE_LEADER,
        });
    }

    // --- Treasury Management ---

    // Deposit funds to guild treasury
    public entry fun deposit_to_treasury(
        guild: &mut Guild,
        payment: Coin<SUI>,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(vector::contains(&guild.members, &sender), E_PLAYER_NOT_IN_GUILD);

        let amount = coin::value(&payment);
        let deposit_balance = coin::into_balance(payment);
        balance::join(&mut guild.treasury, deposit_balance);

        // Update member contribution
        if (dynamic_field::exists_(&guild.id, sender)) {
            let member_info = dynamic_field::borrow_mut<address, MemberInfo>(&mut guild.id, sender);
            member_info.contribution = member_info.contribution + amount;
        };

        event::emit(TreasuryDeposit {
            guild_id: object::uid_to_inner(&guild.id),
            depositor: sender,
            amount,
        });
    }

    // Withdraw funds from guild treasury (leader only)
    public entry fun withdraw_from_treasury(
        guild: &mut Guild,
        amount: u64,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(sender == guild.leader, E_NOT_GUILD_LEADER);
        assert!(balance::value(&guild.treasury) >= amount, E_INSUFFICIENT_FUNDS);

        let withdrawal = coin::take(&mut guild.treasury, amount, ctx);
        transfer::public_transfer(withdrawal, sender);

        event::emit(TreasuryWithdraw {
            guild_id: object::uid_to_inner(&guild.id),
            withdrawer: sender,
            amount,
        });
    }

    // --- Guild Management ---

    // Update message of the day
    public entry fun update_motd(
        guild: &mut Guild,
        new_motd: vector<u8>,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(can_manage_guild(guild, sender), E_INSUFFICIENT_PERMISSIONS);
        guild.motd = string::utf8(new_motd);
    }

    // Update guild description
    public entry fun update_description(
        guild: &mut Guild,
        new_description: vector<u8>,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(sender == guild.leader, E_NOT_GUILD_LEADER);
        guild.description = string::utf8(new_description);
    }

    // Upgrade guild (increase member capacity)
    public entry fun upgrade_guild(
        guild: &mut Guild,
        payment: Coin<SUI>,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(sender == guild.leader, E_NOT_GUILD_LEADER);
        assert!(vector::length(&guild.members) >= MIN_GUILD_MEMBERS_FOR_UPGRADE, E_INVALID_MEMBER_COUNT);

        let upgrade_cost = get_upgrade_cost(guild);
        assert!(coin::value(&payment) >= upgrade_cost, E_INSUFFICIENT_FUNDS);

        // Consume the payment
        let payment_balance = coin::into_balance(payment);
        balance::join(&mut guild.treasury, payment_balance);

        // Upgrade guild
        guild.level = guild.level + 1;
        guild.max_members = guild.max_members + 25; // Increase capacity by 25
        
        if (guild.max_members > MAX_GUILD_MEMBERS) {
            guild.max_members = MAX_GUILD_MEMBERS;
        };
    }

    // Add experience to guild
    public entry fun add_guild_experience(
        guild: &mut Guild,
        experience_points: u64,
        ctx: &mut TxContext
    ) {
        let sender = tx_context::sender(ctx);
        assert!(vector::contains(&guild.members, &sender), E_PLAYER_NOT_IN_GUILD);
        
        guild.experience = guild.experience + experience_points;
        
        // Auto-level up if enough experience
        let required_exp = guild.level * 1000; // 1000 exp per level
        if (guild.experience >= required_exp) {
            guild.level = guild.level + 1;
            guild.experience = guild.experience - required_exp;
        };
    }

    // --- Helper Functions ---

    // Check if member can manage other members
    fun can_manage_members(guild: &Guild, member: address): bool {
        member == guild.leader || vector::contains(&guild.officers, &member)
    }

    // Check if member can manage guild settings
    fun can_manage_guild(guild: &Guild, member: address): bool {
        member == guild.leader || vector::contains(&guild.officers, &member)
    }

    // Get upgrade cost for next level
    public fun get_upgrade_cost(guild: &Guild): u64 {
        guild.level * 1000000000 // 1 SUI per level (in MIST)
    }

    // --- Getter Functions ---

    public fun get_guild_id(guild: &Guild): ID {
        object::uid_to_inner(&guild.id)
    }

    public fun get_guild_name(guild: &Guild): String {
        guild.name
    }

    public fun get_guild_leader(guild: &Guild): address {
        guild.leader
    }

    public fun get_guild_description(guild: &Guild): String {
        guild.description
    }

    public fun get_guild_motd(guild: &Guild): String {
        guild.motd
    }

    public fun get_guild_level(guild: &Guild): u64 {
        guild.level
    }

    public fun get_guild_experience(guild: &Guild): u64 {
        guild.experience
    }

    public fun get_member_count(guild: &Guild): u64 {
        vector::length(&guild.members)
    }

    public fun get_max_members(guild: &Guild): u64 {
        guild.max_members
    }

    public fun get_officer_count(guild: &Guild): u64 {
        vector::length(&guild.officers)
    }

    public fun get_treasury_balance(guild: &Guild): u64 {
        balance::value(&guild.treasury)
    }

    public fun get_created_at(guild: &Guild): u64 {
        guild.created_at
    }

    public fun is_member(guild: &Guild, player: address): bool {
        vector::contains(&guild.members, &player)
    }

    public fun is_officer(guild: &Guild, player: address): bool {
        vector::contains(&guild.officers, &player)
    }

    public fun is_leader(guild: &Guild, player: address): bool {
        guild.leader == player
    }

    public fun get_member_role(guild: &Guild, player: address): u8 {
        if (guild.leader == player) {
            ROLE_LEADER
        } else if (vector::contains(&guild.officers, &player)) {
            ROLE_OFFICER
        } else {
            ROLE_MEMBER
        }
    }

    public fun get_member_info(guild: &Guild, player: address): (u8, u64, u64) {
        if (dynamic_field::exists_(&guild.id, player)) {
            let member_info = dynamic_field::borrow<address, MemberInfo>(&guild.id, player);
            (member_info.role, member_info.joined_at, member_info.contribution)
        } else {
            (255, 0, 0) // Invalid values if not found
        }
    }

    public fun get_members(guild: &Guild): vector<address> {
        guild.members
    }

    public fun get_officers(guild: &Guild): vector<address> {
        guild.officers
    }    // Registry getters
    public fun get_total_guilds(registry: &GuildRegistry): u64 {
        registry.total_guilds
    }

    public fun get_guild_names(registry: &GuildRegistry): vector<String> {
        registry.guild_names
    }

    // --- Test Helper Functions ---
    #[test_only]
    public fun create_registry_for_testing(ctx: &mut TxContext): GuildRegistry {
        GuildRegistry {
            id: object::new(ctx),
            guild_names: vector::empty(),
            total_guilds: 0,
        }
    }

    // --- Test Functions ---
    #[test_only]
    use sui::test_scenario;
    #[test_only]
    use sui::test_utils;

    #[test]
    public fun test_create_guild() {
        let admin = @0xAD;
        let player1 = @0x123;
        
        let scenario_val = test_scenario::begin(admin);
        let scenario = &mut scenario_val;
        
        // Initialize registry
        test_scenario::next_tx(scenario, admin);
        {
            let ctx = test_scenario::ctx(scenario);
            let registry = GuildRegistry {
                id: object::new(ctx),
                guild_names: vector::empty(),
                total_guilds: 0,
            };
            transfer::share_object(registry);
        };
        
        // Create guild
        test_scenario::next_tx(scenario, player1);
        {
            let registry = test_scenario::take_shared<GuildRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            create_guild(
                &mut registry,
                b"Test Guild",
                b"A test guild for testing",
                ctx
            );
            
            test_scenario::return_shared(registry);
        };
        
        // Check guild was created
        test_scenario::next_tx(scenario, player1);
        {
            let guild = test_scenario::take_from_sender<Guild>(scenario);
            
            assert!(get_guild_name(&guild) == string::utf8(b"Test Guild"), 0);
            assert!(get_guild_leader(&guild) == player1, 1);
            assert!(get_member_count(&guild) == 1, 2);
            assert!(is_member(&guild, player1), 3);
            assert!(is_leader(&guild, player1), 4);
            assert!(get_guild_level(&guild) == 1, 5);
            
            test_scenario::return_to_sender(scenario, guild);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    public fun test_add_remove_member() {
        let admin = @0xAD;
        let leader = @0x123;
        let member = @0x456;
        
        let scenario_val = test_scenario::begin(admin);
        let scenario = &mut scenario_val;
        
        // Setup registry and guild
        test_scenario::next_tx(scenario, admin);
        {
            let ctx = test_scenario::ctx(scenario);
            let registry = GuildRegistry {
                id: object::new(ctx),
                guild_names: vector::empty(),
                total_guilds: 0,
            };
            transfer::share_object(registry);
        };
        
        test_scenario::next_tx(scenario, leader);
        {
            let registry = test_scenario::take_shared<GuildRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            create_guild(
                &mut registry,
                b"Test Guild",
                b"A test guild",
                ctx
            );
            
            test_scenario::return_shared(registry);
        };
        
        // Add member
        test_scenario::next_tx(scenario, leader);
        {
            let guild = test_scenario::take_from_sender<Guild>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            add_member(&mut guild, member, ctx);
            
            assert!(get_member_count(&guild) == 2, 0);
            assert!(is_member(&guild, member), 1);
            assert!(!is_leader(&guild, member), 2);
            
            test_scenario::return_to_sender(scenario, guild);
        };
        
        // Remove member
        test_scenario::next_tx(scenario, leader);
        {
            let guild = test_scenario::take_from_sender<Guild>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            remove_member(&mut guild, member, ctx);
            
            assert!(get_member_count(&guild) == 1, 3);
            assert!(!is_member(&guild, member), 4);
            
            test_scenario::return_to_sender(scenario, guild);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    public fun test_promote_demote_officer() {
        let admin = @0xAD;
        let leader = @0x123;
        let member = @0x456;
        
        let scenario_val = test_scenario::begin(admin);
        let scenario = &mut scenario_val;
        
        // Setup
        test_scenario::next_tx(scenario, admin);
        {
            let ctx = test_scenario::ctx(scenario);
            let registry = GuildRegistry {
                id: object::new(ctx),
                guild_names: vector::empty(),
                total_guilds: 0,
            };
            transfer::share_object(registry);
        };
        
        test_scenario::next_tx(scenario, leader);
        {
            let registry = test_scenario::take_shared<GuildRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            create_guild(&mut registry, b"Test Guild", b"Test", ctx);
            test_scenario::return_shared(registry);
        };
        
        test_scenario::next_tx(scenario, leader);
        {
            let guild = test_scenario::take_from_sender<Guild>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            add_member(&mut guild, member, ctx);
            test_scenario::return_to_sender(scenario, guild);
        };
        
        // Promote to officer
        test_scenario::next_tx(scenario, leader);
        {
            let guild = test_scenario::take_from_sender<Guild>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            promote_to_officer(&mut guild, member, ctx);
            
            assert!(is_officer(&guild, member), 0);
            assert!(get_officer_count(&guild) == 1, 1);
            assert!(get_member_role(&guild, member) == ROLE_OFFICER, 2);
            
            test_scenario::return_to_sender(scenario, guild);
        };
        
        // Demote officer
        test_scenario::next_tx(scenario, leader);
        {
            let guild = test_scenario::take_from_sender<Guild>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            demote_officer(&mut guild, member, ctx);
            
            assert!(!is_officer(&guild, member), 3);
            assert!(get_officer_count(&guild) == 0, 4);
            assert!(get_member_role(&guild, member) == ROLE_MEMBER, 5);
            
            test_scenario::return_to_sender(scenario, guild);
        };
        
        test_scenario::end(scenario_val);
    }

    #[test]
    public fun test_treasury_operations() {
        let admin = @0xAD;
        let leader = @0x123;
        
        let scenario_val = test_scenario::begin(admin);
        let scenario = &mut scenario_val;
        
        // Setup
        test_scenario::next_tx(scenario, admin);
        {
            let ctx = test_scenario::ctx(scenario);
            let registry = GuildRegistry {
                id: object::new(ctx),
                guild_names: vector::empty(),
                total_guilds: 0,
            };
            transfer::share_object(registry);
        };
        
        test_scenario::next_tx(scenario, leader);
        {
            let registry = test_scenario::take_shared<GuildRegistry>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            create_guild(&mut registry, b"Test Guild", b"Test", ctx);
            test_scenario::return_shared(registry);
        };
        
        // Deposit to treasury
        test_scenario::next_tx(scenario, leader);
        {
            let guild = test_scenario::take_from_sender<Guild>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            let payment = coin::mint_for_testing<SUI>(1000000000, ctx); // 1 SUI
            deposit_to_treasury(&mut guild, payment, ctx);
            
            assert!(get_treasury_balance(&guild) == 1000000000, 0);
            
            test_scenario::return_to_sender(scenario, guild);
        };
        
        // Withdraw from treasury
        test_scenario::next_tx(scenario, leader);
        {
            let guild = test_scenario::take_from_sender<Guild>(scenario);
            let ctx = test_scenario::ctx(scenario);
            
            withdraw_from_treasury(&mut guild, 500000000, ctx); // 0.5 SUI
            
            assert!(get_treasury_balance(&guild) == 500000000, 1);
            
            test_scenario::return_to_sender(scenario, guild);
        };
        
        test_scenario::end(scenario_val);
    }
}
