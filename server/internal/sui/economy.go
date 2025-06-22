package sui

import (
	"fmt"
	"log"
	"strconv"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/phuhao00/suigserver/server/internal/utils" // For logging
)

// EconomySuiService interacts with the Economic System contracts on the Sui blockchain.
type EconomySuiService struct {
	suiClient     *SuiClient // Use the refactored SuiClient
	packageID     string     // ID of the package containing the economy/token module
	moduleName    string     // Name of the Move module, e.g., "game_coin"
	senderAddress string     // Address of the account sending transactions (e.g., a treasury or admin account for some ops)
	gasObjectID   string     // Object ID of the gas coin for transactions
}

// NewEconomySuiService creates a new EconomySuiService.
func NewEconomySuiService(suiClient *SuiClient, packageID, moduleName, senderAddress, gasObjectID string) *EconomySuiService {
	utils.LogInfo("Initializing Economy Sui Service...")
	if suiClient == nil {
		log.Panic("EconomySuiService: SuiClient cannot be nil")
	}
	if packageID == "" || moduleName == "" {
		// senderAddress and gasObjectID might be optional if the service instance is only used for reads like GetPlayerBalance
		// However, for consistency or if most operations require them, keeping the panic is fine.
		log.Panic("EconomySuiService: packageID and moduleName must be provided.")
	}
	return &EconomySuiService{
		suiClient:     suiClient,
		packageID:     packageID,
		moduleName:    moduleName,
		senderAddress: senderAddress, // Used as default sender for ops like Mint
		gasObjectID:   gasObjectID,   // Used as default gas for ops like Mint
	}
}

// GetPlayerBalance retrieves a player's balance for a specific on-chain coin type.
func (s *EconomySuiService) GetPlayerBalance(playerAddress string, coinType string) (uint64, error) {
	utils.LogInfof("EconomySuiService: Fetching balance for player %s, CoinType: %s", playerAddress, coinType)

	resp, err := s.suiClient.GetBalance(playerAddress, coinType)
	if err != nil {
		utils.LogErrorf("EconomySuiService: Error fetching balance for %s (CoinType: %s): %v", playerAddress, coinType, err)
		return 0, fmt.Errorf("GetBalance failed for player %s, coin %s: %w", playerAddress, coinType, err)
	}

	balance, err := strconv.ParseUint(resp.TotalBalance, 10, 64)
	if err != nil {
		utils.LogErrorf("EconomySuiService: Error parsing balance string '%s' for %s: %v", resp.TotalBalance, playerAddress, err)
		return 0, fmt.Errorf("could not parse balance '%s' from Sui response: %w", resp.TotalBalance, err)
	}

	utils.LogInfof("EconomySuiService: Balance for player %s (CoinType: %s): %d", playerAddress, coinType, balance)
	return balance, nil
}

// TransferTokens prepares a transaction to transfer on-chain tokens (Coins).
// This function now returns TransactionBlockResponse, which contains the txBytes.
// The caller is responsible for signing and executing the transaction.
// Note: The sui-go-sdk's PaySui/PayAllSui methods are more idiomatic for SUI transfers.
// For custom game tokens, a MoveCall to a specific transfer function in your contract is common.
// This example assumes a generic "transfer_coins" function in your module.
func (s *EconomySuiService) TransferTokens(
	fromAddress string, // Sender of the tokens (must sign the transaction)
	coinObjectIDs []string, // Specific coin objects to be spent.
	amount uint64,
	toAddress string,
	coinType string, // Fully qualified type of the coin
	fromGasObjectID string, // Gas object owned by fromAddress
	gasBudget uint64,
) (models.TxnMetaData, error) {
	// This function name should match a function in your Move contract.
	// For SUI, it's better to use PTB commands like SplitCoins and TransferObjects.
	// For a custom Coin<T>, you might have a `transfer(coin: Coin<T>, amount: u64, recipient: address)`
	// or `split_and_transfer(coins: vector<Coin<T>>, amount: u64, recipient: address)`
	// The example below assumes a function that takes coin object IDs.
	functionName := "transfer_game_coin" // Replace with your actual contract function
	utils.LogInfof("EconomySuiService: Preparing to transfer %d of %s from %s to %s using coins %v. FromGasObject: %s, GasBudget: %d",
		amount, coinType, fromAddress, toAddress, coinObjectIDs, fromGasObjectID, gasBudget)

	if fromGasObjectID == "" {
		utils.LogError("EconomySuiService: fromGasObjectID must be provided for TransferTokens by fromAddress.")
		return models.TxnMetaData{}, fmt.Errorf("fromGasObjectID must be provided for TransferTokens")
	}
	if len(coinObjectIDs) == 0 {
		utils.LogError("EconomySuiService: At least one coinObjectID must be provided for transfer.")
		return models.TxnMetaData{}, fmt.Errorf("at least one coinObjectID must be provided for transfer")
	}
	if fromAddress == "" || toAddress == "" {
		utils.LogError("EconomySuiService: fromAddress and toAddress must be provided for transfer.")
		return models.TxnMetaData{}, fmt.Errorf("fromAddress and toAddress must be provided for transfer")
	}

	// Arguments depend heavily on the Move function's signature.
	// Example: if your function takes a vector of coin IDs, amount, and recipient:
	callArgs := []interface{}{
		coinObjectIDs,                  // E.g., a vector of object IDs for the coins to use
		strconv.FormatUint(amount, 10), // amount
		toAddress,                      // recipient
	}
	// If your function is generic over the coin type T (e.g., transfer<T>(...))
	typeArgs := []string{coinType}

	// The `fromAddress` must be the signer. `s.gasObjectID` should be owned by `fromAddress`.
	txBlockResponse, err := s.suiClient.MoveCall(
		fromAddress,     // Signer and owner of coins/gas
		s.packageID,     // Package of the coin/transfer logic
		s.moduleName,    // Module containing the transfer function
		functionName,    // The actual function in your Move contract
		typeArgs,        // Type arguments, if any
		callArgs,        // Arguments for the Move function
		fromGasObjectID, // Corrected: Gas object owned by `fromAddress`
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("EconomySuiService: Error preparing token transfer from %s to %s: %v", fromAddress, toAddress, err)
		return models.TxnMetaData{}, fmt.Errorf("MoveCall failed for token transfer from %s to %s: %w", fromAddress, toAddress, err)
	}
	utils.LogInfof("EconomySuiService: Token transfer transaction prepared from %s to %s. TxBytes: %s",
		fromAddress, toAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// MintGameTokens prepares a transaction to mint new game tokens.
// This is an administrative action performed by s.senderAddress, using s.gasObjectID.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *EconomySuiService) MintGameTokens(recipientAddress string, amount uint64, gasBudget uint64) (models.TxnMetaData, error) {
	functionName := "mint_game_tokens" // Placeholder for your Move mint function
	utils.LogInfof("EconomySuiService: Preparing to mint %d game tokens to %s. Admin sender: %s, GasObject: %s, GasBudget: %d",
		amount, recipientAddress, s.senderAddress, s.gasObjectID, gasBudget)

	if s.senderAddress == "" || s.gasObjectID == "" {
		utils.LogError("EconomySuiService: senderAddress (admin) and gasObjectID must be configured in the service for MintGameTokens.")
		return models.TxnMetaData{}, fmt.Errorf("senderAddress and gasObjectID must be configured for MintGameTokens")
	}
	if recipientAddress == "" {
		utils.LogError("EconomySuiService: recipientAddress must be provided for MintGameTokens.")
		return models.TxnMetaData{}, fmt.Errorf("recipientAddress must be provided for MintGameTokens")
	}

	callArgs := []interface{}{
		recipientAddress,
		strconv.FormatUint(amount, 10),
	}
	typeArgs := []string{} // Assuming coin type is fixed in the mint function

	txBlockResponse, err := s.suiClient.MoveCall(
		s.senderAddress, // Admin/Treasury address with minting capability
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		s.gasObjectID, // Gas object owned by the admin/treasury sender
		gasBudget,
	)

	if err != nil {
		utils.LogErrorf("EconomySuiService: Error preparing mint_game_tokens transaction for %s: %v", recipientAddress, err)
		return models.TxnMetaData{}, fmt.Errorf("MoveCall failed for MintGameTokens for %s: %w", recipientAddress, err)
	}
	utils.LogInfof("EconomySuiService: Mint game tokens transaction prepared for %s. TxBytes: %s",
		recipientAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// BurnGameTokens prepares a transaction to burn game tokens.
// Returns TransactionBlockResponse for subsequent signing and execution.
// The `burnerGasObjectID` must be owned by `burnerAddress`.
func (s *EconomySuiService) BurnGameTokens(burnerAddress string, tokenObjectIDs []string, burnerGasObjectID string, gasBudget uint64) (models.TxnMetaData, error) {
	functionName := "burn_game_tokens" // Placeholder for your Move burn function
	utils.LogInfof("EconomySuiService: Preparing to burn game tokens (IDs: %v) from %s. GasObject: %s, GasBudget: %d",
		tokenObjectIDs, burnerAddress, burnerGasObjectID, gasBudget)

	if burnerGasObjectID == "" {
		utils.LogError("EconomySuiService: burnerGasObjectID must be provided for BurnGameTokens.")
		return models.TxnMetaData{}, fmt.Errorf("burnerGasObjectID must be provided for BurnGameTokens")
	}
	if len(tokenObjectIDs) == 0 {
		utils.LogError("EconomySuiService: At least one tokenObjectID must be provided to burn.")
		return models.TxnMetaData{}, fmt.Errorf("at least one tokenObjectID must be provided to burn")
	}
	if burnerAddress == "" {
		utils.LogError("EconomySuiService: burnerAddress must be provided for BurnGameTokens.")
		return models.TxnMetaData{}, fmt.Errorf("burnerAddress must be provided for BurnGameTokens")
	}

	callArgs := []interface{}{
		tokenObjectIDs, // This would likely be a vector of Coin objects or their IDs
	}
	typeArgs := []string{} // Assuming coin type is fixed in the burn function

	txBlockResponse, err := s.suiClient.MoveCall(
		burnerAddress, // The owner of the tokens to be burned
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		burnerGasObjectID, // Corrected: Gas object owned by `burnerAddress`
		gasBudget,
	)
	if err != nil {
		utils.LogErrorf("EconomySuiService: Error preparing burn_game_tokens transaction for tokens %v from %s: %v", tokenObjectIDs, burnerAddress, err)
		return models.TxnMetaData{}, fmt.Errorf("MoveCall failed for BurnGameTokens (tokens: %v): %w", tokenObjectIDs, err)
	}
	utils.LogInfof("EconomySuiService: Burn game tokens transaction prepared for tokens %v from %s. TxBytes: %s",
		tokenObjectIDs, burnerAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}
