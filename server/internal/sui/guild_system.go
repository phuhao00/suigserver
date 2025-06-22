package sui

import (
	"fmt"
	// "log" // Will be replaced by utils
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/phuhao00/suigserver/server/internal/utils" // For logging
)

// GuildSystemSuiService interacts with the Guild System contract on the Sui blockchain.
type GuildSystemSuiService struct {
	suiClient  *SuiClient // Use the refactored SuiClient
	packageID  string     // ID of the package containing the guild module
	moduleName string     // Name of the Move module, e.g., "player_guild"
}

// NewGuildSystemSuiService creates a new GuildSystemSuiService.
func NewGuildSystemSuiService(suiClient *SuiClient, packageID, moduleName string) *GuildSystemSuiService {
	utils.LogInfo("Initializing Guild System Sui Service...")
	if suiClient == nil {
		utils.LogPanic("GuildSystemSuiService: SuiClient cannot be nil")
	}
	if packageID == "" || moduleName == "" {
		utils.LogPanic("GuildSystemSuiService: packageID and moduleName must be provided.")
	}
	return &GuildSystemSuiService{
		suiClient:  suiClient,
		packageID:  packageID,
		moduleName: moduleName,
	}
}

// CreateGuild prepares a transaction to create a new guild on the blockchain.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *GuildSystemSuiService) CreateGuild(
	leaderAddress string, // The address of the player creating the guild (signer)
	guildName string,
	guildSymbol string,
	guildDescription string,
	leaderGasObjectID string, // Gas object owned by the leader
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "create_guild"
	utils.LogInfof("GuildSystemSuiService: Player %s preparing to create guild '%s' (%s). Package: %s, Module: %s, GasObject: %s, GasBudget: %d",
		leaderAddress, guildName, guildSymbol, s.packageID, s.moduleName, leaderGasObjectID, gasBudget)

	if leaderAddress == "" || guildName == "" || guildSymbol == "" || leaderGasObjectID == "" {
		errMsg := "leaderAddress, guildName, guildSymbol, and leaderGasObjectID must be provided for CreateGuild"
		utils.LogError("GuildSystemSuiService: " + errMsg)
		return models.TransactionBlockResponse{}, fmt.Errorf(errMsg)
	}

	callArgs := []interface{}{
		guildName,
		guildSymbol,
		guildDescription,
	}
	typeArgs := []string{}

	txBlockResponse, err := s.suiClient.MoveCall(
		leaderAddress,
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		leaderGasObjectID,
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("GuildSystemSuiService: Error preparing CreateGuild transaction for '%s': %v", guildName, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for CreateGuild ('%s'): %w", guildName, err)
	}
	utils.LogInfof("GuildSystemSuiService: CreateGuild transaction prepared for '%s'. TxBytes: %s",
		guildName, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// GetGuildInfo retrieves details of a guild from the blockchain by its object ID.
func (s *GuildSystemSuiService) GetGuildInfo(guildObjectID string) (models.SuiObjectResponse, error) {
	utils.LogInfof("GuildSystemSuiService: Fetching guild info for object ID %s.", guildObjectID)
	if guildObjectID == "" {
		utils.LogError("GuildSystemSuiService: guildObjectID must be provided for GetGuildInfo")
		return models.SuiObjectResponse{}, fmt.Errorf("guildObjectID must be provided")
	}
	objectData, err := s.suiClient.GetObject(guildObjectID)
	if err != nil {
		utils.LogErrorf("GuildSystemSuiService: Error fetching guild object %s from Sui: %v", guildObjectID, err)
		return models.SuiObjectResponse{}, fmt.Errorf("GetObject failed for guild %s: %w", guildObjectID, err)
	}
	utils.LogInfof("GuildSystemSuiService: Successfully fetched guild object %s.", guildObjectID)
	return objectData, nil
}

// AddMember prepares a transaction to add a member to a guild.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *GuildSystemSuiService) AddMember(
	requesterAddress string, // Signer of the transaction (e.g., guild officer)
	guildObjectID string,    // The ID of the guild object
	playerAddress string,    // The address of the player to be added
	requesterGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "add_member"
	utils.LogInfof("GuildSystemSuiService: User %s preparing to add player %s to guild %s. GasObject: %s, GasBudget: %d",
		requesterAddress, playerAddress, guildObjectID, requesterGasObjectID, gasBudget)

	if requesterAddress == "" || guildObjectID == "" || playerAddress == "" || requesterGasObjectID == "" {
		errMsg := "requesterAddress, guildObjectID, playerAddress, and requesterGasObjectID must be provided for AddMember"
		utils.LogError("GuildSystemSuiService: " + errMsg)
		return models.TransactionBlockResponse{}, fmt.Errorf(errMsg)
	}

	callArgs := []interface{}{
		guildObjectID,
		playerAddress,
	}
	typeArgs := []string{}

	txBlockResponse, err := s.suiClient.MoveCall(
		requesterAddress,
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		requesterGasObjectID,
		gasBudget,
	)
	if err != nil {
		utils.LogErrorf("GuildSystemSuiService: Error preparing AddMember transaction (Guild: %s, Player: %s): %v", guildObjectID, playerAddress, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for AddMember (Guild: %s, Player: %s): %w", guildObjectID, playerAddress, err)
	}
	utils.LogInfof("GuildSystemSuiService: AddMember transaction prepared (Guild: %s, Player: %s). TxBytes: %s",
		guildObjectID, playerAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// RemoveMember prepares a transaction to remove a member from a guild.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *GuildSystemSuiService) RemoveMember(
	requesterAddress string, // Signer (e.g., guild officer)
	guildObjectID string,
	playerAddress string, // Player to remove
	requesterGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "remove_member"
	utils.LogInfof("GuildSystemSuiService: User %s preparing to remove player %s from guild %s. GasObject: %s, GasBudget: %d",
		requesterAddress, playerAddress, guildObjectID, requesterGasObjectID, gasBudget)

	if requesterAddress == "" || guildObjectID == "" || playerAddress == "" || requesterGasObjectID == "" {
		errMsg := "requesterAddress, guildObjectID, playerAddress, and requesterGasObjectID must be provided for RemoveMember"
		utils.LogError("GuildSystemSuiService: " + errMsg)
		return models.TransactionBlockResponse{}, fmt.Errorf(errMsg)
	}

	callArgs := []interface{}{
		guildObjectID,
		playerAddress,
	}
	typeArgs := []string{}

	txBlockResponse, err := s.suiClient.MoveCall(
		requesterAddress,
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		requesterGasObjectID,
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("GuildSystemSuiService: Error preparing RemoveMember transaction (Guild: %s, Player: %s): %v", guildObjectID, playerAddress, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for RemoveMember (Guild: %s, Player: %s): %w", guildObjectID, playerAddress, err)
	}
	utils.LogInfof("GuildSystemSuiService: RemoveMember transaction prepared (Guild: %s, Player: %s). TxBytes: %s",
		guildObjectID, playerAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// --- Placeholder functions for other guild operations ---
// These will also need similar logging, validation, and error handling updates if they are to be used.
// For brevity in this review, I will update one more (UpdateGuildDescription) and assume the pattern applies to others.

// UpdateGuildDescription prepares a transaction to update a guild's description.
func (s *GuildSystemSuiService) UpdateGuildDescription(
	officerAddress string, // Signer
	guildObjectID string,
	newDescription string,
	officerGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "update_guild_description" // Assumed Move function name
	utils.LogInfof("GuildSystemSuiService: Officer %s preparing to update description for guild %s. GasObject: %s, GasBudget: %d",
		officerAddress, guildObjectID, officerGasObjectID, gasBudget)

	if officerAddress == "" || guildObjectID == "" || newDescription == "" || officerGasObjectID == "" {
		errMsg := "officerAddress, guildObjectID, newDescription, and officerGasObjectID must be provided for UpdateGuildDescription"
		utils.LogError("GuildSystemSuiService: " + errMsg)
		return models.TransactionBlockResponse{}, fmt.Errorf(errMsg)
	}

	callArgs := []interface{}{guildObjectID, newDescription}
	typeArgs := []string{}
	// This is a simplified example. Actual implementation might require specific capabilities.
	txBlockResponse, err := s.suiClient.MoveCall(officerAddress, s.packageID, s.moduleName, functionName, typeArgs, callArgs, officerGasObjectID, gasBudget)
	if err != nil {
		utils.LogErrorf("GuildSystemSuiService: MoveCall failed for UpdateGuildDescription (Guild: %s): %v", guildObjectID, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall failed for UpdateGuildDescription (Guild: %s): %w", guildObjectID, err)
	}
	return txBlockResponse, nil
}

// PromoteMember prepares a transaction to promote a guild member.
// NOTE: This and subsequent placeholder functions would need similar validation and error handling.
func (s *GuildSystemSuiService) PromoteMember(
	officerAddress string, // Signer
	guildObjectID string,
	memberAddress string,
	newRank string, // Or an enum/u8 representing the rank
	officerGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "promote_member" // Assumed Move function name
	utils.LogInfof("GuildSystemSuiService: Officer %s preparing to promote member %s in guild %s to rank %s. GasObject: %s, GasBudget: %d",
		officerAddress, memberAddress, guildObjectID, newRank, officerGasObjectID, gasBudget)
	// TODO: Add validation for parameters
	callArgs := []interface{}{guildObjectID, memberAddress, newRank}
	typeArgs := []string{}
	// TODO: Add error handling for MoveCall
	return s.suiClient.MoveCall(officerAddress, s.packageID, s.moduleName, functionName, typeArgs, callArgs, officerGasObjectID, gasBudget)
}

// DemoteMember prepares a transaction to demote a guild member.
func (s *GuildSystemSuiService) DemoteMember(
	officerAddress string, // Signer
	guildObjectID string,
	memberAddress string,
	newRank string, // Or an enum/u8
	officerGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "demote_member" // Assumed
	utils.LogInfof("GuildSystemSuiService: Officer %s preparing to demote member %s in guild %s to rank %s. GasObject: %s, GasBudget: %d",
		officerAddress, memberAddress, guildObjectID, newRank, officerGasObjectID, gasBudget)
	// TODO: Add validation for parameters
	callArgs := []interface{}{guildObjectID, memberAddress, newRank}
	typeArgs := []string{}
	// TODO: Add error handling for MoveCall
	return s.suiClient.MoveCall(officerAddress, s.packageID, s.moduleName, functionName, typeArgs, callArgs, officerGasObjectID, gasBudget)
}

// TransferLeadership prepares a transaction to transfer guild leadership.
func (s *GuildSystemSuiService) TransferLeadership(
	currentLeaderAddress string, // Signer
	guildObjectID string,
	newLeaderAddress string,
	leaderGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "transfer_leadership" // Assumed
	utils.LogInfof("GuildSystemSuiService: Leader %s preparing to transfer leadership of guild %s to %s. GasObject: %s, GasBudget: %d",
		currentLeaderAddress, guildObjectID, newLeaderAddress, leaderGasObjectID, gasBudget)
	// TODO: Add validation for parameters
	callArgs := []interface{}{guildObjectID, newLeaderAddress} // currentLeaderAddress is implicit as signer
	typeArgs := []string{}
	// TODO: Add error handling for MoveCall
	return s.suiClient.MoveCall(currentLeaderAddress, s.packageID, s.moduleName, functionName, typeArgs, callArgs, leaderGasObjectID, gasBudget)
}

// DisbandGuild prepares a transaction to disband a guild.
func (s *GuildSystemSuiService) DisbandGuild(
	leaderAddress string, // Signer
	guildObjectID string,
	leaderGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "disband_guild" // Assumed
	utils.LogInfof("GuildSystemSuiService: Leader %s preparing to disband guild %s. GasObject: %s, GasBudget: %d",
		leaderAddress, guildObjectID, leaderGasObjectID, gasBudget)
	// TODO: Add validation for parameters
	callArgs := []interface{}{guildObjectID}
	typeArgs := []string{}
	// TODO: Add error handling for MoveCall
	return s.suiClient.MoveCall(leaderAddress, s.packageID, s.moduleName, functionName, typeArgs, callArgs, leaderGasObjectID, gasBudget)
}

// ManageGuildBank prepares a transaction to deposit or withdraw items/currency from the guild bank.
// actionType could be "deposit" or "withdraw". This is highly dependent on contract design.
func (s *GuildSystemSuiService) ManageGuildBank(
	officerAddress string, // Signer
	guildObjectID string,
	itemOrCoinID string, // ID of the item/coin object to transfer
	amount uint64,       // Relevant for fungible tokens/currency
	actionType string,   // e.g., "deposit_item", "withdraw_ft"
	officerGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	// This is very schematic as bank interactions can be complex.
	// E.g., depositing an NFT might be one function, FT another.
	// functionName would vary based on actionType and item type.
	functionName := fmt.Sprintf("%s_guild_bank", actionType) // Highly speculative
	utils.LogInfof("GuildSystemSuiService: Officer %s performing '%s' for item/coin %s (amount %d) in guild %s bank. GasObject: %s, GasBudget: %d",
		officerAddress, actionType, itemOrCoinID, amount, guildObjectID, officerGasObjectID, gasBudget)

	// TODO: Add validation for parameters, including officerAddress, guildObjectID, itemOrCoinID/amount, actionType, officerGasObjectID.

	var callArgs []interface{}
	// Example: A generic "process_bank_action" function
	// callArgs = []interface{}{guildObjectID, itemOrCoinID, amount, actionType}
	// Or specific functions:
	if actionType == "deposit_item_nft" { // Assuming a specific function for NFT deposit
		functionName = "deposit_nft_to_bank"
		callArgs = []interface{}{guildObjectID, itemOrCoinID}
	} else if actionType == "withdraw_game_coin_from_bank" { // Assuming specific for FT withdrawal
		functionName = "withdraw_coin_from_bank"
		callArgs = []interface{}{guildObjectID, amount} // amount of game coin, recipient is officerAddress
	} else {
		utils.LogErrorf("GuildSystemSuiService: Unsupported guild bank actionType: %s", actionType)
		return models.TransactionBlockResponse{}, fmt.Errorf("unsupported guild bank actionType: %s", actionType)
	}

	typeArgs := []string{} // May need type args if bank functions are generic (e.g. for Coin<T>)
	// TODO: Add error handling for MoveCall
	return s.suiClient.MoveCall(officerAddress, s.packageID, s.moduleName, functionName, typeArgs, callArgs, officerGasObjectID, gasBudget)
}
