# MMO Game Guild System

A comprehensive Sui Move smart contract system for managing guild organizations in an MMO game.

## Overview

This contract implements a sophisticated guild management system that supports:
- **Guild Creation & Management**: Complete guild lifecycle management
- **Role-based Permissions**: Leader, officer, and member roles with different capabilities
- **Treasury System**: Shared guild funds with contribution tracking
- **Member Management**: Add, remove, promote, and demote members
- **Guild Progression**: Experience and leveling system for guilds
- **Event System**: Comprehensive event emissions for all major actions

## Features

### Guild Structure

```move
public struct Guild has key, store {
    id: UID,
    name: String,           // Unique guild name
    leader: address,        // Guild leader address
    members: vector<address>, // All guild members
    officers: vector<address>, // Guild officers
    treasury: Balance<SUI>, // Guild treasury
    motd: String,          // Message of the day
    level: u64,            // Guild level
    experience: u64,       // Guild experience points
    created_at: u64,       // Creation timestamp
    max_members: u64,      // Maximum member capacity
    description: String,   // Guild description
}
```

### Role System

- **Leader (0)**: Full control over guild, can transfer leadership
- **Officer (1)**: Can manage members and update MOTD
- **Member (2)**: Basic guild membership

### Core Functions

#### Guild Creation
```move
public entry fun create_guild(
    registry: &mut GuildRegistry,
    name: vector<u8>,
    description: vector<u8>,
    ctx: &mut TxContext
)
```

#### Member Management
```move
// Add member (Leader/Officer only)
public entry fun add_member(
    guild: &mut Guild,
    player_address: address,
    ctx: &mut TxContext
)

// Remove member (Leader/Officer only)
public entry fun remove_member(
    guild: &mut Guild,
    player_address: address,
    ctx: &mut TxContext
)

// Leave guild voluntarily
public entry fun leave_guild(guild: &mut Guild, ctx: &mut TxContext)
```

#### Role Management
```move
// Promote member to officer (Leader only)
public entry fun promote_to_officer(
    guild: &mut Guild,
    player_address: address,
    ctx: &mut TxContext
)

// Demote officer to member (Leader only)
public entry fun demote_officer(
    guild: &mut Guild,
    player_address: address,
    ctx: &mut TxContext
)

// Transfer guild leadership (Leader only)
public entry fun transfer_leadership(
    guild: &mut Guild,
    new_leader: address,
    ctx: &mut TxContext
)
```

#### Treasury Management
```move
// Deposit funds to guild treasury
public entry fun deposit_to_treasury(
    guild: &mut Guild,
    payment: Coin<SUI>,
    ctx: &mut TxContext
)

// Withdraw funds from treasury (Leader only)
public entry fun withdraw_from_treasury(
    guild: &mut Guild,
    amount: u64,
    ctx: &mut TxContext
)
```

#### Guild Management
```move
// Update message of the day (Leader/Officer)
public entry fun update_motd(
    guild: &mut Guild,
    new_motd: vector<u8>,
    ctx: &mut TxContext
)

// Update guild description (Leader only)
public entry fun update_description(
    guild: &mut Guild,
    new_description: vector<u8>,
    ctx: &mut TxContext
)

// Upgrade guild capacity (Leader only)
public entry fun upgrade_guild(
    guild: &mut Guild,
    payment: Coin<SUI>,
    ctx: &mut TxContext
)

// Add experience points
public entry fun add_guild_experience(
    guild: &mut Guild,
    experience_points: u64,
    ctx: &mut TxContext
)
```

## Guild Registry

The system uses a global registry to track all guilds and ensure unique names:

```move
public struct GuildRegistry has key {
    id: UID,
    guild_names: vector<String>,
    total_guilds: u64,
}
```

## Member Information Tracking

Each member has associated metadata stored as dynamic fields:

```move
public struct MemberInfo has store {
    role: u8,              // Member role
    joined_at: u64,        // Join timestamp
    contribution: u64,     // Total SUI contributed
}
```

## Events

The system emits comprehensive events for monitoring:

- `GuildCreated`: When a new guild is created
- `MemberJoined`: When a member joins
- `MemberLeft`: When a member leaves
- `RoleChanged`: When member roles are updated
- `TreasuryDeposit`: When funds are deposited
- `TreasuryWithdraw`: When funds are withdrawn

## Permission System

### Leaders Can:
- Add/remove members
- Promote/demote officers
- Transfer leadership
- Update guild description
- Update MOTD
- Withdraw from treasury
- Upgrade guild

### Officers Can:
- Add/remove members
- Update MOTD

### Members Can:
- Deposit to treasury
- Leave guild
- View guild information

## Guild Progression

- **Starting Level**: 1
- **Starting Capacity**: 25 members
- **Experience Required**: 1000 XP per level
- **Upgrade Cost**: 1 SUI per current level
- **Max Capacity**: 100 members

## Building and Testing

### Prerequisites
- Sui CLI installed
- Sui framework dependencies

### Build
```bash
cd contracts/guild_system
sui move build
```

### Test
```bash
sui move test
```

### Deploy
```bash
sui client publish --gas-budget 30000000
```

## Usage Examples

### Creating a Guild
```move
// First, create the registry (done once)
let registry = guild::create_registry_for_testing(ctx);

// Create a guild
guild::create_guild(
    &mut registry,
    b"Dragon Slayers",
    b"Elite guild for dragon hunting",
    ctx
);
```

### Managing Members
```move
// Add a member
guild::add_member(&mut guild, @new_member, ctx);

// Promote to officer
guild::promote_to_officer(&mut guild, @member, ctx);

// Member leaves
guild::leave_guild(&mut guild, ctx);
```

### Treasury Operations
```move
// Deposit funds
let payment = coin::mint_for_testing<SUI>(1000000000, ctx); // 1 SUI
guild::deposit_to_treasury(&mut guild, payment, ctx);

// Withdraw (leader only)
guild::withdraw_from_treasury(&mut guild, 500000000, ctx); // 0.5 SUI
```

## Error Codes

- `E_GUILD_ALREADY_EXISTS (0)`: Guild name already taken
- `E_NOT_GUILD_LEADER (1)`: Action requires leader privileges
- `E_PLAYER_ALREADY_IN_GUILD (2)`: Player is already a member
- `E_PLAYER_NOT_IN_GUILD (3)`: Player is not a guild member
- `E_INSUFFICIENT_PERMISSIONS (4)`: Insufficient role permissions
- `E_GUILD_FULL (5)`: Guild has reached member capacity
- `E_INVALID_ROLE (6)`: Invalid role operation
- `E_CANNOT_REMOVE_LEADER (7)`: Cannot remove/demote leader
- `E_INSUFFICIENT_FUNDS (8)`: Not enough funds for operation
- `E_INVALID_MEMBER_COUNT (9)`: Invalid member count for operation

## Integration

This guild system is designed to integrate with:
- Player system (for member verification)
- Combat system (for guild wars and raids)
- Items system (for shared guild equipment)
- Economy system (for guild taxes and rewards)

## Security Features

- **Role-based Access Control**: Strict permission checking for all operations
- **Unique Guild Names**: Registry prevents duplicate guild names
- **Member Verification**: All operations verify member status
- **Treasury Protection**: Only leaders can withdraw funds
- **Event Logging**: All major actions emit events for transparency

## Future Enhancements

- Guild wars and alliance systems
- Shared guild storage for items
- Guild skills and bonuses
- Automated tax collection
- Guild quests and achievements
- Member activity tracking
- Guild rankings and leaderboards
