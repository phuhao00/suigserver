package sui

import (
	"fmt"
	"log"
	"strconv"

	"github.com/tidwall/gjson"
)

// EconomySuiService interacts with the Economic System contracts on the Sui blockchain.
type EconomySuiService struct {
	suiClient     *Client
	packageID     string // ID of the package containing the economy/token module
	moduleName    string // Name of the Move module, e.g., "game_coin"
	senderAddress string // Address of the account sending transactions (e.g., a treasury or admin account for some ops)
	gasObjectID   string // Object ID of the gas coin for transactions
}

// NewEconomySuiService creates a new EconomySuiService.
func NewEconomySuiService(client *Client, packageID, moduleName, senderAddress, gasObjectID string) *EconomySuiService {
	log.Println("Initializing Economy Sui Service...")
	if client == nil {
		log.Panic("EconomySuiService: Sui client cannot be nil")
	}
	// Some parameters might be optional depending on methods used.
	// For now, assume packageID and moduleName are core for interacting with a specific game token.
	if packageID == "" || moduleName == "" {
		log.Panic("EconomySuiService: packageID and moduleName must be provided.")
	}
	return &EconomySuiService{
		suiClient:     client,
		packageID:     packageID,
		moduleName:    moduleName,
		senderAddress: senderAddress, // May not be needed for all read-only calls
		gasObjectID:   gasObjectID,   // May not be needed for all read-only calls
	}
}

// GetPlayerBalance retrieves a player's balance for a specific on-chain coin type.
// coinType is the fully qualified type string, e.g., "0x2::sui::SUI" or "0xYOUR_PACKAGE_ID::YOUR_MODULE::YOUR_COIN_TYPE"
func (s *EconomySuiService) GetPlayerBalance(playerAddress string, coinType string) (uint64, error) {
	log.Printf("Fetching balance for player %s, CoinType: %s", playerAddress, coinType)

	// TODO: Implement actual Sui call. This uses `suix_getBalance`.
	// The `s.suiClient.GetBalance` method already exists in client.go.
	result, err := s.suiClient.GetBalance(playerAddress, coinType)
	if err != nil {
		log.Printf("Error fetching balance for %s (CoinType: %s): %v", playerAddress, coinType, err)
		return 0, err
	}

	// Example structure of suix_getBalance response:
	// { "coinType": "0x2::sui::SUI", "coinObjectCount": 1, "totalBalance": "1000000000",
	//   "lockedBalance": { "epochId": "0", "number": "0" } }
	totalBalanceStr := result.Get("totalBalance").String()
	balance, err := strconv.ParseUint(totalBalanceStr, 10, 64)
	if err != nil {
		log.Printf("Error parsing balance string '%s' for %s: %v", totalBalanceStr, playerAddress, err)
		return 0, fmt.Errorf("could not parse balance from Sui response: %w", err)
	}

	log.Printf("Balance for player %s (CoinType: %s): %d", playerAddress, coinType, balance)
	return balance, nil
}

// TransferTokens prepares a transaction to transfer on-chain tokens (Coins) from sender to recipient.
// coinObjectIDs are the specific coin objects owned by `fromAddress` to be used for payment.
// This requires the `fromAddress` to be the `senderAddress` used in `CallMoveFunction` or for the tx to be signed by `fromAddress`.
func (s *EconomySuiService) TransferTokens(
	fromAddress string, // Sender of the tokens (must sign the transaction)
	coinObjectIDs []string, // Specific coin objects to be spent. Their total value must cover 'amount'.
	amount uint64,
	toAddress string,
	coinType string, // Fully qualified type of the coin, e.g., "0xPACKAGE::MODULE::COIN_TYPE"
	gasBudget uint64,
) (gjson.Result, error) {
	functionName := "transfer_coins" // Placeholder: This usually involves PTB with `SplitCoins` and `TransferObjects`
	log.Printf("Preparing to transfer %d of %s from %s to %s. Coins: %v", amount, coinType, fromAddress, toAddress, coinObjectIDs)

	// TODO: This is a simplified placeholder. Actual coin transfer involves:
	// 1. Identifying one or more Coin<T> objects owned by `fromAddress` that sum up to at least `amount`.
	// 2. If a single coin object has > amount, it needs to be split. If multiple coin objects are needed, they are merged/used.
	// 3. The resulting Coin<T> of `amount` is then transferred to `toAddress`.
	// This is typically done using a Programmable Transaction Block (PTB).
	// `unsafe_moveCall` is not ideal for direct coin transfers.
	// A helper function in the Move contract `moduleName` might facilitate this, e.g., a `transfer` function.
	// Or, the server constructs a PTB and uses `sui_executeTransactionBlock`.

	// Placeholder using a hypothetical Move function `transfer_game_coin`
	// This assumes your `s.moduleName` has such a function.
	// module: s.moduleName, function: "transfer_game_coin"
	// args: [recipient_address_str, amount_u64, coin_object_id_to_spend_from_str]
	// This is highly dependent on your contract design.
	if len(coinObjectIDs) == 0 {
		return gjson.Result{}, fmt.Errorf("at least one coinObjectID must be provided for transfer")
	}

	callArgs := []interface{}{
		toAddress,                    // recipient
		strconv.FormatUint(amount, 10), // amount
		coinObjectIDs,                // This might be a single coin or a vector of coins depending on contract
	}
	typeArgs := []string{coinType} // The type of coin being transferred, if function is generic over T for Coin<T>

	// The `fromAddress` must be the signer of the transaction.
	// The `s.senderAddress` configured for EconomySuiService might be different (e.g., a service account).
	// For player-to-player transfers, the transaction must be signed by `fromAddress`.
	// The `gasObjectID` should also belong to `fromAddress`.
	txResult, err := s.suiClient.CallMoveFunction(
		fromAddress, // Actual sender of tokens
		s.packageID, // Package of the coin/transfer logic
		s.moduleName,
		functionName, // This hypothetical function needs to exist
		typeArgs,
		callArgs,
		s.gasObjectID, // Gas object owned by `fromAddress`
		gasBudget,
	)

	if err != nil {
		log.Printf("Error preparing token transfer from %s to %s: %v", fromAddress, toAddress, err)
		return txResult, err
	}
	log.Printf("Token transfer transaction prepared from %s to %s. Digest (from unsafe_moveCall): %s",
		fromAddress, toAddress, txResult.Get("digest").String())
	return txResult, nil
}

// MintGameTokens prepares a transaction to mint new game tokens.
// This typically requires the sender to have administrative/minter capabilities defined in the Move contract.
func (s *EconomySuiService) MintGameTokens(recipientAddress string, amount uint64, gasBudget uint64) (gjson.Result, error) {
	functionName := "mint_game_tokens" // Placeholder for your Move mint function
	log.Printf("Preparing to mint %d game tokens to %s. Admin sender: %s", amount, recipientAddress, s.senderAddress)

	callArgs := []interface{}{
		recipientAddress,
		strconv.FormatUint(amount, 10),
	}
	// Assuming the game token type is implicitly known by the `mint_game_tokens` function in `s.packageID::s.moduleName`
	typeArgs := []string{}

	txResult, err := s.suiClient.CallMoveFunction(
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
		return txResult, err
	}
	log.Printf("Mint game tokens transaction prepared for %s. Digest (from unsafe_moveCall): %s",
		recipientAddress, txResult.Get("digest").String())
	return txResult, nil
}

// BurnGameTokens prepares a transaction to burn game tokens.
// Requires the `burnerAddress` to own the `tokenObjectIDs` and sign the transaction.
func (s *EconomySuiService) BurnGameTokens(burnerAddress string, tokenObjectIDs []string, gasBudget uint64) (gjson.Result, error) {
	functionName := "burn_game_tokens" // Placeholder for your Move burn function
	log.Printf("Preparing to burn game tokens (IDs: %v) from %s.", tokenObjectIDs, burnerAddress)

	if len(tokenObjectIDs) == 0 {
		return gjson.Result{}, fmt.Errorf("at least one tokenObjectID must be provided to burn")
	}

	callArgs := []interface{}{
		tokenObjectIDs, // This would likely be a vector of Coin objects or their IDs
	}
	// Assuming the game token type is implicitly known by the `burn_game_tokens` function
	typeArgs := []string{}

	txResult, err := s.suiClient.CallMoveFunction(
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
		return txResult, err
	}
	log.Printf("Burn game tokens transaction prepared for tokens %v. Digest (from unsafe_moveCall): %s",
		tokenObjectIDs, txResult.Get("digest").String())
	return txResult, nil
}
