package sui

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	// "github.com/tidwall/gjson" // Will be removed as gjson.Result is no longer returned by CallMoveFunction
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/phuhao00/suigserver/server/internal/utils"
)

// ProposalData defines the structure for creating a new governance proposal.
type ProposalData struct {
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	ActionType    string                 `json:"action_type"` // e.g., "UpdateGameParameter", "CommunityPoll"
	ActionPayload map[string]interface{} `json:"action_payload"` // Data specific to the action
	VotingPeriod  uint64                 `json:"voting_period_epochs"` // Duration in epochs
}

// GovernanceSuiService interacts with the Governance contract(s) on the Sui blockchain.
type GovernanceSuiService struct {
	suiClient     *SuiClient // Changed from *Client to *SuiClient
	packageID     string     // ID of the package containing the governance module
	moduleName    string     // Name of the Move module, e.g., "dao_governance"
	adminAddress  string     // An admin address if needed for some operations (e.g. initial setup)
	gasObjectID   string     // Default gas object for transactions sent by this service (e.g. proposal execution)
}

// NewGovernanceSuiService creates a new GovernanceSuiService.
func NewGovernanceSuiService(suiClient *SuiClient, packageID, moduleName, adminAddress, gasObjectID string) *GovernanceSuiService {
	log.Println("Initializing Governance Sui Service...")
	if suiClient == nil {
		utils.LogError("GovernanceSuiService: SuiClient cannot be nil") // Changed log.Panic to utils.LogError
		// Depending on desired behavior, might return nil or an error
		panic("GovernanceSuiService: SuiClient cannot be nil") // Or handle error more gracefully
	}
	if packageID == "" || moduleName == "" { // adminAddress and gasObjectID might be optional for read-only ops
		utils.LogError("GovernanceSuiService: packageID and moduleName must be provided.")
		panic("GovernanceSuiService: packageID and moduleName must be provided.") // Or handle error
	}
	return &GovernanceSuiService{
		suiClient:     suiClient, // Corrected from client to suiClient
		packageID:     packageID,
		moduleName:    moduleName,
		adminAddress:  adminAddress,
		gasObjectID:   gasObjectID,
	}
}

// CreateProposal prepares a transaction to submit a new governance proposal.
// The `proposerAddress` is the one initiating and signing the transaction.
func (s *GovernanceSuiService) CreateProposal(
	proposerAddress string, // Address of the user creating the proposal
	proposal ProposalData,
	proposerGasObjectID string, // Gas object owned by the proposer
	gasBudget uint64,
) (models.TransactionBlockResponse, error) { // Return type changed to models.TransactionBlockResponse
	functionName := "create_proposal"
	utils.LogInfof("GovernanceService: Player %s preparing to create governance proposal '%s'. Package: %s, Module: %s",
		proposerAddress, proposal.Title, s.packageID, s.moduleName)

	payloadJSON, err := json.Marshal(proposal.ActionPayload)
	if err != nil {
		utils.LogErrorf("GovernanceService: Failed to marshal proposal action payload for '%s': %v", proposal.Title, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("failed to marshal proposal action payload: %w", err)
	}

	callArgs := []interface{}{
		proposal.Title,
		proposal.Description,
		proposal.ActionType,
		string(payloadJSON),
		strconv.FormatUint(proposal.VotingPeriod, 10),
		// May require a "treasury_cap" or "dao_object_id" if proposals are associated with a specific DAO
	}
	typeArgs := []string{} // If proposal action involves specific types

	// Directly use suiClient.MoveCall (as CallMoveFunction is deprecated and suiClient is now *SuiClient)
	txBlockResponse, err := s.suiClient.MoveCall(
		proposerAddress, // Transaction sender is the proposer
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		proposerGasObjectID, // Proposer pays gas for creating proposal
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("GovernanceService: Error preparing CreateProposal transaction for '%s': %v", proposal.Title, err)
		return txBlockResponse, fmt.Errorf("CreateProposal MoveCall failed for %s: %w", proposal.Title, err)
	}
	utils.LogInfof("GovernanceService: CreateProposal transaction prepared for '%s'. TxBytes: %s",
		proposal.Title, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// VoteOnProposal prepares a transaction for a player to vote on a proposal.
// `proposalObjectID` is the on-chain ID of the proposal object.
// `voterGasObjectID` is a gas object owned by the voter.
func (s *GovernanceSuiService) VoteOnProposal(
	voterAddress string,
	proposalObjectID string,
	voteOption bool, // true for Yes/For, false for No/Against
	voterGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) { // Return type changed to models.TransactionBlockResponse
	functionName := "cast_vote"
	utils.LogInfof("GovernanceService: Player %s preparing to vote on proposal %s (Option: %t).", voterAddress, proposalObjectID, voteOption)

	callArgs := []interface{}{
		proposalObjectID, // The proposal object itself or its ID
		voteOption,
		// May require a "voting_rights_nft_id" or "governance_token_coin_id" to prove eligibility
	}
	typeArgs := []string{} // If vote involves specific token types

	txBlockResponse, err := s.suiClient.MoveCall( // Changed to MoveCall
		voterAddress, // Voter is the sender
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		voterGasObjectID, // Voter pays gas for their vote
		gasBudget,
	)
	if err != nil {
		utils.LogErrorf("GovernanceService: Error preparing VoteOnProposal transaction for proposal %s by %s: %v", proposalObjectID, voterAddress, err)
		return txBlockResponse, fmt.Errorf("VoteOnProposal MoveCall failed for %s by %s: %w", proposalObjectID, voterAddress, err)
	}
	utils.LogInfof("GovernanceService: VoteOnProposal transaction prepared for proposal %s by %s. TxBytes: %s",
		proposalObjectID, voterAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// GetProposalDetails retrieves the current status and details of a governance proposal object.
func (s *GovernanceSuiService) GetProposalDetails(proposalObjectID string) (models.SuiObjectResponse, error) { // Return type changed
	utils.LogInfof("GovernanceService: Fetching details for proposal object %s.", proposalObjectID)
	// This typically involves fetching a specific Sui object that represents the proposal.
	objectData, err := s.suiClient.GetObject(proposalObjectID) // GetObject already returns models.SuiObjectResponse
	if err != nil {
		utils.LogErrorf("GovernanceService: Error fetching proposal object %s from Sui: %v", proposalObjectID, err)
		return models.SuiObjectResponse{}, fmt.Errorf("GetObject failed for proposal %s: %w", proposalObjectID, err)
	}
	utils.LogInfof("GovernanceService: Successfully fetched proposal object %s.", proposalObjectID)
	// The actual details are within objectData.Data.Content.Fields or similar path if using SuiObjectResponse directly.
	// If previous code relied on gjson.Result's .Raw or .Get("data.content.fields"), adjustments will be needed in the calling code.
	return objectData, nil
}

// ExecuteProposal prepares a transaction to trigger the execution of an approved proposal.
// `executorAddress` is the address initiating the execution (might require specific permissions or be the DAO itself).
// `proposalObjectID` is the ID of the proposal to execute.
func (s *GovernanceSuiService) ExecuteProposal(
	executorAddress string, // Could be s.adminAddress or any permitted address
	proposalObjectID string,
	executorGasObjectID string,
	gasBudget uint64,
) (models.TransactionBlockResponse, error) { // Return type changed to models.TransactionBlockResponse
	functionName := "execute_proposal"
	utils.LogInfof("GovernanceService: User %s attempting to execute proposal %s.", executorAddress, proposalObjectID)

	callArgs := []interface{}{
		proposalObjectID,
		// May require other arguments like "dao_object_id" or "treasury_cap_id"
	}
	typeArgs := []string{}

	txBlockResponse, err := s.suiClient.MoveCall( // Changed to MoveCall
		executorAddress,
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		executorGasObjectID,
		gasBudget,
	)
	if err != nil {
		utils.LogErrorf("GovernanceService: Error preparing ExecuteProposal transaction for proposal %s by %s: %v", proposalObjectID, executorAddress, err)
		return txBlockResponse, fmt.Errorf("ExecuteProposal MoveCall failed for %s by %s: %w", proposalObjectID, executorAddress, err)
	}
	utils.LogInfof("GovernanceService: ExecuteProposal transaction prepared for proposal %s by %s. TxBytes: %s",
		proposalObjectID, executorAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}
