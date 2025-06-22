package sui

import (
	"fmt"
	"log"
	"strconv"

	"github.com/block-vision/sui-go-sdk/models"
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
func NewEconomySuiService(client *SuiClient, packageID, moduleName, senderAddress, gasObjectID string) *EconomySuiService {
	log.Println("Initializing Economy Sui Service...")
	if client == nil {
		log.Panic("EconomySuiService: Sui client cannot be nil")
	}
	if packageID == "" || moduleName == "" {
		log.Panic("EconomySuiService: packageID and moduleName must be provided.")
	}
	return &EconomySuiService{
		suiClient:     client,
		packageID:     packageID,
		moduleName:    moduleName,
		senderAddress: senderAddress,
		gasObjectID:   gasObjectID,
	}
}

// GetPlayerBalance retrieves a player's balance for a specific on-chain coin type.
func (s *EconomySuiService) GetPlayerBalance(playerAddress string, coinType string) (uint64, error) {
	log.Printf("Fetching balance for player %s, CoinType: %s", playerAddress, coinType)

	resp, err := s.suiClient.GetBalance(playerAddress, coinType)
	if err != nil {
		log.Printf("Error fetching balance for %s (CoinType: %s): %v", playerAddress, coinType, err)
		return 0, err
	}

	balance, err := strconv.ParseUint(resp.TotalBalance, 10, 64)
	if err != nil {
		log.Printf("Error parsing balance string '%s' for %s: %v", resp.TotalBalance, playerAddress, err)
		return 0, fmt.Errorf("could not parse balance from Sui response: %w", err)
	}

	log.Printf("Balance for player %s (CoinType: %s): %d", playerAddress, coinType, balance)
	return balance, nil
}

// TransferTokens prepares a transaction to transfer on-chain tokens (Coins).
// This function now returns TransactionBlockResponse, which contains the txBytes.
// The caller is responsible for signing and executing the transaction.
// Note: The sui-go-sdk's PaySui/PayAllSui methods are more idiomatic for SUI transfers.
// For custom game tokens, a MoveCall to a specific transfer function in your contract is common.
// This example assumes a generic "transfer_coins" function in your module.
func (s *EconomySuiService) TransferTokens(
	fromAddress string,     // Sender of the tokens (must sign the transaction)
	coinObjectIDs []string, // Specific coin objects to be spent.
	amount uint64,
	toAddress string,
	coinType string, // Fully qualified type of the coin
	gasBudget uint64,
) (models.TransactionBlockResponse, error) {
	// This function name should match a function in your Move contract.
	// For SUI, it's better to use PTB commands like SplitCoins and TransferObjects.
	// For a custom Coin<T>, you might have a `transfer(coin: Coin<T>, amount: u64, recipient: address)`
	// or `split_and_transfer(coins: vector<Coin<T>>, amount: u64, recipient: address)`
	// The example below assumes a function that takes coin object IDs.
	functionName := "transfer_game_coin" // Replace with your actual contract function
	log.Printf("Preparing to transfer %d of %s from %s to %s using coins %v.", amount, coinType, fromAddress, toAddress, coinObjectIDs)

	if len(coinObjectIDs) == 0 {
		return models.TransactionBlockResponse{}, fmt.Errorf("at least one coinObjectID must be provided for transfer")
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
		fromAddress,   // Signer and owner of coins/gas
		s.packageID,   // Package of the coin/transfer logic
		s.moduleName,  // Module containing the transfer function
		functionName,  // The actual function in your Move contract
		typeArgs,      // Type arguments, if any
		callArgs,      // Arguments for the Move function
		s.gasObjectID, // Gas object owned by `fromAddress`
		gasBudget,
	)

	if err != nil {
		log.Printf("Error preparing token transfer from %s to %s: %v", fromAddress, toAddress, err)
		return models.TransactionBlockResponse{}, err
	}
	log.Printf("Token transfer transaction prepared from %s to %s. TxBytes: %s",
		fromAddress, toAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// MintGameTokens prepares a transaction to mint new game tokens.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *EconomySuiService) MintGameTokens(recipientAddress string, amount uint64, gasBudget uint64) (models.TransactionBlockResponse, error) {
	functionName := "mint_game_tokens" // Placeholder for your Move mint function
	log.Printf("Preparing to mint %d game tokens to %s. Admin sender: %s", amount, recipientAddress, s.senderAddress)

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
		log.Printf("Error preparing mint_game_tokens transaction for %s: %v", recipientAddress, err)
		return models.TransactionBlockResponse{}, err
	}
	log.Printf("Mint game tokens transaction prepared for %s. TxBytes: %s",
		recipientAddress, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// BurnGameTokens prepares a transaction to burn game tokens.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *EconomySuiService) BurnGameTokens(burnerAddress string, tokenObjectIDs []string, gasBudget uint64) (models.TransactionBlockResponse, error) {
	functionName := "burn_game_tokens" // Placeholder for your Move burn function
	log.Printf("Preparing to burn game tokens (IDs: %v) from %s.", tokenObjectIDs, burnerAddress)

	if len(tokenObjectIDs) == 0 {
		return models.TransactionBlockResponse{}, fmt.Errorf("at least one tokenObjectID must be provided to burn")
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
		s.gasObjectID, // Gas object owned by `burnerAddress`
		gasBudget,
	)
	if err != nil {
		log.Printf("Error preparing burn_game_tokens transaction for tokens %v: %v", tokenObjectIDs, err)
		return models.TransactionBlockResponse{}, err
	}
	log.Printf("Burn game tokens transaction prepared for tokens %v. TxBytes: %s",
		tokenObjectIDs, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}
