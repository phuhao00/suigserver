package sui

import (
	"fmt"
	"log"

	"github.com/tidwall/gjson"
)

// GuildSystemSuiService interacts with the Guild System contract on the Sui blockchain.
type GuildSystemSuiService struct {
	suiClient   *Client
	packageID   string // ID of the package containing the guild module
	moduleName  string // Name of the Move module, e.g., "player_guild"
	// senderAddress string // May not be needed if all calls are signed by users
	// gasObjectID string // Default gas object if service makes transactions
}

// NewGuildSystemSuiService creates a new GuildSystemSuiService.
func NewGuildSystemSuiService(client *Client, packageID, moduleName string) *GuildSystemSuiService {
	log.Println("Initializing Guild System Sui Service...")
	if client == nil {
		log.Panic("GuildSystemSuiService: Sui client cannot be nil")
	}
	if packageID == "" || moduleName == "" {
		log.Panic("GuildSystemSuiService: packageID and moduleName must be provided.")
	}
	return &GuildSystemSuiService{
		suiClient:   client,
		packageID:   packageID,
		moduleName:  moduleName,
	}
}

// CreateGuild prepares a transaction to create a new guild on the blockchain.
// This would involve minting a Guild NFT or a similar representative object.
// The `leaderAddress` is the sender of this transaction and pays for gas.
func (s *GuildSystemSuiService) CreateGuild(
	leaderAddress string, // The address of the player creating the guild
	guildName string,
	guildSymbol string, // e.g., a short tag for the guild
	guildDescription string,
	leaderGasObjectID string, // Gas object owned by the leader
	gasBudget uint64,
) (gjson.Result, error) {
	functionName := "create_guild"
	log.Printf("Player %s preparing to create guild '%s' (%s). Package: %s, Module: %s",
		leaderAddress, guildName, guildSymbol, s.packageID, s.moduleName)

	callArgs := []interface{}{
		guildName,
		guildSymbol,
		guildDescription,
		// Other args: initial members, guild settings, required item to create guild, etc.
	}
	typeArgs := []string{} // If the create_guild function is generic

	txResult, err := s.suiClient.CallMoveFunction(
		leaderAddress, // Sender is the guild leader
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		leaderGasObjectID, // Leader pays for guild creation
		gasBudget,
	)

	if err != nil {
		log.Printf("Error preparing CreateGuild transaction for '%s': %v", guildName, err)
		return txResult, err
	}
	log.Printf("CreateGuild transaction prepared for '%s'. Digest (from unsafe_moveCall): %s",
		guildName, txResult.Get("digest").String())
	// The response (txResult) will contain the txBytes. After signing and execution,
	// one of the created objects in the effects should be the new Guild object/NFT.
	return txResult, nil
}

// GetGuildInfo retrieves details of a guild from the blockchain by its object ID.
func (s *GuildSystemSuiService) GetGuildInfo(guildObjectID string) (gjson.Result, error) {
	log.Printf("Fetching guild info for object ID %s.", guildObjectID)
	// Assumes the guild is represented by a Sui object.
	objectData, err := s.suiClient.GetObject(guildObjectID)
	if err != nil {
		log.Printf("Error fetching guild object %s from Sui: %v", guildObjectID, err)
		return gjson.Result{}, err
	}
	log.Printf("Successfully fetched guild object %s: %s", guildObjectID, objectData.Raw)
	// Details are in objectData.Get("data.content.fields") or similar.
	return objectData, nil
}

// AddMember prepares a transaction to add a member to a guild.
// `requesterAddress` is the address initiating the action (e.g., guild officer or the player themselves if applying).
// `guildObjectID` is the ID of the guild object.
// `playerAddress` is the address of the player to be added.
func (s *GuildSystemSuiService) AddMember(
	requesterAddress string,
	guildObjectID string,
	playerAddress string,
	requesterGasObjectID string,
	gasBudget uint64,
) (gjson.Result, error) {
	functionName := "add_member" // Or "join_guild", "accept_application"
	log.Printf("User %s preparing to add player %s to guild %s.", requesterAddress, playerAddress, guildObjectID)

	callArgs := []interface{}{
		guildObjectID,   // The Guild object itself or its ID
		playerAddress, // The address of the player to add
		// May require a "member_nft_capability" or similar if adding members is restricted
	}
	typeArgs := []string{}

	txResult, err := s.suiClient.CallMoveFunction(
		requesterAddress, // Sender could be a guild officer or system with permissions
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
		return txResult, err
	}
	log.Printf("AddMember transaction prepared (Guild: %s, Player: %s). Digest: %s",
		guildObjectID, playerAddress, txResult.Get("digest").String())
	return txResult, nil
}

// RemoveMember prepares a transaction to remove a member from a guild.
func (s *GuildSystemSuiService) RemoveMember(
	requesterAddress string, // e.g., guild officer
	guildObjectID string,
	playerAddress string, // Player to remove
	requesterGasObjectID string,
	gasBudget uint64,
) (gjson.Result, error) {
	functionName := "remove_member"
	log.Printf("User %s preparing to remove player %s from guild %s.", requesterAddress, playerAddress, guildObjectID)

	callArgs := []interface{}{
		guildObjectID,
		playerAddress,
	}
	typeArgs := []string{}

	txResult, err := s.suiClient.CallMoveFunction(
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
		return txResult, err
	}
	log.Printf("RemoveMember transaction prepared (Guild: %s, Player: %s). Digest: %s",
		guildObjectID, playerAddress, txResult.Get("digest").String())
	return txResult, nil
}

// TODO: Add other guild operations as needed:
// - UpdateGuildDescription(guildID, newDescription, officerAddress, gas...)
// - PromoteMember(guildID, memberAddress, newRank, officerAddress, gas...)
// - DemoteMember(guildID, memberAddress, newRank, officerAddress, gas...)
// - TransferLeadership(guildID, currentLeaderAddress, newLeaderAddress, gas...)
// - DisbandGuild(guildID, leaderAddress, gas...)
// - ManageGuildBank(guildID, itemID, amount, actionType, officerAddress, gas...)
