package sui

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/block-vision/sui-go-sdk/models"
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
func NewItemNFTService(client *SuiClient, packageID, moduleName, adminAddress, adminGasObjID string) *ItemNFTService {
	log.Println("Initializing Item NFT Service...")
	if client == nil {
		log.Panic("ItemNFTService: Sui client cannot be nil")
	}
	if packageID == "" || moduleName == "" {
		log.Panic("ItemNFTService: packageID and moduleName must be provided.")
	}
	// adminAddress and adminGasObjID are needed for minting if it's an admin operation.
	// For transfers or updates initiated by owners, these specific admin ones might not be used directly in those calls.
	return &ItemNFTService{
		suiClient:     client,
		packageID:     packageID,
		moduleName:    moduleName,
		adminAddress:  adminAddress,
		adminGasObjID: adminGasObjID,
	}
}

// MintItemNFT prepares a transaction to mint a new Item NFT.
// Returns TransactionBlockResponse for subsequent signing and execution by the admin/minter.
func (s *ItemNFTService) MintItemNFT(itemType string, metadata map[string]interface{}, ownerAddress string, gasBudget uint64) (models.TransactionBlockResponse, error) {
	functionName := "mint_item_nft" // Assumed Move function name
	log.Printf("Preparing to mint Item NFT of type %s for %s by admin %s.", itemType, ownerAddress, s.adminAddress)

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return models.TransactionBlockResponse{}, fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}

	callArgs := []interface{}{
		itemType,             // e.g., "sword", "armor"
		string(metadataJSON), // Metadata as a JSON string
		ownerAddress,         // The intended recipient of the new NFT
	}
	typeArgs := []string{} // If mint function is generic

	if s.adminAddress == "" || s.adminGasObjID == "" {
		return models.TransactionBlockResponse{}, fmt.Errorf("adminAddress and adminGasObjID must be configured for minting")
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
		log.Printf("Error preparing MintItemNFT transaction (Type: %s for %s): %v", itemType, ownerAddress, err)
		return models.TransactionBlockResponse{}, err
	}
	log.Printf("MintItemNFT transaction prepared (Type: %s for %s). TxBytes: %s", itemType, ownerAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// GetItemNFT retrieves details of an Item NFT by its object ID.
func (s *ItemNFTService) GetItemNFT(nftID string) (models.SuiObjectResponse, error) {
	log.Printf("Fetching Item NFT with ID %s.", nftID)
	objectData, err := s.suiClient.GetObject(nftID)
	if err != nil {
		log.Printf("Error fetching Item NFT object %s from Sui: %v", nftID, err)
		return models.SuiObjectResponse{}, err
	}
	log.Printf("Successfully fetched Item NFT object %s.", nftID)
	return objectData, nil
}

// TransferItemNFT prepares a transaction to transfer an Item NFT to another player.
// `fromAddress` must be the signer of the transaction and own the `nftID` and `gasObjectID`.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *ItemNFTService) TransferItemNFT(nftID, fromAddress, toAddress, gasObjectID string, gasBudget uint64) (models.TransactionBlockResponse, error) {
	functionName := "transfer_item_nft" // Assumed Move function, often this is a generic `sui::transfer::public_transfer`
	log.Printf("Preparing to transfer Item NFT %s from %s to %s.", nftID, fromAddress, toAddress)

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
		log.Printf("Error preparing TransferItemNFT transaction (%s from %s to %s): %v", nftID, fromAddress, toAddress, err)
		return models.TransactionBlockResponse{}, err
	}
	log.Printf("TransferItemNFT transaction prepared (%s from %s to %s). TxBytes: %s", nftID, fromAddress, toAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// UpdateItemNFT prepares a transaction to update mutable aspects of an Item NFT (e.g., durability, stats).
// `ownerAddress` must be the signer and own the `nftID` and `gasObjectID`.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *ItemNFTService) UpdateItemNFT(nftID string, ownerAddress string, updates map[string]interface{}, gasObjectID string, gasBudget uint64) (models.TransactionBlockResponse, error) {
	functionName := "update_item_nft" // Assumed Move function name
	log.Printf("Preparing to update Item NFT %s by owner %s with data %v.", nftID, ownerAddress, updates)

	updatesJSON, err := json.Marshal(updates)
	if err != nil {
		return models.TransactionBlockResponse{}, fmt.Errorf("failed to marshal updates to JSON: %w", err)
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
		log.Printf("Error preparing UpdateItemNFT transaction for %s: %v", nftID, err)
		return models.TransactionBlockResponse{}, err
	}
	log.Printf("UpdateItemNFT transaction prepared for %s. TxBytes: %s", nftID, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}
