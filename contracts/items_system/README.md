# MMO Game Items System

A comprehensive Sui Move smart contract system for managing NFT-based items in an MMO game.

## Overview

This contract implements a flexible item system that supports:
- **Equipment**: Weapons, armor, and accessories with stat bonuses
- **Consumables**: Potions, scrolls, and other usable items with charges
- **Item upgrading**: Level up items to increase their power
- **Item rarity system**: Common, uncommon, rare, epic, and legendary items
- **Comprehensive testing**: Full test suite covering all functionality

## Features

### Item Types

1. **Weapons** (`ITEM_TYPE_WEAPON = 1`)
   - Provide attack bonuses
   - Can be upgraded to increase damage

2. **Armor** (`ITEM_TYPE_ARMOR = 2`)
   - Provide defense bonuses
   - Can be upgraded to increase protection

3. **Consumables** (`ITEM_TYPE_CONSUMABLE = 3`)
   - Have limited charges
   - Apply various effects (healing, mana restore, buffs)
   - Can be repaired to restore charges

4. **Accessories** (`ITEM_TYPE_ACCESSORY = 4`)
   - Provide both attack and defense bonuses
   - Balanced equipment type

### Rarity System

- **Common (1)**: Max level 10, basic stats
- **Uncommon (2)**: Max level 25, improved stats
- **Rare (3)**: Max level 50, good stats
- **Epic (4)**: Max level 75, high stats
- **Legendary (5)**: Max level 100, maximum stats

### Core Functions

#### Item Creation
```move
// Create a weapon
public entry fun create_weapon(
    item_type_id: u64,
    name: vector<u8>,
    description: vector<u8>,
    rarity: u8,
    level: u64,
    attack_bonus: u64,
    recipient: address,
    ctx: &mut TxContext
)

// Create armor
public entry fun create_armor(
    item_type_id: u64,
    name: vector<u8>,
    description: vector<u8>,
    rarity: u8,
    level: u64,
    defense_bonus: u64,
    recipient: address,
    ctx: &mut TxContext
)

// Create consumable
public entry fun create_consumable(
    item_type_id: u64,
    name: vector<u8>,
    description: vector<u8>,
    rarity: u8,
    charges: u8,
    effect_type: u8,
    effect_value: u64,
    recipient: address,
    ctx: &mut TxContext
)
```

#### Item Management
```move
// Use a consumable item
public entry fun use_consumable(item_nft: &mut ItemNFT)

// Upgrade an item's level
public entry fun upgrade_item(item_nft: &mut ItemNFT, levels: u64)

// Repair a consumable (restore charges)
public entry fun repair_consumable(item_nft: &mut ItemNFT, charges_to_add: u8)
```

#### Utility Functions
```move
// Calculate total combat power
public fun get_combat_power(item: &ItemNFT): u64

// Check if item can be upgraded
public fun can_upgrade(item: &ItemNFT): bool

// Get upgrade cost
public fun get_upgrade_cost(item: &ItemNFT): u64

// Check if items can be combined
public fun can_combine_items(item1: &ItemNFT, item2: &ItemNFT): bool
```

### Data Structure

```move
public struct ItemNFT has key, store {
    id: UID,
    item_type_id: u64,     // Reference to item definition
    name: String,          // Item name
    description: String,   // Item description
    item_type: u8,         // Type constant (weapon, armor, etc.)
    rarity: u8,           // Rarity level (1-5)
    level: u64,           // Current level
    attack_bonus: u64,    // Attack stat bonus
    defense_bonus: u64,   // Defense stat bonus
    charges: u8,          // Remaining charges (for consumables)
    effect_type: u8,      // Effect type (for consumables)
    effect_value: u64,    // Effect strength
}
```

## Building and Testing

### Prerequisites
- Sui CLI installed
- Sui framework dependencies

### Build
```bash
sui move build
```

### Test
```bash
sui move test
```

### Deploy
```bash
sui client publish --gas-budget 20000000
```

## Usage Examples

### Creating Items

```move
// Create a legendary sword
item::create_weapon(
    1,                          // item_type_id
    b"Excalibur",              // name
    b"A legendary blade",       // description
    5,                         // rarity (legendary)
    1,                         // level
    100,                       // attack_bonus
    @player_address,           // recipient
    ctx
);

// Create a health potion
item::create_consumable(
    2,                         // item_type_id
    b"Health Potion",          // name
    b"Restores 50 HP",         // description
    1,                         // rarity (common)
    5,                         // charges
    1,                         // effect_type (healing)
    50,                        // effect_value
    @player_address,           // recipient
    ctx
);
```

### Using Items

```move
// Use a consumable
item::use_consumable(&mut health_potion);

// Upgrade equipment
item::upgrade_item(&mut sword, 5); // Upgrade by 5 levels
```

## Error Codes

- `E_NOT_CONSUMABLE (0)`: Item is not a consumable type
- `E_NO_CHARGES (1)`: Consumable has no remaining charges
- `E_INVALID_ITEM_TYPE (2)`: Invalid item type specified
- `E_NOT_EQUIPMENT (3)`: Item is not equipment type

## Integration

This items system is designed to integrate with:
- Player system (for inventory management)
- Combat system (for stat calculations)
- Market system (for trading)
- Guild system (for shared equipment)

## Security Features

- All functions properly validate input parameters
- Ownership is enforced through Sui's object model
- Item stats are immutable except through designated upgrade functions
- Comprehensive error handling prevents invalid state transitions

## Future Enhancements

- Item durability system
- Set bonuses for equipment collections
- Crafting and enchantment systems
- Item fusion and combination mechanics
- Dynamic stat calculations based on player level
