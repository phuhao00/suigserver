// Combat System Contract for Sui MMO Game
// Handles combat logic, results recording, PvP/PvE interactions.
// Complex real-time calculations happen off-chain on the Go server.

module mmo_game::combat {
    use std::string::{Self, String};
    use std::vector;
    use std::option::{Self, Option};
    use sui::object::{Self, UID, ID};
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};
    use sui::event;    use sui::table::{Self, Table};
    use sui::balance::{Self, Balance};
    use sui::coin::{Self, Coin};
    use sui::sui::SUI;

    // Error codes
    const E_NOT_AUTHORIZED: u64 = 1;
    const E_INVALID_COMBAT_ID: u64 = 2;
    const E_COMBAT_ALREADY_EXISTS: u64 = 3;
    const E_INSUFFICIENT_REWARDS: u64 = 4;
    const E_INVALID_PARTICIPANT: u64 = 5;
    const E_INVALID_COMBAT_TYPE: u64 = 6;

    /// Admin capability for managing combat system
    public struct AdminCap has key, store { 
        id: UID 
    }

    /// Combat outcome types
    const COMBAT_TYPE_PVP: u8 = 1;
    const COMBAT_TYPE_PVE: u8 = 2;
    const COMBAT_TYPE_GUILD_WAR: u8 = 3;

    /// Combat status
    const STATUS_INITIATED: u8 = 0;
    const STATUS_IN_PROGRESS: u8 = 1;
    const STATUS_FINISHED: u8 = 2;
    const STATUS_CANCELLED: u8 = 3;

    /// Combat session data
    public struct CombatSession has key, store {
        id: UID,
        combat_id: u64,
        combat_type: u8,
        participants: vector<address>,
        status: u8,
        start_time: u64,
        end_time: Option<u64>,
        winner: Option<address>,
        loser: Option<address>,
        damage_dealt: Table<address, u64>,
        healing_done: Table<address, u64>,
        rewards_pool: Balance<SUI>,
        metadata: String, // JSON string for additional data
    }

    /// Combat reward distribution
    public struct RewardDistribution has copy, drop, store {
        recipient: address,
        sui_amount: u64,
        experience_points: u64,
        item_type_ids: vector<u64>,
        special_rewards: vector<String>,
    }

    /// Combat statistics tracking
    public struct CombatStats has key {
        id: UID,
        player_address: address,
        total_combats: u64,
        wins: u64,
        losses: u64,
        damage_dealt_total: u64,
        damage_taken_total: u64,
        healing_done_total: u64,
        pvp_rating: u64,
        last_combat_time: u64,
    }

    /// Combat registry for global state management
    public struct CombatRegistry has key {
        id: UID,
        total_combats: u64,
        active_combats: vector<u64>,
        combat_history: Table<u64, ID>, // combat_id -> CombatSession ID
        player_stats: Table<address, ID>, // player -> CombatStats ID
    }

    /// Combat participant info
    public struct ParticipantInfo has store {
        player_address: address,
        initial_health: u64,
        current_health: u64,
        initial_mana: u64,
        current_mana: u64,
        equipment_power: u64,
        buffs_active: vector<String>,
        status_effects: vector<String>,
    }

    /// Combat action for detailed tracking
    public struct CombatAction has store {
        actor: address,
        action_type: String, // "attack", "heal", "spell", "item_use"
        target: Option<address>,
        value: u64, // damage/healing amount
        timestamp: u64,
        success: bool,
        critical_hit: bool,
        additional_effects: vector<String>,
    }

    /// Enhanced combat session with detailed tracking
    public struct DetailedCombatSession has key, store {
        id: UID,
        combat_id: u64,
        combat_type: u8,
        participants_info: Table<address, ParticipantInfo>,
        combat_actions: vector<CombatAction>,
        status: u8,
        start_time: u64,
        end_time: Option<u64>,
        winner: Option<address>,
        rewards_pool: Balance<SUI>,
        metadata: String,
        turn_order: vector<address>,
        current_turn: u64,
        max_turns: u64,
        environment_effects: vector<String>,
    }

    // === Events ===

    /// Event for combat initiation
    public struct CombatInitiated has copy, drop {
        combat_id: u64,
        combat_type: u8,
        participants: vector<address>,
        start_time: u64,
    }

    /// Event for combat outcome
    public struct CombatOutcome has copy, drop {
        combat_id: u64,
        combat_type: u8,
        winner_address: Option<address>,
        loser_address: Option<address>,
        duration: u64,
        total_damage: u64,
        rewards_distributed: vector<RewardDistribution>,
    }

    /// Event for damage dealt
    public struct DamageEvent has copy, drop {
        combat_id: u64,
        attacker: address,
        target: address,
        damage: u64,
        damage_type: String,
        timestamp: u64,
    }

    /// Event for healing
    public struct HealingEvent has copy, drop {
        combat_id: u64,
        healer: address,
        target: address,
        healing: u64,
        timestamp: u64,
    }

    /// Event for combat action
    public struct CombatActionEvent has copy, drop {
        combat_id: u64,
        actor: address,
        action_type: String,
        target: Option<address>,
        value: u64,
        success: bool,
        critical_hit: bool,
        turn_number: u64,
    }

    /// Event for status effect changes
    public struct StatusEffectEvent has copy, drop {
        combat_id: u64,
        target: address,
        effect_type: String,
        applied: bool, // true for apply, false for remove
        duration: u64,
    }

    /// Event for turn changes
    public struct TurnChangeEvent has copy, drop {
        combat_id: u64,
        current_player: address,
        turn_number: u64,
        time_limit: u64,
    }

    /// Initialize the combat system
    fun init(ctx: &mut TxContext) {
        let admin_cap = AdminCap { 
            id: object::new(ctx)
        };

        let registry = CombatRegistry {
            id: object::new(ctx),
            total_combats: 0,
            active_combats: vector::empty(),
            combat_history: table::new(ctx),
            player_stats: table::new(ctx),
        };

        transfer::transfer(admin_cap, tx_context::sender(ctx));
        transfer::share_object(registry);
    }

    /// Initiate a new combat session
    public entry fun initiate_combat(
        _admin_cap: &AdminCap,
        combat_id: u64,
        combat_type: u8,
        participants: vector<address>,
        reward_amount: Coin<SUI>,
        metadata: vector<u8>,
        ctx: &mut TxContext
    ) {
        assert!(vector::length(&participants) >= 1, E_INVALID_PARTICIPANT);
        assert!(combat_type >= COMBAT_TYPE_PVP && combat_type <= COMBAT_TYPE_GUILD_WAR, E_INVALID_COMBAT_TYPE);

        let combat_session = CombatSession {
            id: object::new(ctx),
            combat_id,
            combat_type,
            participants: participants,
            status: STATUS_INITIATED,
            start_time: tx_context::epoch(ctx),
            end_time: option::none(),
            winner: option::none(),
            loser: option::none(),
            damage_dealt: table::new(ctx),
            healing_done: table::new(ctx),
            rewards_pool: coin::into_balance(reward_amount),
            metadata: string::utf8(metadata),
        };

        let participants_copy = combat_session.participants;
        transfer::share_object(combat_session);        event::emit(CombatInitiated {
            combat_id,
            combat_type,
            participants: participants_copy,
            start_time: tx_context::epoch(ctx),
        });
    }

    /// Record damage dealt during combat
    public entry fun record_damage(
        _admin_cap: &AdminCap,
        combat_session: &mut CombatSession,
        attacker: address,
        target: address,
        damage: u64,
        damage_type: vector<u8>,
        ctx: &mut TxContext
    ) {
        assert!(combat_session.status == STATUS_IN_PROGRESS, E_INVALID_COMBAT_ID);
        assert!(vector::contains(&combat_session.participants, &attacker), E_INVALID_PARTICIPANT);
        
        if (table::contains(&combat_session.damage_dealt, attacker)) {
            let current_damage = table::borrow_mut(&mut combat_session.damage_dealt, attacker);
            *current_damage = *current_damage + damage;
        } else {
            table::add(&mut combat_session.damage_dealt, attacker, damage);
        };        event::emit(DamageEvent {
            combat_id: combat_session.combat_id,
            attacker,
            target,
            damage,
            damage_type: string::utf8(damage_type),
            timestamp: tx_context::epoch(ctx),
        });
    }

    /// Record healing done during combat
    public entry fun record_healing(
        _admin_cap: &AdminCap,
        combat_session: &mut CombatSession,
        healer: address,
        target: address,
        healing: u64,
        ctx: &mut TxContext
    ) {
        assert!(combat_session.status == STATUS_IN_PROGRESS, E_INVALID_COMBAT_ID);
        assert!(vector::contains(&combat_session.participants, &healer), E_INVALID_PARTICIPANT);

        if (table::contains(&combat_session.healing_done, healer)) {
            let current_healing = table::borrow_mut(&mut combat_session.healing_done, healer);
            *current_healing = *current_healing + healing;
        } else {
            table::add(&mut combat_session.healing_done, healer, healing);
        };        event::emit(HealingEvent {
            combat_id: combat_session.combat_id,
            healer,
            target,
            healing,
            timestamp: tx_context::epoch(ctx),
        });
    }    /// Finalize combat and distribute rewards
    public fun finalize_combat(
        _admin_cap: &AdminCap,
        combat_session: &mut CombatSession,
        winner: Option<address>,
        loser: Option<address>,
        reward_distributions: vector<RewardDistribution>,
        ctx: &mut TxContext
    ) {
        assert!(combat_session.status == STATUS_IN_PROGRESS, E_INVALID_COMBAT_ID);

        combat_session.status = STATUS_FINISHED;
        combat_session.end_time = option::some(tx_context::epoch(ctx));
        combat_session.winner = winner;
        combat_session.loser = loser;

        let total_damage = calculate_total_damage(combat_session);
        let duration = tx_context::epoch(ctx) - combat_session.start_time;        // Distribute SUI rewards
        let mut i = 0;
        let reward_distributions_copy = reward_distributions;
        while (i < vector::length(&reward_distributions_copy)) {
            let reward = vector::borrow(&reward_distributions_copy, i);
            if (reward.sui_amount > 0) {
                let current_balance = balance::value(&combat_session.rewards_pool);
                assert!(current_balance >= reward.sui_amount, E_INSUFFICIENT_REWARDS);
                let reward_coin = coin::take(&mut combat_session.rewards_pool, reward.sui_amount, ctx);
                transfer::public_transfer(reward_coin, reward.recipient);
            };
            i = i + 1;
        };

        event::emit(CombatOutcome {
            combat_id: combat_session.combat_id,
            combat_type: combat_session.combat_type,
            winner_address: winner,
            loser_address: loser,
            duration,            total_damage,
            rewards_distributed: reward_distributions,
        });
    }

    /// Entry wrapper for finalize_combat with primitive parameters
    public entry fun finalize_combat_entry(
        _admin_cap: &AdminCap,
        combat_session: &mut CombatSession,
        winner: Option<address>,
        loser: Option<address>,
        recipients: vector<address>,
        sui_amounts: vector<u64>,
        ctx: &mut TxContext
    ) {
        assert!(vector::length(&recipients) == vector::length(&sui_amounts), E_INVALID_PARTICIPANT);
        
        // Convert primitive parameters to RewardDistribution vector
        let mut reward_distributions = vector::empty<RewardDistribution>();
        let mut i = 0;
        while (i < vector::length(&recipients)) {
            let recipient = *vector::borrow(&recipients, i);
            let sui_amount = *vector::borrow(&sui_amounts, i);
            
            let reward = RewardDistribution {
                recipient,
                sui_amount,
                experience_points: 0, // Can be extended later
                item_type_ids: vector::empty(),
                special_rewards: vector::empty(),
            };
            
            vector::push_back(&mut reward_distributions, reward);
            i = i + 1;
        };
        
        finalize_combat(_admin_cap, combat_session, winner, loser, reward_distributions, ctx);
    }

    /// Start combat (change status to in progress)
    public entry fun start_combat(
        _admin_cap: &AdminCap,
        combat_session: &mut CombatSession,
        _ctx: &mut TxContext    ) {
        assert!(combat_session.status == STATUS_INITIATED, E_INVALID_COMBAT_ID);
        combat_session.status = STATUS_IN_PROGRESS;
    }

    /// Cancel combat
    public entry fun cancel_combat(
        _admin_cap: &AdminCap,
        combat_session: &mut CombatSession,
        ctx: &mut TxContext
    ) {
        assert!(combat_session.status != STATUS_FINISHED, E_INVALID_COMBAT_ID);
        
        combat_session.status = STATUS_CANCELLED;
        combat_session.end_time = option::some(tx_context::epoch(ctx));

        // Return rewards to participants equally
        let total_balance = balance::value(&combat_session.rewards_pool);
        if (total_balance > 0) {
            let participant_count = vector::length(&combat_session.participants);            if (participant_count > 0) {
                let reward_per_participant = total_balance / participant_count;
                
                let mut i = 0;
                while (i < participant_count) {
                    let participant = *vector::borrow(&combat_session.participants, i);
                    if (reward_per_participant > 0) {
                        let refund = coin::take(&mut combat_session.rewards_pool, reward_per_participant, ctx);
                        transfer::public_transfer(refund, participant);
                    };
                    i = i + 1;
                };
            };
        };
    }

    /// Initialize or update combat stats for a player
    public entry fun update_combat_stats(
        player_address: address,
        wins_delta: u64,
        losses_delta: u64,
        damage_dealt: u64,
        damage_taken: u64,
        healing_done: u64,
        ctx: &mut TxContext
    ) {
        // This would typically be called by the game server
        // In a real implementation, you'd need proper authorization
        
        let stats = CombatStats {
            id: object::new(ctx),
            player_address,
            total_combats: wins_delta + losses_delta,
            wins: wins_delta,
            losses: losses_delta,
            damage_dealt_total: damage_dealt,
            damage_taken_total: damage_taken,
            healing_done_total: healing_done,
            pvp_rating: calculate_pvp_rating(wins_delta, losses_delta),
            last_combat_time: tx_context::epoch(ctx),
        };

        transfer::transfer(stats, player_address);
    }

    /// Create a detailed combat session with participant info
    public entry fun create_detailed_combat(
        _admin_cap: &AdminCap,
        registry: &mut CombatRegistry,
        combat_id: u64,
        combat_type: u8,
        participants: vector<address>,
        participant_health: vector<u64>,
        participant_mana: vector<u64>,
        participant_power: vector<u64>,
        reward_amount: Coin<SUI>,
        max_turns: u64,
        metadata: vector<u8>,
        ctx: &mut TxContext
    ) {
        assert!(vector::length(&participants) >= 1, E_INVALID_PARTICIPANT);
        assert!(vector::length(&participants) == vector::length(&participant_health), E_INVALID_PARTICIPANT);
        assert!(vector::length(&participants) == vector::length(&participant_mana), E_INVALID_PARTICIPANT);
        assert!(vector::length(&participants) == vector::length(&participant_power), E_INVALID_PARTICIPANT);
        assert!(combat_type >= COMBAT_TYPE_PVP && combat_type <= COMBAT_TYPE_GUILD_WAR, E_INVALID_COMBAT_TYPE);        let session_id = object::new(ctx);
        let session_id_inner = object::uid_to_inner(&session_id);

        let mut participants_info = table::new(ctx);
        let mut i = 0;
        while (i < vector::length(&participants)) {
            let participant = *vector::borrow(&participants, i);
            let health = *vector::borrow(&participant_health, i);
            let mana = *vector::borrow(&participant_mana, i);
            let power = *vector::borrow(&participant_power, i);

            let info = ParticipantInfo {
                player_address: participant,
                initial_health: health,
                current_health: health,
                initial_mana: mana,
                current_mana: mana,
                equipment_power: power,
                buffs_active: vector::empty(),
                status_effects: vector::empty(),
            };

            table::add(&mut participants_info, participant, info);
            i = i + 1;
        };

        let detailed_session = DetailedCombatSession {
            id: session_id,
            combat_id,
            combat_type,
            participants_info,
            combat_actions: vector::empty(),
            status: STATUS_INITIATED,
            start_time: tx_context::epoch(ctx),
            end_time: option::none(),
            winner: option::none(),
            rewards_pool: coin::into_balance(reward_amount),
            metadata: string::utf8(metadata),
            turn_order: participants,
            current_turn: 0,
            max_turns,
            environment_effects: vector::empty(),
        };

        // Update registry
        registry.total_combats = registry.total_combats + 1;
        vector::push_back(&mut registry.active_combats, combat_id);
        table::add(&mut registry.combat_history, combat_id, session_id_inner);

        let participants_copy = detailed_session.turn_order;
        transfer::share_object(detailed_session);

        event::emit(CombatInitiated {
            combat_id,
            combat_type,
            participants: participants_copy,
            start_time: tx_context::epoch(ctx),
        });
    }

    /// Execute a combat action
    public entry fun execute_combat_action(
        _admin_cap: &AdminCap,
        detailed_session: &mut DetailedCombatSession,
        actor: address,
        action_type: vector<u8>,
        target: Option<address>,
        value: u64,
        success: bool,
        critical_hit: bool,
        additional_effects: vector<String>,
        ctx: &mut TxContext
    ) {
        assert!(detailed_session.status == STATUS_IN_PROGRESS, E_INVALID_COMBAT_ID);
        assert!(table::contains(&detailed_session.participants_info, actor), E_INVALID_PARTICIPANT);

        let action = CombatAction {
            actor,
            action_type: string::utf8(action_type),
            target,
            value,
            timestamp: tx_context::epoch(ctx),
            success,
            critical_hit,
            additional_effects,
        };

        vector::push_back(&mut detailed_session.combat_actions, action);

        // Apply action effects
        if (success) {
            let action_str = string::utf8(action_type);
            if (action_str == string::utf8(b"attack") && option::is_some(&target)) {
                let target_addr = *option::borrow(&target);
                if (table::contains(&detailed_session.participants_info, target_addr)) {
                    let target_info = table::borrow_mut(&mut detailed_session.participants_info, target_addr);
                    if (target_info.current_health >= value) {
                        target_info.current_health = target_info.current_health - value;
                    } else {
                        target_info.current_health = 0;
                    };
                };
            } else if (action_str == string::utf8(b"heal") && option::is_some(&target)) {
                let target_addr = *option::borrow(&target);
                if (table::contains(&detailed_session.participants_info, target_addr)) {
                    let target_info = table::borrow_mut(&mut detailed_session.participants_info, target_addr);
                    target_info.current_health = target_info.current_health + value;
                    if (target_info.current_health > target_info.initial_health) {
                        target_info.current_health = target_info.initial_health;
                    };
                };
            };
        };

        event::emit(CombatActionEvent {
            combat_id: detailed_session.combat_id,
            actor,
            action_type: string::utf8(action_type),
            target,
            value,
            success,
            critical_hit,
            turn_number: detailed_session.current_turn,
        });
    }

    /// Advance to next turn
    public entry fun advance_turn(
        _admin_cap: &AdminCap,
        detailed_session: &mut DetailedCombatSession,
        ctx: &mut TxContext
    ) {
        assert!(detailed_session.status == STATUS_IN_PROGRESS, E_INVALID_COMBAT_ID);

        detailed_session.current_turn = detailed_session.current_turn + 1;
        
        // Check if combat should end due to turn limit
        if (detailed_session.current_turn >= detailed_session.max_turns) {
            detailed_session.status = STATUS_FINISHED;
            detailed_session.end_time = option::some(tx_context::epoch(ctx));
            return
        };

        let current_player_index = (detailed_session.current_turn % vector::length(&detailed_session.turn_order));
        let current_player = *vector::borrow(&detailed_session.turn_order, current_player_index);

        event::emit(TurnChangeEvent {
            combat_id: detailed_session.combat_id,
            current_player,
            turn_number: detailed_session.current_turn,
            time_limit: 30000, // 30 seconds per turn
        });
    }

    /// Apply status effect to participant
    public entry fun apply_status_effect(
        _admin_cap: &AdminCap,
        detailed_session: &mut DetailedCombatSession,
        target: address,
        effect_type: vector<u8>,
        duration: u64,
        _ctx: &mut TxContext
    ) {
        assert!(detailed_session.status == STATUS_IN_PROGRESS, E_INVALID_COMBAT_ID);
        assert!(table::contains(&detailed_session.participants_info, target), E_INVALID_PARTICIPANT);

        let target_info = table::borrow_mut(&mut detailed_session.participants_info, target);
        let effect_str = string::utf8(effect_type);
        vector::push_back(&mut target_info.status_effects, effect_str);

        event::emit(StatusEffectEvent {
            combat_id: detailed_session.combat_id,
            target,
            effect_type: effect_str,
            applied: true,
            duration,
        });
    }

    /// Remove status effect from participant
    public entry fun remove_status_effect(
        _admin_cap: &AdminCap,
        detailed_session: &mut DetailedCombatSession,
        target: address,
        effect_type: vector<u8>,
        _ctx: &mut TxContext
    ) {
        assert!(detailed_session.status == STATUS_IN_PROGRESS, E_INVALID_COMBAT_ID);
        assert!(table::contains(&detailed_session.participants_info, target), E_INVALID_PARTICIPANT);

        let target_info = table::borrow_mut(&mut detailed_session.participants_info, target);
        let effect_str = string::utf8(effect_type);
          // Remove effect from status_effects vector
        let mut i = 0;
        let mut found = false;
        while (i < vector::length(&target_info.status_effects) && !found) {
            let effect = vector::borrow(&target_info.status_effects, i);
            if (*effect == effect_str) {
                vector::remove(&mut target_info.status_effects, i);
                found = true;
            } else {
                i = i + 1;
            };
        };

        event::emit(StatusEffectEvent {
            combat_id: detailed_session.combat_id,
            target,
            effect_type: effect_str,
            applied: false,
            duration: 0,
        });
    }

    // === Helper Functions ===    /// Calculate total damage dealt in a combat session
    fun calculate_total_damage(combat_session: &CombatSession): u64 {
        let mut total = 0;
        let mut i = 0;
        while (i < vector::length(&combat_session.participants)) {
            let participant = *vector::borrow(&combat_session.participants, i);
            if (table::contains(&combat_session.damage_dealt, participant)) {
                let damage = *table::borrow(&combat_session.damage_dealt, participant);
                total = total + damage;
            };
            i = i + 1;
        };
        total
    }

    /// Calculate PvP rating based on wins and losses
    fun calculate_pvp_rating(wins: u64, losses: u64): u64 {
        let total_games = wins + losses;
        if (total_games == 0) {
            1000 // Starting rating
        } else {
            1000 + (wins * 20) - (losses * 15)
        }
    }

    // === Query Functions ===

    /// Get combat session info
    public fun get_combat_info(combat_session: &CombatSession): (u64, u8, u8, vector<address>) {
        (combat_session.combat_id, combat_session.combat_type, combat_session.status, combat_session.participants)
    }

    /// Get combat stats
    public fun get_combat_stats(stats: &CombatStats): (u64, u64, u64, u64, u64) {
        (stats.total_combats, stats.wins, stats.losses, stats.pvp_rating, stats.last_combat_time)
    }    // === Test Functions ===
    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(ctx);
    }

    #[test_only]
    public fun test_initiate_combat() {
        // Test logic here
    }

    #[test_only]
    public fun test_record_damage() {
        // Test logic here
    }
}
