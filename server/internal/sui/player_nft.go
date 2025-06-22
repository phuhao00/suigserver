package sui

import (
	"encoding/json"
	"fmt"
	// "log" // Will be replaced by utils
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/phuhao00/suigserver/server/internal/utils" // For logging
)

// PlayerNFTService interacts with the Player NFT contract on the Sui blockchain.
type PlayerNFTService struct {
	suiClient     *SuiClient // Use the refactored SuiClient
	packageID     string     // ID of the package containing the player NFT module
	moduleName    string     // Name of the Move module, e.g., "player_character"
	adminAddress  string     // Address with minting/admin capabilities for Player NFTs (if applicable)
	adminGasObjID string     // Gas object ID for admin operations
}

// NewPlayerNFTService creates a new PlayerNFTService.
func NewPlayerNFTService(suiClient *SuiClient, packageID, moduleName, adminAddress, adminGasObjID string) *PlayerNFTService {
	utils.LogInfo("Initializing Player NFT Service...")
	if suiClient == nil {
		utils.LogPanic("PlayerNFTService: SuiClient cannot be nil")
	}
	if packageID == "" || moduleName == "" {
		utils.LogPanic("PlayerNFTService: packageID and moduleName must be provided.")
	}
	// adminAddress and adminGasObjID are crucial for MintPlayerNFT if it's an admin action.
	// GetPlayerNFT and UpdatePlayerNFT (if owner-called) might not need them.
	// Consider validation in methods that specifically require them.
	return &PlayerNFTService{
		suiClient:     suiClient,
		packageID:     packageID,
		moduleName:    moduleName,
		adminAddress:  adminAddress,  // Required if minting is an admin action
		adminGasObjID: adminGasObjID, // Required if minting is an admin action
	}
}

// MintPlayerNFT prepares a transaction to mint a new Player NFT.
// This typically assigns the NFT to the `playerAddress`.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *PlayerNFTService) MintPlayerNFT(playerAddress string, initialAttributes map[string]interface{}, gasBudget uint64) (models.TransactionBlockResponse, error) {
	functionName := "mint_player_nft" // Assumed Move function name for minting a player NFT
	utils.LogInfof("PlayerNFTService: Preparing to mint Player NFT for address %s by admin %s. GasObject: %s, GasBudget: %d",
		playerAddress, s.adminAddress, s.adminGasObjID, gasBudget)

	if s.adminAddress == "" || s.adminGasObjID == "" {
		errMsg := "adminAddress and adminGasObjID must be configured in PlayerNFTService for minting"
		utils.LogError("PlayerNFTService: " + errMsg)
		return models.TransactionBlockResponse{}, fmt.Errorf(errMsg)
	}
	if playerAddress == "" {
		utils.LogError("PlayerNFTService: playerAddress must be provided for MintPlayerNFT")
		return models.TransactionBlockResponse{}, fmt.Errorf("playerAddress must be provided for MintPlayerNFT")
	}

	attributesJSON, err := json.Marshal(initialAttributes)
	if err != nil {
		utils.LogErrorf("PlayerNFTService: Failed to marshal initialAttributes for player %s: %v", playerAddress, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("failed to marshal initialAttributes to JSON: %w", err)
	}

	callArgs := []interface{}{
		playerAddress,          // The address to receive the Player NFT
		string(attributesJSON), // Initial attributes as a JSON string
		// Any other required args for minting, e.g., character class, starting zone ID, etc.
	}
	typeArgs := []string{} // If mint function is generic

	txBlockResponse, err := s.suiClient.MoveCall(
		s.adminAddress, // The address performing the mint (e.g., an admin or the system)
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		s.adminGasObjID, // Gas object owned by the admin/minter
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("PlayerNFTService: Error preparing MintPlayerNFT transaction for %s: %v", playerAddress, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for MintPlayerNFT (player %s): %w", playerAddress, err)
	}
	utils.LogInfof("PlayerNFTService: MintPlayerNFT transaction prepared for %s. TxBytes: %s", playerAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// GetPlayerNFT retrieves details of a Player NFT by its object ID.
func (s *PlayerNFTService) GetPlayerNFT(nftID string) (models.SuiObjectResponse, error) {
	utils.LogInfof("PlayerNFTService: Fetching Player NFT with ID %s.", nftID)
	if nftID == "" {
		utils.LogError("PlayerNFTService: nftID must be provided for GetPlayerNFT")
		return models.SuiObjectResponse{}, fmt.Errorf("nftID must be provided")
	}
	objectData, err := s.suiClient.GetObject(nftID)
	if err != nil {
		utils.LogErrorf("PlayerNFTService: Error fetching Player NFT object %s from Sui: %v", nftID, err)
		return models.SuiObjectResponse{}, fmt.Errorf("GetObject failed for PlayerNFT %s: %w", nftID, err)
	}
	utils.LogInfof("PlayerNFTService: Successfully fetched Player NFT object %s.", nftID)
	return objectData, nil
}

// UpdatePlayerNFT prepares a transaction to update mutable aspects of a Player NFT (e.g., experience, level, stats).
// `playerAddress` (owner of the NFT) must be the signer of this transaction.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *PlayerNFTService) UpdatePlayerNFT(nftID string, playerAddress string, updates map[string]interface{}, playerGasObjID string, gasBudget uint64) (models.TransactionBlockResponse, error) {
	functionName := "update_player_nft" // Assumed Move function name for updating player NFT
	utils.LogInfof("PlayerNFTService: Preparing to update Player NFT %s by owner %s with data %v. GasObject: %s, GasBudget: %d",
		nftID, playerAddress, updates, playerGasObjID, gasBudget)

	if nftID == "" || playerAddress == "" || playerGasObjID == "" {
		errMsg := "nftID, playerAddress, and playerGasObjID must be provided for UpdatePlayerNFT"
		utils.LogError("PlayerNFTService: " + errMsg)
		return models.TransactionBlockResponse{}, fmt.Errorf(errMsg)
	}
	if len(updates) == 0 {
		utils.LogWarn("PlayerNFTService: UpdatePlayerNFT called with empty updates map.")
		// Depending on contract logic, this might be an error or a no-op.
		// For now, proceed, but contract might require non-empty updates.
	}

	updatesJSON, err := json.Marshal(updates)
	if err != nil {
		utils.LogErrorf("PlayerNFTService: Failed to marshal updates for NFT %s: %v", nftID, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("failed to marshal updates to JSON: %w", err)
	}

	callArgs := []interface{}{
		nftID,               // The Player NFT object to update
		string(updatesJSON), // Updates as a JSON string, assuming Move function parses it
		// Other args like specific fields to update if not using a generic JSON approach.
	}
	typeArgs := []string{}

	txBlockResponse, err := s.suiClient.MoveCall(
		playerAddress, // Signer (owner of the NFT or an authorized address like a session key)
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		playerGasObjID, // Gas object owned by `playerAddress`
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("PlayerNFTService: Error preparing UpdatePlayerNFT transaction for %s: %v", nftID, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for UpdatePlayerNFT (NFT %s): %w", nftID, err)
	}
	utils.LogInfof("PlayerNFTService: UpdatePlayerNFT transaction prepared for %s. TxBytes: %s", nftID, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}
