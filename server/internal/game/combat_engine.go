package game

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/phuhao00/suigserver/server/internal/sui" // For interacting with Sui blockchain
)

// CombatantStats represents the basic stats for a combatant.
// This would likely be part of a larger Character or NPC model.
type CombatantStats struct {
	ID          string
	Health      int
	MaxHealth   int
	AttackPower int
	Defense     int
	Speed       int // Determines attack order or frequency
	// Add other relevant stats: critical chance, evasion, resistances, etc.
}

// CombatResult holds the outcome of a combat interaction.
type CombatResult struct {
	AttackerID         string
	DefenderID         string
	DamageDealt        int
	DefenderHealth     int // Defender's health after the attack
	AttackerHealth     int // Attacker's health (if defender counter-attacks or reflects)
	IsCriticalHit      bool
	IsEvaded           bool
	CombatLog          []string // Log of events during this combat turn/round
	IsDefenderDefeated bool
}

// CombatEngine handles all combat calculations and logic.
type CombatEngine struct {
	suiCombatService *sui.CombatResultsSuiService // For recording combat results on-chain
	// dbCache *DBCacheLayer    // For fetching/updating combatant stats if not passed directly
	baseHitChance       float64
	baseCritChance      float64
	baseEvadeChance     float64
	critDamageBonus     float64 // e.g., 1.5 for 50% extra damage
	minDamagePercentage float64 // Minimum percentage of attack power to deal as damage
	// Configurable parameters
	skillDefinitions  map[string]interface{} // Placeholder for skill data
	statusEffectRules map[string]interface{} // Placeholder for status effect rules
	elementalChart    map[string]interface{} // Placeholder for elemental advantages
}

// NewCombatEngine creates a new CombatEngine.
func NewCombatEngine(suiCombatService *sui.CombatResultsSuiService /*, dbCache *DBCacheLayer*/) *CombatEngine {
	log.Println("Initializing Combat Engine...")
	rand.Seed(time.Now().UnixNano()) // Seed random number generator
	return &CombatEngine{
		suiCombatService: suiCombatService,
		// dbCache: dbCache,
		baseHitChance:       0.90, // 90% base chance to hit
		baseCritChance:      0.10, // 10% base chance for a critical hit
		baseEvadeChance:     0.05, // 5% base chance to evade
		critDamageBonus:     1.5,
		minDamagePercentage: 0.1, // Ensure at least 10% of attack power as damage if hit
	}
}

// Start begins the combat engine operations.
// This is where you might load configurations for skills, effects, etc.
func (ce *CombatEngine) Start(config *CombatEngineConfig) { // Assuming a config struct
	log.Println("Combat Engine starting...")
	if config != nil {
		// Example: Load skill definitions from config
		ce.skillDefinitions = config.SkillDefinitions
		log.Printf("Loaded %d skill definitions.", len(ce.skillDefinitions))

		// Example: Load status effect rules
		ce.statusEffectRules = config.StatusEffectRules
		log.Printf("Loaded %d status effect rules.", len(ce.statusEffectRules))

		// Example: Load elemental chart
		ce.elementalChart = config.ElementalChart
		log.Printf("Loaded elemental chart with %d entries.", len(ce.elementalChart))

		// Override base chances if provided in config
		if config.BaseHitChance > 0 {
			ce.baseHitChance = config.BaseHitChance
		}
		if config.BaseCritChance > 0 {
			ce.baseCritChance = config.BaseCritChance
		}
		if config.BaseEvadeChance > 0 {
			ce.baseEvadeChance = config.BaseEvadeChance
		}
		if config.CritDamageBonus > 0 {
			ce.critDamageBonus = config.CritDamageBonus
		}
		if config.MinDamagePercentage > 0 {
			ce.minDamagePercentage = config.MinDamagePercentage
		}
	} else {
		log.Println("Combat Engine started with default parameters (no config provided).")
	}

	log.Printf("Combat Parameters: HitChance=%.2f, CritChance=%.2f, EvadeChance=%.2f, CritBonus=%.2fx, MinDamageFactor=%.2f",
		ce.baseHitChance, ce.baseCritChance, ce.baseEvadeChance, ce.critDamageBonus, ce.minDamagePercentage)
	log.Println("Combat Engine started successfully.")
}

// CombatEngineConfig holds configuration for the combat engine.
// This would typically be loaded from a JSON/YAML file.
type CombatEngineConfig struct {
	SkillDefinitions    map[string]interface{} `json:"skillDefinitions"`
	StatusEffectRules   map[string]interface{} `json:"statusEffectRules"`
	ElementalChart      map[string]interface{} `json:"elementalChart"`
	BaseHitChance       float64                `json:"baseHitChance,omitempty"`
	BaseCritChance      float64                `json:"baseCritChance,omitempty"`
	BaseEvadeChance     float64                `json:"baseEvadeChance,omitempty"`
	CritDamageBonus     float64                `json:"critDamageBonus,omitempty"`
	MinDamagePercentage float64                `json:"minDamagePercentage,omitempty"`
}

// Stop gracefully shuts down the combat engine.
func (ce *CombatEngine) Stop() {
	log.Println("Combat Engine stopped.")
}

// SimulateCombatTurn simulates a single turn of combat between an attacker and a defender.
// In a real game, you'd fetch full stats for attacker and defender.
// For now, we'll pass simplified stats.
func (ce *CombatEngine) SimulateCombatTurn(attacker, defender CombatantStats) *CombatResult {
	log.Printf("Simulating combat turn: Attacker %s vs Defender %s", attacker.ID, defender.ID)
	result := &CombatResult{
		AttackerID:     attacker.ID,
		DefenderID:     defender.ID,
		AttackerHealth: attacker.Health, // Assuming no counter-attack for simplicity here
		DefenderHealth: defender.Health,
		CombatLog:      make([]string, 0),
	}

	result.CombatLog = append(result.CombatLog, time.Now().Format(time.RFC3339)+": "+attacker.ID+" prepares to attack "+defender.ID+".")

	// 1. Check for evasion
	if rand.Float64() < ce.baseEvadeChance { // Simplified evasion check
		result.IsEvaded = true
		result.DamageDealt = 0
		result.CombatLog = append(result.CombatLog, defender.ID+" evades the attack!")
		log.Printf("Combat: %s evades %s's attack.", defender.ID, attacker.ID)
		return result
	}

	// 2. Check for hit (using baseHitChance, can be modified by stats like accuracy/evasion)
	if rand.Float64() > ce.baseHitChance {
		result.DamageDealt = 0
		result.CombatLog = append(result.CombatLog, attacker.ID+" misses "+defender.ID+".")
		log.Printf("Combat: %s misses %s.", attacker.ID, defender.ID)
		return result
	}

	// 3. Calculate base damage
	// Simplified: damage = attacker's attack power - defender's defense
	// Ensure damage is not negative and respects minDamagePercentage
	baseDamage := attacker.AttackPower - defender.Defense
	minDamage := int(float64(attacker.AttackPower) * ce.minDamagePercentage)
	if minDamage < 1 {
		minDamage = 1 // Always at least 1 damage if hit and not fully mitigated by minDamagePercentage rule.
	}

	if baseDamage < minDamage {
		baseDamage = minDamage
	}
	if baseDamage <= 0 && minDamage <= 0 { // If defense is extremely high and minDamage is 0 or less
		baseDamage = 0 // No damage dealt, but it was a hit.
		result.CombatLog = append(result.CombatLog, attacker.ID+" hits "+defender.ID+" but deals no damage (fully mitigated).")
	}

	// 4. Check for critical hit
	actualDamage := baseDamage
	if rand.Float64() < ce.baseCritChance { // Simplified crit check
		result.IsCriticalHit = true
		actualDamage = int(float64(baseDamage) * ce.critDamageBonus)
		result.CombatLog = append(result.CombatLog, "Critical Hit!")
		log.Printf("Combat: %s lands a CRITICAL HIT on %s.", attacker.ID, defender.ID)
	}

	result.DamageDealt = actualDamage
	result.DefenderHealth -= actualDamage
	if result.DefenderHealth < 0 {
		result.DefenderHealth = 0
	}
	result.IsDefenderDefeated = result.DefenderHealth == 0

	result.CombatLog = append(result.CombatLog, fmt.Sprintf("%s attacks %s for %d damage.", attacker.ID, defender.ID, actualDamage))
	result.CombatLog = append(result.CombatLog, fmt.Sprintf("%s's health is now %d/%d.", defender.ID, result.DefenderHealth, defender.MaxHealth))

	if result.IsDefenderDefeated {
		result.CombatLog = append(result.CombatLog, defender.ID+" has been defeated!")
		log.Printf("Combat: %s has defeated %s.", attacker.ID, defender.ID)
	}

	log.Printf("Combat turn result for %s vs %s: Damage: %d, Defender HP: %d. Log: %v",
		attacker.ID, defender.ID, result.DamageDealt, result.DefenderHealth, result.CombatLog)

	// Record combat results on Sui blockchain if service is available
	if ce.suiCombatService != nil && result.IsDefenderDefeated { // Example: Record only if someone is defeated
		go func(combatOutcome CombatResult) {
			// Prepare data for Sui. This needs to match CombatResultData in sui package
			// and the expected arguments of the Move contract.
			suiCombatData := sui.CombatResultData{
				CombatLogID:   fmt.Sprintf("%s_vs_%s_%d", combatOutcome.AttackerID, combatOutcome.DefenderID, time.Now().UnixNano()), // Generate a unique ID
				WinnerAddress: combatOutcome.AttackerID,                                                                              // Assuming attacker wins if defender is defeated
				LoserAddress:  combatOutcome.DefenderID,
				Rewards:       map[string]interface{}{"xp_gained": 100, "items_dropped": "none"}, // Placeholder rewards
				AdditionalData: map[string]interface{}{
					"damage_dealt":       combatOutcome.DamageDealt,
					"final_health_c1":    combatOutcome.AttackerHealth, // This might be the attacker's health before this turn
					"final_health_c2":    combatOutcome.DefenderHealth,
					"combat_log_onchain": combatOutcome.CombatLog, // Consider if full log is needed on-chain
				},
			}
			// Gas budget for the transaction (example value)
			var gasBudget uint64 = 10000000

			// The RecordCombatOutcome now returns TransactionBlockResponse (txBytes)
			// The actual signing and execution should happen elsewhere, e.g., by a dedicated transaction manager
			// or directly by a service that has access to a signer.
			// For this example, we'll just log the preparation.
			txBlockResponse, err := ce.suiCombatService.RecordCombatOutcome(suiCombatData, gasBudget)
			if err != nil {
				log.Printf("Error PREPARING transaction for combat result on Sui (%s vs %s): %v",
					combatOutcome.AttackerID, combatOutcome.DefenderID, err)
			} else {
				log.Printf("Transaction for combat result (%s vs %s) PREPARED. TxBytes: %s. Digest (if any from prep): %s",
					combatOutcome.AttackerID, combatOutcome.DefenderID, txBlockResponse.TxBytes, txBlockResponse.Digest)
				// In a real system:
				// 1. Get txBlockResponse.TxBytes
				// 2. Sign these bytes with the appropriate private key (e.g., a server-held key for system transactions)
				// 3. Execute the signed transaction using suiClient.ExecuteTransactionBlock(signedTxBytes, signatures, ...)
				// For simplicity, this example does not implement signing and execution here.
			}
		}(*result) // Pass a copy of the result to the goroutine
	}

	return result
}

// Example: Simulate a full combat encounter until one combatant is defeated or max rounds reached
func (ce *CombatEngine) SimulateFullEncounter(combatant1, combatant2 CombatantStats, maxRounds int) []string {
	var overallCombatLog []string
	c1 := combatant1
	c2 := combatant2

	overallCombatLog = append(overallCombatLog, fmt.Sprintf("Encounter starts: %s (HP: %d) vs %s (HP: %d)", c1.ID, c1.Health, c2.ID, c2.Health))

	for round := 1; round <= maxRounds; round++ {
		overallCombatLog = append(overallCombatLog, fmt.Sprintf("\n--- Round %d ---", round))

		// Determine attack order (simplified: c1 then c2, could be speed-based)
		var turnResult *CombatResult

		// C1 attacks C2
		if c1.Health > 0 {
			turnResult = ce.SimulateCombatTurn(c1, c2)
			c2.Health = turnResult.DefenderHealth // Update c2's health from result
			overallCombatLog = append(overallCombatLog, turnResult.CombatLog...)
			if turnResult.IsDefenderDefeated {
				overallCombatLog = append(overallCombatLog, c1.ID+" wins the encounter!")
				break
			}
		}

		// C2 attacks C1 (if C2 is still alive)
		if c2.Health > 0 {
			turnResult = ce.SimulateCombatTurn(c2, c1)
			c1.Health = turnResult.DefenderHealth // Update c1's health from result
			overallCombatLog = append(overallCombatLog, turnResult.CombatLog...)
			if turnResult.IsDefenderDefeated {
				overallCombatLog = append(overallCombatLog, c2.ID+" wins the encounter!")
				break
			}
		}
		if c1.Health <= 0 || c2.Health <= 0 { // Should be caught by IsDefenderDefeated, but as a safeguard
			break
		}
		if round == maxRounds {
			overallCombatLog = append(overallCombatLog, "\nMax rounds reached. Combat ends.")
		}
	}
	return overallCombatLog
}
