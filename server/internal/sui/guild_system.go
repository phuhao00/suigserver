package sui

import (
	"fmt"
	"log"

	"github.com/block-vision/sui-go-sdk/models"
)

// GuildSystemSuiService interacts with the Guild System contract on the Sui blockchain.
type GuildSystemSuiService struct {
	suiClient  *SuiClient // Use the refactored SuiClient
	packageID  string     // ID of the package containing the guild module
	moduleName string     // Name of the Move module, e.g., "player_guild"
}

// NewGuildSystemSuiService creates a new GuildSystemSuiService.
func NewGuildSystemSuiService(client *SuiClient, packageID, moduleName string) *GuildSystemSuiService {
	log.Println("Initializing Guild System Sui Service...")
	if client == nil {
		log.Panic("GuildSystemSuiService: Sui client cannot be nil")
	}
	if packageID == "" || moduleName == "" {
		log.Panic("GuildSystemSuiService: packageID and moduleName must be provided.")
	}
	return &GuildSystemSuiService{
		suiClient:  client,
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
	log.Printf("Player %s preparing to create guild '%s' (%s). Package: %s, Module: %s",
		leaderAddress, guildName, guildSymbol, s.packageID, s.moduleName)

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
		log.Printf("Error preparing CreateGuild transaction for '%s': %v", guildName, err)
		return models.TransactionBlockResponse{}, err
	}
	log.Printf("CreateGuild transaction prepared for '%s'. TxBytes: %s",
		guildName, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// GetGuildInfo retrieves details of a guild from the blockchain by its object ID.
func (s *GuildSystemSuiService) GetGuildInfo(guildObjectID string) (models.SuiObjectResponse, error) {
	log.Printf("Fetching guild info for object ID %s.", guildObjectID)
	objectData, err := s.suiClient.GetObject(guildObjectID)
	if err != nil {
		log.Printf("Error fetching guild object %s from Sui: %v", guildObjectID, err)
		return models.SuiObjectResponse{}, err
	}
	log.Printf("Successfully fetched guild object %s.", guildObjectID)
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
	log.Printf("User %s preparing to add player %s to guild %s.", requesterAddress, playerAddress, guildObjectID)

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
		log.Printf("Error preparing AddMember transaction (Guild: %s, Player: %s): %v", guildObjectID, playerAddress, err)
		return models.TransactionBlockResponse{}, err
	}
	log.Printf("AddMember transaction prepared (Guild: %s, Player: %s). TxBytes: %s",
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
	log.Printf("User %s preparing to remove player %s from guild %s.", requesterAddress, playerAddress, guildObjectID)

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
		log.Printf("Error preparing RemoveMember transaction (Guild: %s, Player: %s): %v", guildObjectID, playerAddress, err)
		return models.TransactionBlockResponse{}, err
	}
	log.Printf("RemoveMember transaction prepared (Guild: %s, Player: %s). TxBytes: %s",
		guildObjectID, playerAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// --- Placeholder functions for other guild operations ---

// UpdateGuildDescription prepares a transaction to update a guild's description.
func (s *GuildSystemSuiService) UpdateGuildDescription(
	officerAddress string, // Signer
	guildObjectID string,
	newDescription string,
	officerGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "update_guild_description" // Assumed Move function name
	log.Printf("Officer %s preparing to update description for guild %s.", officerAddress, guildObjectID)
	callArgs := []interface{}{guildObjectID, newDescription}
	typeArgs := []string{}
	// This is a simplified example. Actual implementation might require specific capabilities.
	return s.suiClient.MoveCall(officerAddress, s.packageID, s.moduleName, functionName, typeArgs, callArgs, officerGasObjectID, gasBudget)
}

// PromoteMember prepares a transaction to promote a guild member.
func (s *GuildSystemSuiService) PromoteMember(
	officerAddress string, // Signer
	guildObjectID string,
	memberAddress string,
	newRank string, // Or an enum/u8 representing the rank
	officerGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	functionName := "promote_member" // Assumed Move function name
	log.Printf("Officer %s preparing to promote member %s in guild %s to rank %s.", officerAddress, memberAddress, guildObjectID, newRank)
	callArgs := []interface{}{guildObjectID, memberAddress, newRank}
	typeArgs := []string{}
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
	log.Printf("Officer %s preparing to demote member %s in guild %s to rank %s.", officerAddress, memberAddress, guildObjectID, newRank)
	callArgs := []interface{}{guildObjectID, memberAddress, newRank}
	typeArgs := []string{}
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
	log.Printf("Leader %s preparing to transfer leadership of guild %s to %s.", currentLeaderAddress, guildObjectID, newLeaderAddress)
	callArgs := []interface{}{guildObjectID, newLeaderAddress} // currentLeaderAddress is implicit as signer
	typeArgs := []string{}
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
	log.Printf("Leader %s preparing to disband guild %s.", leaderAddress, guildObjectID)
	callArgs := []interface{}{guildObjectID}
	typeArgs := []string{}
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
	log.Printf("Officer %s performing '%s' for item/coin %s (amount %d) in guild %s bank.",
		officerAddress, actionType, itemOrCoinID, amount, guildObjectID)

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
		return models.TransactionBlockResponse{}, fmt.Errorf("unsupported guild bank actionType: %s", actionType)
	}

	typeArgs := []string{} // May need type args if bank functions are generic (e.g. for Coin<T>)

	return s.suiClient.MoveCall(officerAddress, s.packageID, s.moduleName, functionName, typeArgs, callArgs, officerGasObjectID, gasBudget)
}
