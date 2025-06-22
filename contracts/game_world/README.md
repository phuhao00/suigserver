# MMO Game World System

A comprehensive Sui Move smart contract system for managing the game world, including resource nodes, regions, dynamic events, and treasure locations.

## Overview

This contract implements a sophisticated world management system that supports:
- **Resource Nodes**: Harvestable resources with regeneration and ownership
- **World Regions**: Territorial control with different properties and bonuses
- **Dynamic Events**: Timed world events with participation and rewards
- **Treasure System**: Discoverable treasures with rewards
- **Coordinate System**: 3D world positioning and distance calculations

## Features

### Resource Management

#### Resource Types
- **Gold (1)**: Basic currency resource
- **Iron (2)**: Common crafting material
- **Gems (3)**: Valuable decorative stones
- **Mithril (4)**: Rare magical metal
- **Crystals (5)**: Mystical energy sources

#### Resource Node Properties
```move
public struct ResourceNode has key, store {
    id: UID,
    resource_type: u8,           // Type of resource
    coordinates: Coordinates,    // World position
    last_harvested_epoch: u64,   // Cooldown tracking
    yield_per_harvest: u64,      // Amount per harvest
    max_yield: u64,              // Maximum capacity
    current_yield: u64,          // Current available
    harvest_cost: u64,           // SUI cost to harvest
    regeneration_rate: u64,      // Regeneration per epoch
    owner: Option<address>,      // Optional owner
    is_public: bool,             // Public or private
}
```

### World Regions

#### Region Types
- **Plains (1)**: Open grasslands
- **Forest (2)**: Dense woodlands
- **Mountains (3)**: Rocky highlands
- **Desert (4)**: Arid wastelands
- **Swamp (5)**: Marshy wetlands
- **Arctic (6)**: Frozen tundra

#### Region Properties
```move
public struct WorldRegion has key, store {
    id: UID,
    name: String,                    // Region name
    region_type: u8,                // Type constant
    coordinates: RegionBounds,       // Area boundaries
    climate: String,                 // Climate description
    danger_level: u8,               // Difficulty (1-10)
    resource_multiplier: u64,       // Resource bonus
    special_properties: vector<String>, // Special effects
    controlled_by: Option<address>, // Controller
    control_fee: u64,               // Control cost
}
```

### Dynamic Events

#### Event Types
- **Dragon Invasion (1)**: Epic dragon attacks
- **Resource Boom (2)**: Increased resource yields
- **Merchant Caravan (3)**: Trading opportunities
- **Mystical Fog (4)**: Magic effects
- **Treasure Hunt (5)**: Hidden treasure events

#### Event Structure
```move
public struct WorldEvent has key, store {
    id: UID,
    event_type: u8,              // Event type constant
    name: String,                // Event name
    description: String,         // Event description
    start_epoch: u64,           // Start time
    end_epoch: u64,             // End time
    affected_regions: vector<ID>, // Affected areas
    effect_multiplier: u64,     // Effect strength
    reward_pool: Balance<SUI>,  // Reward funds
    participants: vector<address>, // Participants
    is_active: bool,            // Active status
}
```

## Core Functions

### Resource Node Management

```move
// Create a new resource node (Admin only)
public entry fun create_resource_node(
    _admin_cap: &AdminCap,
    registry: &mut WorldRegistry,
    resource_type: u8,
    x: u64, y: u64, z: u64,        // Coordinates
    initial_yield: u64,
    max_yield: u64,
    harvest_cost: u64,
    regeneration_rate: u64,
    is_public: bool,
    ctx: &mut TxContext
)

// Harvest resources from a node
public entry fun harvest_resource(
    node: &mut ResourceNode,
    payment: Coin<SUI>,
    clock: &Clock,
    ctx: &mut TxContext
)

// Regenerate node resources (Admin only)
public entry fun regenerate_node(
    _admin_cap: &AdminCap,
    node: &mut ResourceNode,
    clock: &Clock,
    _ctx: &mut TxContext
)

// Purchase ownership of a node
public entry fun purchase_node_ownership(
    node: &mut ResourceNode,
    payment: Coin<SUI>,
    ctx: &mut TxContext
)
```

### Region Management

```move
// Create a world region (Admin only)
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
)

// Claim control of a region
public entry fun claim_region_control(
    region: &mut WorldRegion,
    payment: Coin<SUI>,
    ctx: &mut TxContext
)

// Release region control
public entry fun release_region_control(
    region: &mut WorldRegion,
    ctx: &mut TxContext
)
```

### Event Management

```move
// Trigger a world event (Admin only)
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
    ctx: &mut TxContext
)

// Participate in an event
public entry fun participate_in_event(
    world_event: &mut WorldEvent,
    clock: &Clock,
    ctx: &mut TxContext
)

// End an event and distribute rewards (Admin only)
public entry fun end_world_event(
    _admin_cap: &AdminCap,
    registry: &mut WorldRegistry,
    world_event: &mut WorldEvent,
    clock: &Clock,
    ctx: &mut TxContext
)
```

### Treasure System

```move
// Create treasure location (Admin only)
public entry fun create_treasure_location(
    _admin_cap: &AdminCap,
    x: u64, y: u64, z: u64,
    treasure_type: vector<u8>,
    value: u64,
    discovery_reward: u64,
    ctx: &mut TxContext
)

// Discover treasure
public entry fun discover_treasure(
    treasure: &mut TreasureLocation,
    ctx: &mut TxContext
)
```

## Coordinate System

### 3D Positioning
```move
public struct Coordinates has store, copy, drop {
    x: u64,  // Horizontal position
    y: u64,  // Horizontal position
    z: u64,  // Vertical position
}
```

### Region Boundaries
```move
public struct RegionBounds has store, copy, drop {
    min_x: u64, max_x: u64,
    min_y: u64, max_y: u64,
    min_z: u64, max_z: u64,
}
```

## Events

The system emits comprehensive events for all major actions:

- `WorldEventTriggered`: When world events begin
- `ResourceHarvested`: When resources are collected
- `RegionControlChanged`: When region ownership changes
- `TreasureDiscovered`: When treasures are found
- `NodeCreated`: When new resource nodes are created

## Economic Model

### Resource Harvesting
- **Cooldown Period**: 1 hour between harvests
- **Cost Structure**: SUI payment required for harvesting
- **Yield System**: Configurable yield per harvest
- **Regeneration**: Automatic resource regeneration over time

### Region Control
- **Control Fees**: SUI payment to claim regions
- **Benefits**: Resource multipliers for controlled regions
- **Transferable**: Control can be released or transferred

### Event Participation
- **Reward Pools**: SUI rewards distributed to participants
- **Equal Distribution**: Rewards split among all participants
- **Time-Limited**: Events have specific duration

## Building and Testing

### Prerequisites
- Sui CLI installed
- Sui framework dependencies

### Build
```bash
cd contracts/game_world
sui move build
```

### Test
```bash
sui move test
```

### Deploy
```bash
sui client publish --gas-budget 50000000
```

## Usage Examples

### Creating a Resource Node
```move
// Admin creates a gold mine
world::create_resource_node(
    &admin_cap,
    &mut registry,
    1,              // RESOURCE_GOLD
    100, 200, 50,   // coordinates
    1000,           // initial yield
    5000,           // max yield
    100000000,      // harvest cost (0.1 SUI)
    10,             // regeneration rate
    true,           // is public
    ctx
);
```

### Harvesting Resources
```move
// Player harvests from a node
let payment = coin::mint_for_testing<SUI>(100000000, ctx);
world::harvest_resource(&mut node, payment, &clock, ctx);
```

### Claiming Region Control
```move
// Player claims a region
let payment = coin::mint_for_testing<SUI>(1000000000, ctx); // 1 SUI
world::claim_region_control(&mut region, payment, ctx);
```

### Participating in Events
```move
// Player joins a world event
world::participate_in_event(&mut event, &clock, ctx);
```

## Error Codes

- `E_NOT_ADMIN (0)`: Admin capability required
- `E_RESOURCE_EXHAUSTED (1)`: Resource node is empty
- `E_COOLDOWN_ACTIVE (2)`: Harvest cooldown still active
- `E_INVALID_COORDINATES (3)`: Invalid coordinate values
- `E_REGION_NOT_FOUND (4)`: Region does not exist
- `E_INSUFFICIENT_PAYMENT (5)`: Payment amount too low
- `E_EVENT_NOT_ACTIVE (6)`: Event is not currently active
- `E_INVALID_RESOURCE_TYPE (7)`: Invalid resource type
- `E_NODE_ALREADY_EXISTS (8)`: Resource node already exists

## Integration

This world system integrates with:
- **Player System**: For resource collection and progression
- **Guild System**: For region control and group activities
- **Items System**: For resource-based crafting
- **Combat System**: For region battles and event participation

## Security Features

- **Admin Protection**: Critical functions require admin capability
- **Payment Verification**: All transactions verify payment amounts
- **Cooldown Enforcement**: Prevents resource farming abuse
- **Event Validation**: Ensures events are properly timed and active
- **Ownership Tracking**: Secure ownership and control mechanisms

## Future Enhancements

- Dynamic resource pricing based on scarcity
- Weather systems affecting resource generation
- Seasonal events with unique mechanics
- Player-owned territories with custom rules
- Resource trading markets
- Exploration rewards and discovery bonuses
- Environmental hazards and protection systems
