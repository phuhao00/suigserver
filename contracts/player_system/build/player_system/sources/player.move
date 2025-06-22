// Player System Contract for Sui MMO Game
// Defines player character NFTs, stats, experience, levels, inventory management

#[allow(duplicate_alias, unused_use, unused_const, unused_field, unused_mut_parameter)]
module mmo_game::player {
    use std::string::{Self, String};
    use std::vector;
    use std::option::{Self, Option};
    use sui::object::{Self, UID, ID};
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};
    use sui::event;
    use sui::table::{Self, Table};
    use sui::clock::{Self, Clock};
    use sui::coin::{Self, Coin};
    use sui::sui::SUI;
    use sui::balance::{Self, Balance};

    // Error codes
    const E_NOT_AUTHORIZED: u64 = 1;
    const E_PLAYER_NOT_FOUND: u64 = 2;
    const E_INSUFFICIENT_EXPERIENCE: u64 = 3;
    const E_MAX_LEVEL_REACHED: u64 = 4;
    const E_INVALID_STAT_ALLOCATION: u64 = 5;
    const E_INSUFFICIENT_STAT_POINTS: u64 = 6;
    const E_INVALID_CLASS: u64 = 7;
    const E_NAME_TOO_LONG: u64 = 8;
    const E_NAME_TOO_SHORT: u64 = 9;
    const E_COOLDOWN_ACTIVE: u64 = 10;

    // Game constants
    const MAX_LEVEL: u64 = 100;
    const MAX_NAME_LENGTH: u64 = 20;
    const MIN_NAME_LENGTH: u64 = 3;
    const BASE_EXPERIENCE_PER_LEVEL: u64 = 1000;
    const STAT_POINTS_PER_LEVEL: u64 = 5;

    // Player classes
    const CLASS_WARRIOR: u8 = 1;
    const CLASS_MAGE: u8 = 2;
    const CLASS_ARCHER: u8 = 3;
    const CLASS_ROGUE: u8 = 4;

    // Admin capability for managing player system
    public struct AdminCap has key, store {
        id: UID,
    }

    /// Player Character NFT - represents a player's character
    public struct PlayerNFT has key, store {
        id: UID,
        name: String,
        class: u8,
        level: u64,
        experience: u64,
        stats: PlayerStats,
        available_stat_points: u64,
        equipment: Equipment,
        inventory_size: u64,
        creation_time: u64,
        last_login: u64,
        total_playtime: u64,
        achievements: vector<u64>,
        is_active: bool,
    }    /// Player statistics
    public struct PlayerStats has store, drop {
        strength: u64,
        agility: u64,
        intelligence: u64,
        vitality: u64,
        luck: u64,
        // Derived stats (calculated from base stats)
        max_health: u64,
        max_mana: u64,
        attack_power: u64,
        defense: u64,
        critical_chance: u64,
        critical_damage: u64,
    }

    /// Equipment slots
    public struct Equipment has store {
        weapon: Option<ID>,
        armor: Option<ID>,
        helmet: Option<ID>,
        boots: Option<ID>,
        gloves: Option<ID>,
        ring1: Option<ID>,
        ring2: Option<ID>,
        necklace: Option<ID>,
    }

    /// Player registry for managing all players
    public struct PlayerRegistry has key {
        id: UID,
        total_players: u64,
        active_players: u64,
        player_names: Table<String, address>, // name -> owner address
        player_count_by_class: Table<u8, u64>,
        admin: address,
    }

    /// Achievement data
    public struct Achievement has copy, drop, store {
        id: u64,
        name: String,
        description: String,
        reward_type: u8, // 1: experience, 2: items, 3: title
        reward_amount: u64,
    }

    /// Level up rewards
    public struct LevelUpReward has copy, drop, store {
        experience_bonus: u64,
        stat_points: u64,
        special_rewards: vector<String>,
    }

    // === Events ===

    public struct PlayerCreated has copy, drop {
        player_id: ID,
        owner: address,
        name: String,
        class: u8,
        creation_time: u64,
    }

    public struct PlayerLevelUp has copy, drop {
        player_id: ID,
        owner: address,
        new_level: u64,
        stat_points_gained: u64,
        timestamp: u64,
    }

    public struct ExperienceGained has copy, drop {
        player_id: ID,
        owner: address,
        amount: u64,
        source: String,
        timestamp: u64,
    }

    public struct StatsAllocated has copy, drop {
        player_id: ID,
        owner: address,
        strength_added: u64,
        agility_added: u64,
        intelligence_added: u64,
        vitality_added: u64,
        luck_added: u64,
        timestamp: u64,
    }

    public struct AchievementUnlocked has copy, drop {
        player_id: ID,
        owner: address,
        achievement_id: u64,
        timestamp: u64,
    }

    /// Initialize the player system
    fun init(ctx: &mut TxContext) {
        let admin_cap = AdminCap {
            id: object::new(ctx),
        };        let mut registry = PlayerRegistry {
            id: object::new(ctx),
            total_players: 0,
            active_players: 0,
            player_names: table::new(ctx),
            player_count_by_class: table::new(ctx),
            admin: tx_context::sender(ctx),
        };

        // Initialize class counters
        table::add(&mut registry.player_count_by_class, CLASS_WARRIOR, 0);
        table::add(&mut registry.player_count_by_class, CLASS_MAGE, 0);
        table::add(&mut registry.player_count_by_class, CLASS_ARCHER, 0);
        table::add(&mut registry.player_count_by_class, CLASS_ROGUE, 0);

        transfer::transfer(admin_cap, tx_context::sender(ctx));
        transfer::share_object(registry);
    }

    /// Create a new player character NFT
    public entry fun create_player(
        registry: &mut PlayerRegistry,
        name: vector<u8>,
        class: u8,
        ctx: &mut TxContext
    ) {
        let name_str = string::utf8(name);
        let name_length = string::length(&name_str);
        
        // Validate name
        assert!(name_length >= MIN_NAME_LENGTH, E_NAME_TOO_SHORT);
        assert!(name_length <= MAX_NAME_LENGTH, E_NAME_TOO_LONG);
        assert!(!table::contains(&registry.player_names, name_str), E_PLAYER_NOT_FOUND);
        
        // Validate class
        assert!(class >= CLASS_WARRIOR && class <= CLASS_ROGUE, E_INVALID_CLASS);

        // Create base stats based on class
        let base_stats = get_class_base_stats(class);
        let derived_stats = calculate_derived_stats(&base_stats);
        
        let stats = PlayerStats {
            strength: base_stats.strength,
            agility: base_stats.agility,
            intelligence: base_stats.intelligence,
            vitality: base_stats.vitality,
            luck: base_stats.luck,
            max_health: derived_stats.max_health,
            max_mana: derived_stats.max_mana,
            attack_power: derived_stats.attack_power,
            defense: derived_stats.defense,
            critical_chance: derived_stats.critical_chance,
            critical_damage: derived_stats.critical_damage,
        };

        let player_nft = PlayerNFT {
            id: object::new(ctx),
            name: name_str,
            class,
            level: 1,
            experience: 0,
            stats,
            available_stat_points: 0,
            equipment: Equipment {
                weapon: option::none(),
                armor: option::none(),
                helmet: option::none(),
                boots: option::none(),
                gloves: option::none(),
                ring1: option::none(),
                ring2: option::none(),
                necklace: option::none(),
            },
            inventory_size: 20, // Starting inventory size
            creation_time: tx_context::epoch(ctx),
            last_login: tx_context::epoch(ctx),
            total_playtime: 0,
            achievements: vector::empty(),
            is_active: true,
        };

        let player_id = object::id(&player_nft);
        let owner = tx_context::sender(ctx);

        // Update registry
        table::add(&mut registry.player_names, name_str, owner);
        registry.total_players = registry.total_players + 1;
        registry.active_players = registry.active_players + 1;
        
        let class_count = table::borrow_mut(&mut registry.player_count_by_class, class);
        *class_count = *class_count + 1;

        transfer::transfer(player_nft, owner);

        event::emit(PlayerCreated {
            player_id,
            owner,
            name: name_str,
            class,
            creation_time: tx_context::epoch(ctx),
        });
    }

    /// Grant experience to a player (admin only for now)
    public entry fun grant_experience(
        _admin_cap: &AdminCap,
        player_nft: &mut PlayerNFT,
        amount: u64,
        source: vector<u8>,
        ctx: &mut TxContext
    ) {
        let old_level = player_nft.level;
        player_nft.experience = player_nft.experience + amount;
        
        // Check for level up
        let new_level = calculate_level_from_experience(player_nft.experience);
        if (new_level > old_level && new_level <= MAX_LEVEL) {
            level_up_player(player_nft, new_level, ctx);
        };

        event::emit(ExperienceGained {
            player_id: object::id(player_nft),
            owner: tx_context::sender(ctx),
            amount,
            source: string::utf8(source),
            timestamp: tx_context::epoch(ctx),
        });
    }

    /// Allocate stat points
    public entry fun allocate_stats(
        player_nft: &mut PlayerNFT,
        strength: u64,
        agility: u64,
        intelligence: u64,
        vitality: u64,
        luck: u64,
        ctx: &mut TxContext
    ) {
        let total_points = strength + agility + intelligence + vitality + luck;
        assert!(total_points <= player_nft.available_stat_points, E_INSUFFICIENT_STAT_POINTS);
        assert!(total_points > 0, E_INVALID_STAT_ALLOCATION);

        // Apply stat increases
        player_nft.stats.strength = player_nft.stats.strength + strength;
        player_nft.stats.agility = player_nft.stats.agility + agility;
        player_nft.stats.intelligence = player_nft.stats.intelligence + intelligence;
        player_nft.stats.vitality = player_nft.stats.vitality + vitality;
        player_nft.stats.luck = player_nft.stats.luck + luck;

        // Reduce available points
        player_nft.available_stat_points = player_nft.available_stat_points - total_points;

        // Recalculate derived stats
        let derived_stats = calculate_derived_stats(&player_nft.stats);
        player_nft.stats.max_health = derived_stats.max_health;
        player_nft.stats.max_mana = derived_stats.max_mana;
        player_nft.stats.attack_power = derived_stats.attack_power;
        player_nft.stats.defense = derived_stats.defense;
        player_nft.stats.critical_chance = derived_stats.critical_chance;
        player_nft.stats.critical_damage = derived_stats.critical_damage;

        event::emit(StatsAllocated {
            player_id: object::id(player_nft),
            owner: tx_context::sender(ctx),
            strength_added: strength,
            agility_added: agility,
            intelligence_added: intelligence,
            vitality_added: vitality,
            luck_added: luck,
            timestamp: tx_context::epoch(ctx),
        });
    }

    /// Update player login time
    public entry fun update_login_time(
        player_nft: &mut PlayerNFT,
        ctx: &mut TxContext
    ) {
        player_nft.last_login = tx_context::epoch(ctx);
    }

    /// Unlock achievement for player
    public entry fun unlock_achievement(
        _admin_cap: &AdminCap,
        player_nft: &mut PlayerNFT,
        achievement_id: u64,
        ctx: &mut TxContext
    ) {
        if (!vector::contains(&player_nft.achievements, &achievement_id)) {
            vector::push_back(&mut player_nft.achievements, achievement_id);
            
            event::emit(AchievementUnlocked {
                player_id: object::id(player_nft),
                owner: tx_context::sender(ctx),
                achievement_id,
                timestamp: tx_context::epoch(ctx),
            });
        };
    }

    // === Helper Functions ===

    /// Level up a player
    fun level_up_player(player_nft: &mut PlayerNFT, new_level: u64, ctx: &mut TxContext) {
        let levels_gained = new_level - player_nft.level;
        let stat_points_gained = levels_gained * STAT_POINTS_PER_LEVEL;
        
        player_nft.level = new_level;
        player_nft.available_stat_points = player_nft.available_stat_points + stat_points_gained;

        event::emit(PlayerLevelUp {
            player_id: object::id(player_nft),
            owner: tx_context::sender(ctx),
            new_level,
            stat_points_gained,
            timestamp: tx_context::epoch(ctx),
        });
    }

    /// Calculate level from experience
    fun calculate_level_from_experience(experience: u64): u64 {
        if (experience == 0) return 1;        
        let mut level = 1;
        let mut required_exp = BASE_EXPERIENCE_PER_LEVEL;
        
        while (experience >= required_exp && level < MAX_LEVEL) {
            level = level + 1;
            required_exp = required_exp + (BASE_EXPERIENCE_PER_LEVEL * level);
        };
        
        level
    }

    /// Get base stats for character class
    fun get_class_base_stats(class: u8): PlayerStats {
        if (class == CLASS_WARRIOR) {
            PlayerStats {
                strength: 15,
                agility: 10,
                intelligence: 5,
                vitality: 15,
                luck: 5,
                max_health: 0,
                max_mana: 0,
                attack_power: 0,
                defense: 0,
                critical_chance: 0,
                critical_damage: 0,
            }
        } else if (class == CLASS_MAGE) {
            PlayerStats {
                strength: 5,
                agility: 8,
                intelligence: 20,
                vitality: 10,
                luck: 7,
                max_health: 0,
                max_mana: 0,
                attack_power: 0,
                defense: 0,
                critical_chance: 0,
                critical_damage: 0,
            }
        } else if (class == CLASS_ARCHER) {
            PlayerStats {
                strength: 8,
                agility: 18,
                intelligence: 7,
                vitality: 12,
                luck: 5,
                max_health: 0,
                max_mana: 0,
                attack_power: 0,
                defense: 0,
                critical_chance: 0,
                critical_damage: 0,
            }
        } else { // CLASS_ROGUE
            PlayerStats {
                strength: 10,
                agility: 15,
                intelligence: 8,
                vitality: 10,
                luck: 7,
                max_health: 0,
                max_mana: 0,
                attack_power: 0,
                defense: 0,
                critical_chance: 0,
                critical_damage: 0,
            }
        }
    }    /// Calculate derived stats from base stats
    fun calculate_derived_stats(stats: &PlayerStats): PlayerStats {
        PlayerStats {
            strength: stats.strength,
            agility: stats.agility,
            intelligence: stats.intelligence,
            vitality: stats.vitality,
            luck: stats.luck,
            max_health: 100 + (stats.vitality * 10),
            max_mana: 50 + (stats.intelligence * 10), // Changed from 5 to 10 multiplier
            attack_power: stats.strength * 2 + stats.agility,
            defense: stats.vitality + (stats.strength / 2),
            critical_chance: stats.luck * 2,
            critical_damage: 150 + (stats.luck * 3),
        }
    }

    // === Query Functions ===

    /// Get player info
    public fun get_player_info(player: &PlayerNFT): (String, u8, u64, u64, u64) {
        (player.name, player.class, player.level, player.experience, player.available_stat_points)
    }    /// Get player stats
    public fun get_player_stats(player: &PlayerNFT): &PlayerStats {
        &player.stats
    }

    /// Get individual stat values for testing
    public fun get_strength(stats: &PlayerStats): u64 { stats.strength }
    public fun get_agility(stats: &PlayerStats): u64 { stats.agility }
    public fun get_intelligence(stats: &PlayerStats): u64 { stats.intelligence }
    public fun get_vitality(stats: &PlayerStats): u64 { stats.vitality }
    public fun get_luck(stats: &PlayerStats): u64 { stats.luck }
    public fun get_max_health(stats: &PlayerStats): u64 { stats.max_health }
    public fun get_max_mana(stats: &PlayerStats): u64 { stats.max_mana }

    /// Get registry info
    public fun get_registry_info(registry: &PlayerRegistry): (u64, u64) {
        (registry.total_players, registry.active_players)
    }    /// Check if name is available
    public fun is_name_available(registry: &PlayerRegistry, name: String): bool {
        !table::contains(&registry.player_names, name)
    }

    /// Get experience required for next level
    public fun get_experience_for_level(level: u64): u64 {
        if (level <= 1) return 0;
        
        let mut total_exp = 0;
        let mut i = 2;
        while (i <= level) {
            total_exp = total_exp + (BASE_EXPERIENCE_PER_LEVEL * i);
            i = i + 1;
        };
        total_exp
    }

    // === Test Functions ===
    #[test_only]
    public fun test_init(ctx: &mut TxContext) {
        init(ctx);
    }

    #[test_only]
    public fun test_create_admin_cap(ctx: &mut TxContext): AdminCap {
        AdminCap { id: object::new(ctx) }
    }

    #[test_only]
    public fun test_get_class_base_stats(class: u8): PlayerStats {
        get_class_base_stats(class)
    }
}
