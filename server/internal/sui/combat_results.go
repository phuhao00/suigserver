package sui

import (
	"fmt"
	"log"

	"github.com/tidwall/gjson"
	// "github.com/phuhao00/suigserver/server/internal/game" // For CombatResult struct if needed
)

// CombatResultData defines the structure for combat outcome to be sent to Sui.
// This should mirror the expected arguments for your Move contract function.
type CombatResultData struct {
	CombatLogID    string                 `json:"combat_log_id"` // A unique identifier for this combat
	WinnerAddress  string                 `json:"winner_address"`
	LoserAddress   string                 `json:"loser_address"`
	Rewards        map[string]interface{} `json:"rewards"`        // e.g., {"nft_id": "0x...", "token_amount": 100}
	AdditionalData map[string]interface{} `json:"additional_data"` // Any other relevant data
}

// CombatResultsSuiService interacts with the Combat Results contract on the Sui blockchain.
type CombatResultsSuiService struct {
	suiClient     *Client // Assumes the unified Client from client.go
	packageID     string  // ID of the package containing the combat results module
	moduleName    string  // Name of the Move module, e.g., "combat_results"
	senderAddress string  // Address of the account sending the transaction
	gasObjectID   string  // Object ID of the gas coin for transactions
}

// NewCombatResultsSuiService creates a new CombatResultsSuiService.
// These parameters would typically come from a configuration file.
func NewCombatResultsSuiService(client *Client, packageID, moduleName, senderAddress, gasObjectID string) *CombatResultsSuiService {
	log.Println("Initializing Combat Results Sui Service...")
	if client == nil {
		log.Panic("CombatResultsSuiService: Sui client cannot be nil")
	}
	if packageID == "" || moduleName == "" || senderAddress == "" || gasObjectID == "" {
		log.Panic("CombatResultsSuiService: packageID, moduleName, senderAddress, and gasObjectID must be provided.")
	}
	return &CombatResultsSuiService{
		suiClient:     client,
		packageID:     packageID,
		moduleName:    moduleName,
		senderAddress: senderAddress,
		gasObjectID:   gasObjectID,
	}
}

// RecordCombatOutcome records the result of a combat encounter on the blockchain.
func (s *CombatResultsSuiService) RecordCombatOutcome(data CombatResultData, gasBudget uint64) (gjson.Result, error) {
	functionName := "record_combat_outcome" // Name of the Move function
	log.Printf("Recording combat outcome for log %s: Winner %s, Loser %s. Package: %s, Module: %s, Function: %s",
		data.CombatLogID, data.WinnerAddress, data.LoserAddress, s.packageID, s.moduleName, functionName)

	// TODO: Structure `callArgs` precisely according to the Move function's signature.
	// This is a placeholder and assumes the Move function takes these arguments directly.
	// Arguments might need to be strings, numbers, or specific Sui types (e.g., Object IDs as strings).
	// For complex structs, they might need to be passed as serialized BCS or JSON strings if the contract expects that,
	// or broken down into primitive arguments.
	callArgs := []interface{}{
		data.CombatLogID,
		data.WinnerAddress,
		data.LoserAddress,
		// Rewards and AdditionalData might need specific formatting for the Move contract.
		// For example, if rewards involve transferring objects, those object IDs would be arguments.
		// If it's just logging data, serializing them (e.g., to JSON string) might be an option if the contract handles it.
		// For simplicity, let's assume they are passed as is, but this needs verification with the contract.
		fmt.Sprintf("%v", data.Rewards),        // Placeholder: Convert map to string; contract needs to parse
		fmt.Sprintf("%v", data.AdditionalData), // Placeholder: Convert map to string
	}
	typeArgs := []string{} // No type arguments for this example

	// This call only prepares the transaction. Signing and execution are separate steps.
	// See client.go CallMoveFunction comments for the full lifecycle.
	txResult, err := s.suiClient.CallMoveFunction(
		s.senderAddress,
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		s.gasObjectID,
		gasBudget,
	)

	if err != nil {
		log.Printf("Error preparing transaction for RecordCombatOutcome (log %s): %v", data.CombatLogID, err)
		return txResult, err
	}

	log.Printf("Transaction prepared for RecordCombatOutcome (log %s). Digest (from unsafe_moveCall): %s",
		data.CombatLogID, txResult.Get("digest").String())
	// To complete: sign txResult.Get("txBytes").String() and then execute it.
	return txResult, nil
}

// GetCombatOutcome retrieves the recorded outcome of a specific combat using an object ID.
// This assumes combat outcomes are stored as distinct objects on-chain.
func (s *CombatResultsSuiService) GetCombatOutcome(combatOutcomeObjectID string) (gjson.Result, error) {
	log.Printf("Fetching combat outcome for object ID %s (Sui interaction placeholder).\n", combatOutcomeObjectID)

	// TODO: Implement actual Sui contract call or direct object query.
	// If combat outcomes are stored as objects, use suiClient.GetObject(combatOutcomeObjectID)
	// Example:
	// objectData, err := s.suiClient.GetObject(combatOutcomeObjectID)
	// if err != nil {
	// 	log.Printf("Error fetching combat outcome object %s from Sui: %v", combatOutcomeObjectID, err)
	// 	return gjson.Result{}, err
	// }
	// log.Printf("Successfully fetched combat outcome object %s: %s", combatOutcomeObjectID, objectData.Raw)
	// return objectData, nil

	// Placeholder response:
	mockData := fmt.Sprintf(`{
		"data": {
			"content": {
				"dataType": "moveObject",
				"type": "%s::%s::CombatOutcome",
				"fields": {
					"combat_log_id": "%s_retrieved",
					"winner_address": "mock_winner_addr_retrieved",
					"loser_address": "mock_loser_addr_retrieved",
					"rewards_summary": "some_rewards_info"
				},
				"has_public_transfer": true
			},
			"owner": {"AddressOwner": "%s"},
			"objectId": "%s"
		}
	}`, s.packageID, s.moduleName, combatOutcomeObjectID, s.senderAddress, combatOutcomeObjectID)
	log.Printf("Returning MOCK combat outcome data for %s", combatOutcomeObjectID)
	return gjson.Parse(mockData).Get("data"), nil
}
