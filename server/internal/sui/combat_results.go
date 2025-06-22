package sui

import (
	"encoding/json"
	"fmt"
	// "log" // Will be replaced by utils
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/phuhao00/suigserver/server/internal/utils" // For logging
	// "github.com/phuhao00/suigserver/server/internal/game" // For CombatResult struct if needed
)

// CombatResultData defines the structure for combat outcome to be sent to Sui.
// This should mirror the expected arguments for your Move contract function.
type CombatResultData struct {
	CombatLogID    string                 `json:"combat_log_id"` // A unique identifier for this combat
	WinnerAddress  string                 `json:"winner_address"`
	LoserAddress   string                 `json:"loser_address"`
	Rewards        map[string]interface{} `json:"rewards"`         // e.g., {"nft_id": "0x...", "token_amount": 100}
	AdditionalData map[string]interface{} `json:"additional_data"` // Any other relevant data
}

// CombatResultsSuiService interacts with the Combat Results contract on the Sui blockchain.
type CombatResultsSuiService struct {
	suiClient     *SuiClient // Use the refactored SuiClient
	packageID     string     // ID of the package containing the combat results module
	moduleName    string     // Name of the Move module, e.g., "combat_results"
	senderAddress string     // Address of the account sending the transaction
	gasObjectID   string     // Object ID of the gas coin for transactions
}

// NewCombatResultsSuiService creates a new CombatResultsSuiService.
// These parameters would typically come from a configuration file.
func NewCombatResultsSuiService(suiClient *SuiClient, packageID, moduleName, senderAddress, gasObjectID string) *CombatResultsSuiService {
	utils.LogInfo("Initializing Combat Results Sui Service...")
	if suiClient == nil {
		utils.LogPanic("CombatResultsSuiService: SuiClient cannot be nil")
	}
	if packageID == "" || moduleName == "" || senderAddress == "" || gasObjectID == "" {
		utils.LogPanic("CombatResultsSuiService: packageID, moduleName, senderAddress, and gasObjectID must be provided.")
	}
	return &CombatResultsSuiService{
		suiClient:     suiClient,
		packageID:     packageID,
		moduleName:    moduleName,
		senderAddress: senderAddress,
		gasObjectID:   gasObjectID,
	}
}

// RecordCombatOutcome records the result of a combat encounter on the blockchain.
// It prepares and returns the transaction bytes. The actual signing and execution
// should be handled by the caller or a higher-level service.
func (s *CombatResultsSuiService) RecordCombatOutcome(data CombatResultData, gasBudget uint64) (models.TransactionBlockResponse, error) {
	functionName := "record_combat_outcome" // Name of the Move function
	utils.LogInfof("CombatResultsSuiService: Preparing to record combat outcome for log %s: Winner %s, Loser %s. Package: %s, Module: %s, Function: %s, GasBudget: %d, GasObject: %s",
		data.CombatLogID, data.WinnerAddress, data.LoserAddress, s.packageID, s.moduleName, functionName, gasBudget, s.gasObjectID)

	// Convert maps to JSON strings if the Move contract expects stringified JSON.
	// Adjust this based on the actual Move function signature.
	rewardsJSON, err := json.Marshal(data.Rewards)
	if err != nil {
		utils.LogErrorf("CombatResultsSuiService: Failed to marshal rewards to JSON for combat log %s: %v", data.CombatLogID, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("failed to marshal rewards to JSON: %w", err)
	}
	additionalDataJSON, err := json.Marshal(data.AdditionalData)
	if err != nil {
		utils.LogErrorf("CombatResultsSuiService: Failed to marshal additionalData to JSON for combat log %s: %v", data.CombatLogID, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("failed to marshal additionalData to JSON: %w", err)
	}

	callArgs := []interface{}{
		data.CombatLogID,
		data.WinnerAddress,
		data.LoserAddress,
		string(rewardsJSON),        // Pass as JSON string
		string(additionalDataJSON), // Pass as JSON string
	}
	typeArgs := []string{} // No type arguments for this example

	txBlockResponse, err := s.suiClient.MoveCall(
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
		utils.LogErrorf("CombatResultsSuiService: Error preparing transaction for RecordCombatOutcome (log %s): %v", data.CombatLogID, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for RecordCombatOutcome (log %s): %w", data.CombatLogID, err)
	}

	utils.LogInfof("CombatResultsSuiService: Transaction prepared for RecordCombatOutcome (log %s). TxBytes: %s",
		data.CombatLogID, txBlockResponse.TxBytes)
	// The caller is responsible for signing txBlockResponse.TxBytes and then executing it.
	return txBlockResponse, nil
}

// GetCombatOutcome retrieves the recorded outcome of a specific combat using an object ID.
// This assumes combat outcomes are stored as distinct objects on-chain.
func (s *CombatResultsSuiService) GetCombatOutcome(combatOutcomeObjectID string) (models.SuiObjectResponse, error) {
	utils.LogInfof("CombatResultsSuiService: Fetching combat outcome for object ID %s.", combatOutcomeObjectID)

	objectData, err := s.suiClient.GetObject(combatOutcomeObjectID)
	if err != nil {
		utils.LogErrorf("CombatResultsSuiService: Error fetching combat outcome object %s from Sui: %v", combatOutcomeObjectID, err)
		return models.SuiObjectResponse{}, fmt.Errorf("GetObject failed for combat outcome %s: %w", combatOutcomeObjectID, err)
	}
	utils.LogInfof("CombatResultsSuiService: Successfully fetched combat outcome object %s.", combatOutcomeObjectID)
	return objectData, nil
}
