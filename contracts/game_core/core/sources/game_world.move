module mmo_core::game_world {
    use std::string::{Self, String};
    use std::vector;
    use std::option::{Self, Option};
    use sui::object::{Self, UID, ID};
    use sui::transfer;
    use sui::tx_context::{Self, TxContext};
    use sui::event;
    use sui::table::{Self, Table};
    use sui::clock::{Self, Clock};

    // 错误码
    const E_NOT_AUTHORIZED: u64 = 1;
    const E_REGION_NOT_FOUND: u64 = 2;
    const E_PLAYER_NOT_FOUND: u64 = 3;
    const E_INVALID_POSITION: u64 = 4;

    /// 游戏世界管理器
    public struct GameWorld has key {
        id: UID,
        name: String,
        regions: Table<String, RegionConfig>,
        server_info: ServerInfo,
        admin: address,
        created_at: u64,
    }

    /// 服务器信息
    public struct ServerInfo has store {
        version: String,
        max_players: u64,
        current_players: u64,
        status: u8, // 0: 维护, 1: 正常, 2: 满载
        last_update: u64,
    }

    /// 区域配置
    public struct RegionConfig has store {
        name: String,
        map_id: String,
        max_players: u64,
        current_players: u64,
        region_type: u8, // 1: 安全区, 2: PvP区, 3: 副本
        spawn_points: vector<Position>,
        level_range: LevelRange,
        is_active: bool,
    }

    /// 等级范围
    public struct LevelRange has store {
        min_level: u64,
        max_level: u64,
    }

    /// 位置信息
    public struct Position has copy, drop, store {
        x: u64,
        y: u64,
        z: u64,
        map_id: String,
    }

    /// 玩家会话 - 在线状态管理
    public struct PlayerSession has key {
        id: UID,
        player_address: address,
        character_id: ID,
        current_region: String,
        current_position: Position,
        login_time: u64,
        last_heartbeat: u64,
        server_id: String,
        session_data: SessionData,
    }

    /// 会话数据
    public struct SessionData has store {
        player_name: String,
        level: u64,
        health: u64,
        mana: u64,
        combat_state: u8, // 0: 正常, 1: 战斗中, 2: 死亡
        buffs: vector<u64>,
        temporary_data: Table<String, vector<u8>>,
    }

    /// 初始化游戏世界
    fun init(ctx: &mut TxContext) {
        let game_world = GameWorld {
            id: object::new(ctx),
            name: string::utf8(b"Sui MMO World"),
            regions: table::new(ctx),
            server_info: ServerInfo {
                version: string::utf8(b"1.0.0"),
                max_players: 10000,
                current_players: 0,
                status: 1,
                last_update: tx_context::epoch(ctx),
            },
            admin: tx_context::sender(ctx),
            created_at: tx_context::epoch(ctx),
        };

        transfer::share_object(game_world);

        event::emit(GameWorldCreated {
            world_id: object::uid_to_address(&game_world.id),
            admin: tx_context::sender(ctx),
        });
    }

    /// 玩家登录 - 创建会话
    public entry fun player_login(
        world: &mut GameWorld,
        character_id: ID,
        region_name: String,
        server_id: vector<u8>,
        ctx: &mut TxContext
    ) {
        assert!(table::contains(&world.regions, region_name), E_REGION_NOT_FOUND);

        let region = table::borrow_mut(&mut world.regions, region_name);
        assert!(region.current_players < region.max_players, E_REGION_NOT_FOUND);

        // 创建玩家会话
        let session = PlayerSession {
            id: object::new(ctx),
            player_address: tx_context::sender(ctx),
            character_id,
            current_region: region_name,
            current_position: *vector::borrow(&region.spawn_points, 0),
            login_time: tx_context::epoch(ctx),
            last_heartbeat: tx_context::epoch(ctx),
            server_id: string::utf8(server_id),
            session_data: SessionData {
                player_name: string::utf8(b""),
                level: 1,
                health: 100,
                mana: 100,
                combat_state: 0,
                buffs: vector::empty(),
                temporary_data: table::new(ctx),
            },
        };

        region.current_players = region.current_players + 1;
        world.server_info.current_players = world.server_info.current_players + 1;

        let session_id = object::uid_to_address(&session.id);
        transfer::transfer(session, tx_context::sender(ctx));

        event::emit(PlayerLoggedIn {
            session_id,
            player_address: tx_context::sender(ctx),
            character_id,
            region_name,
            login_time: tx_context::epoch(ctx),
        });
    }

    /// 玩家移动
    public entry fun update_player_position(
        session: &mut PlayerSession,
        new_position: Position,
        ctx: &mut TxContext
    ) {
        assert!(session.player_address == tx_context::sender(ctx), E_NOT_AUTHORIZED);

        let old_position = session.current_position;
        session.current_position = new_position;
        session.last_heartbeat = tx_context::epoch(ctx);

        event::emit(PlayerMoved {
            session_id: object::uid_to_address(&session.id),
            player_address: session.player_address,
            from_position: old_position,
            to_position: new_position,
            timestamp: tx_context::epoch(ctx),
        });
    }

    /// 心跳包 - 保持连接活跃
    public entry fun player_heartbeat(
        session: &mut PlayerSession,
        health: u64,
        mana: u64,
        ctx: &mut TxContext
    ) {
        assert!(session.player_address == tx_context::sender(ctx), E_NOT_AUTHORIZED);

        session.last_heartbeat = tx_context::epoch(ctx);
        session.session_data.health = health;
        session.session_data.mana = mana;

        event::emit(PlayerHeartbeat {
            session_id: object::uid_to_address(&session.id),
            timestamp: tx_context::epoch(ctx),
        });
    }

    /// 玩家登出
    public entry fun player_logout(
        world: &mut GameWorld,
        session: PlayerSession,
        ctx: &mut TxContext
    ) {
        let PlayerSession {
            id,
            player_address,
            character_id: _,
            current_region,
            current_position: _,
            login_time: _,
            last_heartbeat: _,
            server_id: _,
            session_data: _,
        } = session;

        assert!(player_address == tx_context::sender(ctx), E_NOT_AUTHORIZED);

        // 更新区域和服务器玩家数量
        let region = table::borrow_mut(&mut world.regions, current_region);
        region.current_players = region.current_players - 1;
        world.server_info.current_players = world.server_info.current_players - 1;

        let session_id = object::uid_to_address(&id);
        object::delete(id);

        event::emit(PlayerLoggedOut {
            session_id,
            player_address,
            logout_time: tx_context::epoch(ctx),
        });
    }

    /// 管理员创建区域
    public entry fun create_region(
        world: &mut GameWorld,
        name: vector<u8>,
        map_id: vector<u8>,
        max_players: u64,
        region_type: u8,
        min_level: u64,
        max_level: u64,
        spawn_x: u64,
        spawn_y: u64,
        spawn_z: u64,
        ctx: &mut TxContext
    ) {
        assert!(world.admin == tx_context::sender(ctx), E_NOT_AUTHORIZED);

        let region_name = string::utf8(name);
        let spawn_points = vector::empty();
        vector::push_back(&mut spawn_points, Position {
            x: spawn_x,
            y: spawn_y,
            z: spawn_z,
            map_id: string::utf8(map_id),
        });

        let region = RegionConfig {
            name: region_name,
            map_id: string::utf8(map_id),
            max_players,
            current_players: 0,
            region_type,
            spawn_points,
            level_range: LevelRange { min_level, max_level },
            is_active: true,
        };

        table::add(&mut world.regions, region_name, region);

        event::emit(RegionCreated {
            world_id: object::uid_to_address(&world.id),
            region_name,
            map_id: string::utf8(map_id),
            max_players,
            region_type,
        });
    }

    // 查询函数
    public fun get_server_info(world: &GameWorld): &ServerInfo {
        &world.server_info
    }

    public fun get_region_info(world: &GameWorld, region_name: String): &RegionConfig {
        table::borrow(&world.regions, region_name)
    }

    public fun get_session_info(session: &PlayerSession): (address, String, Position, u64) {
        (session.player_address, session.current_region, session.current_position, session.last_heartbeat)
    }

    // 事件定义
    public struct GameWorldCreated has copy, drop {
        world_id: address,
        admin: address,
    }

    public struct PlayerLoggedIn has copy, drop {
        session_id: address,
        player_address: address,
        character_id: ID,
        region_name: String,
        login_time: u64,
    }

    public struct PlayerMoved has copy, drop {
        session_id: address,
        player_address: address,
        from_position: Position,
        to_position: Position,
        timestamp: u64,
    }

    public struct PlayerHeartbeat has copy, drop {
        session_id: address,
        timestamp: u64,
    }

    public struct PlayerLoggedOut has copy, drop {
        session_id: address,
        player_address: address,
        logout_time: u64,
    }

    public struct RegionCreated has copy, drop {
        world_id: address,
        region_name: String,
        map_id: String,
        max_players: u64,
        region_type: u8,
    }
}