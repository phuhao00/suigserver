package sui

import (
	"encoding/json"
	"fmt"
	"log" // Will be replaced by utils.LogX

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/phuhao00/suigserver/server/internal/utils" // For logging
)

// ItemNFTService interacts with the Item/Equipment NFT contracts on the Sui blockchain.
type ItemNFTService struct {
	suiClient     *SuiClient // Use the refactored SuiClient
	packageID     string     // ID of the package containing the item NFT module
	moduleName    string     // Name of the Move module, e.g., "item_nft"
	adminAddress  string     // Address with minting/admin capabilities for NFTs (if centralized minting)
	adminGasObjID string     // Gas object ID for admin operations
}

// NewItemNFTService creates a new ItemNFTService.
// Parameters like packageID, moduleName, adminAddress, adminGasObjID would typically come from config.
func NewItemNFTService(suiClient *SuiClient, packageID, moduleName, adminAddress, adminGasObjID string) *ItemNFTService {
	utils.LogInfo("Initializing Item NFT Service...") // Changed to utils
	if suiClient == nil {
		log.Panic("ItemNFTService: SuiClient cannot be nil") // Changed to utils
	}
	if packageID == "" || moduleName == "" {
		log.Panic("ItemNFTService: packageID and moduleName must be provided.") // Changed to utils
	}
	// adminAddress and adminGasObjID are needed for minting if it's an admin operation.
	// For transfers or updates initiated by owners, these specific admin ones might not be used directly in those calls.
	return &ItemNFTService{
		suiClient:     suiClient, // Corrected: client to suiClient
		packageID:     packageID,
		moduleName:    moduleName,
		adminAddress:  adminAddress,
		adminGasObjID: adminGasObjID,
	}
}

// MintItemNFT prepares a transaction to mint a new Item NFT.
// Returns TransactionBlockResponse for subsequent signing and execution by the admin/minter.
func (s *ItemNFTService) MintItemNFT(itemType string, metadata map[string]interface{}, ownerAddress string, gasBudget uint64) (models.TxnMetaData, error) {
	functionName := "mint_item_nft" // Assumed Move function name
	utils.LogInfof("ItemNFTService: Preparing to mint Item NFT of type %s for %s by admin %s.", itemType, ownerAddress, s.adminAddress)

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		utils.LogErrorf("ItemNFTService: Failed to marshal metadata for minting item type %s: %v", itemType, err)
		return models.TxnMetaData{}, fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}

	callArgs := []interface{}{
		itemType,             // e.g., "sword", "armor"
		string(metadataJSON), // Metadata as a JSON string
		ownerAddress,         // The intended recipient of the new NFT
	}
	typeArgs := []string{} // If mint function is generic

	if s.adminAddress == "" || s.adminGasObjID == "" {
		utils.LogError("ItemNFTService: adminAddress and adminGasObjID must be configured for minting")
		return models.TxnMetaData{}, fmt.Errorf("adminAddress and adminGasObjID must be configured for minting")
	}

	txBlockResponse, err := s.suiClient.MoveCall(
		s.adminAddress, // Minter address
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		s.adminGasObjID, // Gas object owned by the minter
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("ItemNFTService: Error preparing MintItemNFT transaction (Type: %s for %s): %v", itemType, ownerAddress, err)
		return models.TxnMetaData{}, fmt.Errorf("MoveCall failed for MintItemNFT: %w", err)
	}
	utils.LogInfof("ItemNFTService: MintItemNFT transaction prepared (Type: %s for %s). TxBytes: %s", itemType, ownerAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// MintItemNFTAndExecute is an EXAMPLE function demonstrating the full flow:
// 1. Prepare Transaction (using MintItemNFT)
// 2. Sign Transaction (using conceptual server-side SignTransactionBytesWithServerKey)
// 3. Execute Transaction
// It requires serverPrivateKeyHex to be configured.
// WARNING: This is for demonstration. Private key management needs to be secure in production.
func (s *ItemNFTService) MintItemNFTAndExecute(
	itemType string,
	metadata map[string]interface{},
	ownerAddress string,
	gasBudget uint64,
	serverPrivateKeyHex string, // Server's private key for signing. HANDLE WITH EXTREME CARE.
) (models.SuiTransactionBlockResponse, error) {
	utils.LogInfof("ItemNFTService: Attempting to mint and execute Item NFT of type %s for %s", itemType, ownerAddress)

	// 1. Prepare the transaction
	txBlockResponse, err := s.MintItemNFT(itemType, metadata, ownerAddress, gasBudget)
	if err != nil {
		// MintItemNFT already logs the specific error
		return models.SuiTransactionBlockResponse{}, fmt.Errorf("failed to prepare MintItemNFT transaction: %w", err)
	}

	if txBlockResponse.TxBytes == "" {
		utils.LogError("ItemNFTService: Prepared transaction resulted in empty TxBytes.")
		return models.SuiTransactionBlockResponse{}, fmt.Errorf("prepared transaction resulted in empty TxBytes")
	}
	utils.LogDebugf("ItemNFTService: TxBytes for signing: %s", txBlockResponse.TxBytes)

	// 2. Sign the transaction (using conceptual server-side signing)
	//    In a real scenario, ensure serverPrivateKeyHex is securely managed.
	signature, err := SignTransactionBytesWithServerKey(txBlockResponse.TxBytes, serverPrivateKeyHex)
	if err != nil {
		utils.LogErrorf("ItemNFTService: Failed to sign transaction for MintItemNFT (Type: %s): %v", itemType, err)
		return models.SuiTransactionBlockResponse{}, fmt.Errorf("failed to sign transaction: %w", err)
	}
	utils.LogDebugf("ItemNFTService: Conceptual signature obtained: %s", signature)

	// 3. Execute the transaction
	utils.LogInfo("ItemNFTService: Executing transaction...")
	executeResponse, err := s.suiClient.ExecuteTransactionBlock(txBlockResponse.TxBytes, []string{signature})
	if err != nil {
		utils.LogErrorf("ItemNFTService: Failed to execute MintItemNFT transaction (Type: %s): %v", itemType, err)
		// Log more details from executeResponse if available, e.g., executeResponse.Error
		return models.SuiTransactionBlockResponse{}, fmt.Errorf("failed to execute transaction: %w", err)
	}

	// Check for errors in execution effects
	// The sui-go-sdk's ExecuteTransactionBlock already returns an error if the RPC call itself failed.
	// TODO: Fix effects structure access based on actual sui-go-sdk models

	utils.LogInfof("ItemNFTService: MintItemNFT transaction executed successfully (Type: %s for %s). Digest: %s",
		itemType, ownerAddress, executeResponse.Digest)

	// TODO: Log created objects when effects structure is clarified

	return executeResponse, nil
}

// GetItemNFT retrieves details of an Item NFT by its object ID.
func (s *ItemNFTService) GetItemNFT(nftID string) (models.SuiObjectResponse, error) {
	utils.LogInfof("ItemNFTService: Fetching Item NFT with ID %s.", nftID) // Changed log to utils
	objectData, err := s.suiClient.GetObject(nftID)
	if err != nil {
		utils.LogErrorf("ItemNFTService: Error fetching Item NFT object %s from Sui: %v", nftID, err) // Changed log to utils
		return models.SuiObjectResponse{}, fmt.Errorf("GetObject failed for ItemNFT %s: %w", nftID, err)
	}
	utils.LogInfof("ItemNFTService: Successfully fetched Item NFT object %s.", nftID) // Changed log to utils
	return objectData, nil
}

// TransferItemNFT prepares a transaction to transfer an Item NFT to another player.
// `fromAddress` must be the signer of the transaction and own the `nftID` and `gasObjectID`.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *ItemNFTService) TransferItemNFT(nftID, fromAddress, toAddress, gasObjectID string, gasBudget uint64) (models.TxnMetaData, error) {
	functionName := "transfer_item_nft" // Assumed Move function, often this is a generic `sui::transfer::public_transfer`
	utils.LogInfof("ItemNFTService: Preparing to transfer Item NFT %s from %s to %s. GasObject: %s", nftID, fromAddress, toAddress, gasObjectID)

	// For public_transfer, the arguments are typically the object itself and the recipient address.
	// The object being transferred (nftID) is usually the first argument to transfer functions or handled by PTB.
	callArgs := []interface{}{
		nftID,     // The NFT object to transfer
		toAddress, // The recipient's address
	}
	typeArgs := []string{} // May need type arg if the NFT type is generic in the transfer function.

	// If using a generic sui::transfer::public_transfer, the packageID and moduleName would be "0x2" and "transfer".
	// If it's a custom transfer function in your NFT module:
	txBlockResponse, err := s.suiClient.MoveCall(
		fromAddress,  // Signer of the transaction (current owner)
		s.packageID,  // Or "0x2" if using sui::transfer
		s.moduleName, // Or "transfer" if using sui::transfer
		functionName, // Or "public_transfer" if using sui::transfer
		typeArgs,
		callArgs,
		gasObjectID, // Gas object owned by `fromAddress`
		gasBudget,
	)
	// Note: For simple transfers, building a PTB with sui::transfer::transfer or public_transfer is more common
	// than having a dedicated `transfer_item_nft` in the item module, unless there are specific logics.

	if err != nil {
		utils.LogErrorf("ItemNFTService: Error preparing TransferItemNFT transaction (%s from %s to %s): %v", nftID, fromAddress, toAddress, err)
		return models.TxnMetaData{}, fmt.Errorf("MoveCall failed for TransferItemNFT: %w", err)
	}
	utils.LogInfof("ItemNFTService: TransferItemNFT transaction prepared (%s from %s to %s). TxBytes: %s", nftID, fromAddress, toAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// UpdateItemNFT prepares a transaction to update mutable aspects of an Item NFT (e.g., durability, stats).
// `ownerAddress` must be the signer and own the `nftID` and `gasObjectID`.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *ItemNFTService) UpdateItemNFT(nftID string, ownerAddress string, updates map[string]interface{}, gasObjectID string, gasBudget uint64) (models.TxnMetaData, error) {
	functionName := "update_item_nft" // Assumed Move function name
	utils.LogInfof("ItemNFTService: Preparing to update Item NFT %s by owner %s with data %v. GasObject: %s", nftID, ownerAddress, updates, gasObjectID)

	updatesJSON, err := json.Marshal(updates)
	if err != nil {
		utils.LogErrorf("ItemNFTService: Failed to marshal updates for NFT %s: %v", nftID, err)
		return models.TxnMetaData{}, fmt.Errorf("failed to marshal updates to JSON: %w", err)
	}

	callArgs := []interface{}{
		nftID,               // The NFT object to update
		string(updatesJSON), // Updates as a JSON string, assuming Move function parses it
		// Or, flatten `updates` into direct arguments if the Move function expects that.
	}
	typeArgs := []string{}

	txBlockResponse, err := s.suiClient.MoveCall(
		ownerAddress, // Signer (current owner or an authorized address)
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		gasObjectID, // Gas object owned by `ownerAddress`
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("ItemNFTService: Error preparing UpdateItemNFT transaction for %s: %v", nftID, err)
		return models.TxnMetaData{}, fmt.Errorf("MoveCall failed for UpdateItemNFT: %w", err)
	}
	utils.LogInfof("ItemNFTService: UpdateItemNFT transaction prepared for %s. TxBytes: %s", nftID, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}
