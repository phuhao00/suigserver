# Combat System Smart Contract

This smart contract implements a comprehensive combat system for a Sui-based MMO game. It handles combat logic, results recording, and PvP/PvE interactions with complex real-time calculations handled off-chain by a Go server.

## Features

### Core Combat Functionality
- **Combat Session Management**: Create, start, and finalize combat sessions
- **Damage & Healing Tracking**: Record damage dealt and healing performed during combat
- **Turn-based Combat**: Support for detailed turn-based combat with action tracking
- **Status Effects**: Apply and remove status effects on participants
- **PvP Rating System**: Automatic calculation of PvP ratings based on wins/losses

### Combat Types
- **PvP (Player vs Player)**: Direct combat between players
- **PvE (Player vs Environment)**: Combat against NPCs/environment
- **Guild Wars**: Large-scale guild vs guild combat

### Data Structures

#### Core Structs
- `AdminCap`: Administrative capability for system management
- `CombatSession`: Basic combat session with participant tracking
- `DetailedCombatSession`: Advanced combat session with turn-based mechanics
- `CombatStats`: Player combat statistics tracking
- `CombatRegistry`: Global combat state management

#### Event Structs
- `CombatInitiated`: Combat session start event
- `CombatOutcome`: Combat completion event with results
- `DamageEvent`: Damage dealt during combat
- `HealingEvent`: Healing performed during combat
- `CombatActionEvent`: Individual combat actions
- `StatusEffectEvent`: Status effect changes
- `TurnChangeEvent`: Turn progression in detailed combat

## Smart Contract Functions

### Administrative Functions

#### `init(ctx: &mut TxContext)`
Initializes the combat system, creating admin capabilities and combat registry.

#### `initiate_combat(...)`
Creates a new combat session with specified parameters.
- Parameters: combat_id, combat_type, participants, reward_amount, metadata
- Creates a new `CombatSession` object

#### `create_detailed_combat(...)`
Creates an advanced combat session with detailed participant information and turn-based mechanics.
- Supports health, mana, and equipment power tracking
- Enables turn-based combat flow

### Combat Flow Functions

#### `start_combat(...)`
Transitions a combat session from initiated to in-progress status.

#### `record_damage(...)`
Records damage dealt by a participant to a target.
- Updates damage tracking tables
- Emits damage events for off-chain processing

#### `record_healing(...)`
Records healing performed by a participant.
- Updates healing tracking tables
- Emits healing events

#### `execute_combat_action(...)`
Executes a combat action in detailed combat sessions.
- Supports various action types (attack, heal, spell, item use)
- Applies effects based on action success
- Tracks critical hits and additional effects

#### `advance_turn(...)`
Progresses to the next turn in turn-based combat.
- Manages turn order
- Handles turn limits and combat completion
- Emits turn change events

#### `finalize_combat(...)`
Completes a combat session and distributes rewards.
- Calculates final damage totals
- Distributes SUI rewards to participants
- Records combat outcome

#### `cancel_combat(...)`
Cancels an ongoing combat session and refunds participants.

### Status Effect Management

#### `apply_status_effect(...)`
Applies a status effect to a combat participant.

#### `remove_status_effect(...)`
Removes a status effect from a participant.

### Utility Functions

#### `get_combat_info(combat_session: &CombatSession)`
Returns basic information about a combat session.

#### `get_combat_stats(stats: &CombatStats)`
Returns player combat statistics.

#### `calculate_total_damage(combat_session: &CombatSession)`
Calculates total damage dealt in a combat session.

#### `calculate_pvp_rating(wins: u64, losses: u64)`
Calculates PvP rating based on win/loss record.

## Error Codes

- `E_NOT_AUTHORIZED (1)`: Unauthorized access attempt
- `E_INVALID_COMBAT_ID (2)`: Invalid combat session ID
- `E_COMBAT_ALREADY_EXISTS (3)`: Combat session already exists
- `E_INSUFFICIENT_REWARDS (4)`: Insufficient reward balance
- `E_INVALID_PARTICIPANT (5)`: Invalid combat participant
- `E_INVALID_COMBAT_TYPE (6)`: Invalid combat type specified

## Combat Status Codes

- `STATUS_INITIATED (0)`: Combat created but not started
- `STATUS_IN_PROGRESS (1)`: Combat actively running
- `STATUS_FINISHED (2)`: Combat completed
- `STATUS_CANCELLED (3)`: Combat cancelled

## Combat Type Constants

- `COMBAT_TYPE_PVP (1)`: Player vs Player combat
- `COMBAT_TYPE_PVE (2)`: Player vs Environment combat
- `COMBAT_TYPE_GUILD_WAR (3)`: Guild warfare combat

## Usage Examples

### Basic Combat Flow

1. **Initialize Combat System**
   ```move
   combat::init(ctx);
   ```

2. **Create Combat Session**
   ```move
   combat::initiate_combat(
       &admin_cap,
       combat_id,
       COMBAT_TYPE_PVP,
       participants,
       reward_coin,
       metadata,
       ctx
   );
   ```

3. **Start Combat**
   ```move
   combat::start_combat(&admin_cap, &mut combat_session, ctx);
   ```

4. **Record Combat Actions**
   ```move
   combat::record_damage(&admin_cap, &mut combat_session, attacker, target, damage, damage_type, ctx);
   combat::record_healing(&admin_cap, &mut combat_session, healer, target, healing, ctx);
   ```

5. **Finalize Combat**
   ```move
   combat::finalize_combat(&admin_cap, &mut combat_session, winner, loser, rewards, ctx);
   ```

### Advanced Turn-Based Combat

1. **Create Detailed Combat Session**
   ```move
   combat::create_detailed_combat(
       &admin_cap,
       &mut registry,
       combat_id,
       COMBAT_TYPE_PVP,
       participants,
       health_values,
       mana_values,
       power_values,
       reward_coin,
       max_turns,
       metadata,
       ctx
   );
   ```

2. **Execute Actions**
   ```move
   combat::execute_combat_action(
       &admin_cap,
       &mut detailed_session,
       actor,
       action_type,
       target,
       value,
       success,
       critical_hit,
       additional_effects,
       ctx
   );
   ```

3. **Advance Turns**
   ```move
   combat::advance_turn(&admin_cap, &mut detailed_session, ctx);
   ```

## Integration with Off-Chain Systems

The combat system is designed to work with off-chain game servers that handle:

- Real-time combat calculations
- Complex game logic
- User interface updates
- Network synchronization

The smart contract serves as the authoritative source for:

- Combat results and outcomes
- Reward distribution
- Player statistics
- Combat history

## Security Considerations

- All administrative functions require `AdminCap` to prevent unauthorized access
- Participant validation ensures only valid players can participate in combat
- Balance checks prevent reward distribution exceeding available funds
- Status validation prevents invalid state transitions

## Testing

The contract includes test functions for verification:

- `test_init()`: Tests system initialization
- `test_initiate_combat()`: Tests combat creation
- `test_record_damage()`: Tests damage recording

Comprehensive test suite is available in `tests/combat_tests.move`.

## Future Enhancements

- Support for team-based combat
- Complex spell and ability systems
- Equipment and item interactions
- Tournament and ladder systems
- Cross-chain combat capabilities
