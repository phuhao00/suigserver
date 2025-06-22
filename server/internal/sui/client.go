package sui

import (
	"context"
	"fmt"

	// "log" // Replaced by utils.LogX
	"strconv"
	"strings"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	"github.com/phuhao00/suigserver/server/internal/utils" // Logger
	// "github.com/tidwall/gjson" // No longer needed if adaptToGJSON is removed
)

// SuiClient represents a client for interacting with Sui blockchain using sui-go-sdk
type SuiClient struct {
	sdkClient sui.ISuiAPI
	nodeURL   string
}

// NewSuiClient creates a new Sui client using sui-go-sdk
func NewSuiClient(nodeURL string) *SuiClient {
	if nodeURL == "" {
		nodeURL = "https://fullnode.testnet.sui.io:443" // Default to testnet if not specified
	}
	// Initialize the sui-go-sdk client
	// The sdk does not require explicit timeout setting in the same way as resty here.
	// It uses default http client settings which can be customized if needed by creating a custom http.Client.
	cli := sui.NewSuiClient(nodeURL)

	return &SuiClient{
		sdkClient: cli,
		nodeURL:   nodeURL,
	}
}

// GetObject retrieves an object from Sui
func (c *SuiClient) GetObject(objectID string) (models.SuiObjectResponse, error) {
	return c.sdkClient.SuiGetObject(context.Background(), models.SuiGetObjectRequest{
		ObjectId: objectID,
		Options: models.SuiObjectDataOptions{
			ShowType:                true,
			ShowOwner:               true,
			ShowPreviousTransaction: true,
			ShowDisplay:             false,
			ShowContent:             true,
			ShowBcs:                 false,
			ShowStorageRebate:       true,
		},
	})
}

// GetOwnedObjects retrieves objects owned by an address
func (c *SuiClient) GetOwnedObjects(address string, objectType *string) (models.PaginatedObjectsResponse, error) {
	var filter interface{}
	if objectType != nil {
		filter = map[string]interface{}{"StructType": *objectType}
	}

	return c.sdkClient.SuiXGetOwnedObjects(context.Background(), models.SuiXGetOwnedObjectsRequest{
		Address: address,
		Query: models.SuiObjectResponseQuery{
			Filter: filter,
			Options: models.SuiObjectDataOptions{
				ShowType:                true,
				ShowOwner:               true,
				ShowPreviousTransaction: true,
				ShowDisplay:             false,
				ShowContent:             true,
				ShowBcs:                 false,
				ShowStorageRebate:       true,
			},
		},
	})
}

// MoveCall prepares a transaction block for a Move function call.
// Note: sui-go-sdk's MoveCall is part of building a transaction block.
// This function will now return a models.TxnMetaData which contains transaction metadata.
// The actual execution requires signing and then calling ExecuteTransactionBlock.
func (c *SuiClient) MoveCall(sender, packageID, module, function string, typeArguments []string, arguments []interface{}, gas string, gasBudget uint64) (models.TxnMetaData, error) {
	gasBudgetStr := strconv.FormatUint(gasBudget, 10)
	// gasPriceStr := strconv.FormatUint(1000, 10) // Example gas price

	// Convert typeArguments to []interface{}
	typeArgs := make([]interface{}, len(typeArguments))
	for i, arg := range typeArguments {
		typeArgs[i] = arg
	}

	return c.sdkClient.MoveCall(context.Background(), models.MoveCallRequest{
		Signer:          sender,
		PackageObjectId: packageID,
		Module:          module,
		Function:        function,
		TypeArguments:   typeArgs,
		Arguments:       arguments,
		Gas:             &gas,
		GasBudget:       gasBudgetStr,
		// GasPrice:      gasPriceStr, // GasPrice is often fetched dynamically or set globally
	})
}

// ExecuteTransactionBlock executes a transaction block
func (c *SuiClient) ExecuteTransactionBlock(txBytes string, signatures []string) (models.SuiTransactionBlockResponse, error) {
	return c.sdkClient.SuiExecuteTransactionBlock(context.Background(), models.SuiExecuteTransactionBlockRequest{
		TxBytes:   txBytes,
		Signature: signatures,
		Options: models.SuiTransactionBlockOptions{
			ShowInput:          true,
			ShowRawInput:       false,
			ShowEffects:        true,
			ShowEvents:         true,
			ShowObjectChanges:  true,
			ShowBalanceChanges: true,
		},
		RequestType: "WaitForLocalExecution",
	})
}

// QueryEvents queries events from Sui
func (c *SuiClient) QueryEvents(query models.SuiEventFilter, cursor *string, limit *uint64, descendingOrder bool) (models.PaginatedEventsResponse, error) {
	var actualLimit uint64 = 50 // Default limit
	if limit != nil {
		actualLimit = *limit
	}

	var actualCursor *models.EventId = nil
	if cursor != nil {
		// Assuming cursor is in "txDigest:eventSeq" format
		// This needs proper parsing if sui-go-sdk expects models.EventId
		// For simplicity, this example assumes the SDK handles string cursor directly or requires a different format
		// If models.EventId is required, you'll need to parse *cursor into its components.
		// For now, we'll pass nil if direct string cursor isn't supported, or adjust if the SDK has a way.
		// Example: parts := strings.Split(*cursor, ":"); if len(parts) == 2 { actualCursor = &models.EventId{TxDigest: parts[0], EventSeq: parts[1]}}
		parts := strings.SplitN(*cursor, ":", 2)
		if len(parts) == 2 {
			eventSeqStr := parts[1]
			actualCursor = &models.EventId{TxDigest: parts[0], EventSeq: eventSeqStr}
		} else {
			// Log the error and proceed with actualCursor as nil.
			utils.LogErrorf("SUI Client: Could not parse cursor string '%s' into TxDigest and EventSeq. Proceeding without cursor.", *cursor)
			actualCursor = nil // Explicitly set to nil on error
		}
	}

	return c.sdkClient.SuiXQueryEvents(context.Background(), models.SuiXQueryEventsRequest{
		SuiEventFilter:  query,
		Cursor:          actualCursor,
		Limit:           actualLimit,
		DescendingOrder: descendingOrder,
	})
}

// GetCoins retrieves coins owned by an address
func (c *SuiClient) GetCoins(address, coinType string) (models.PaginatedCoinsResponse, error) {
	return c.sdkClient.SuiXGetCoins(context.Background(), models.SuiXGetCoinsRequest{
		Owner:    address,
		CoinType: coinType,
	})
}

// GetBalance gets the balance for a specific coin type
func (c *SuiClient) GetBalance(address, coinType string) (models.CoinBalanceResponse, error) {
	return c.sdkClient.SuiXGetBalance(context.Background(), models.SuiXGetBalanceRequest{
		Owner:    address,
		CoinType: coinType,
	})
}

// Legacy Client struct for backward compatibility.
// It now embeds the new SuiClient which uses sui-go-sdk.
type Client struct {
	*SuiClient // Embedded SuiClient for direct access to its methods
}

// NewClient creates a new legacy client wrapper.
// DEPRECATED: Use NewSuiClient directly. This legacy client will be removed in a future version.
// It initializes SuiClient with a default nodeURL if not specified.
func NewClient() (*Client, error) {
	utils.LogWarn("SUI Client: NewClient is deprecated. Use NewSuiClient directly.")
	// Consider making nodeURL configurable, e.g., from environment variables or config file
	suiClient := NewSuiClient("https://fullnode.testnet.sui.io:443") // Or your preferred default
	return &Client{
		SuiClient: suiClient,
	}, nil
}

// CallMoveFunction is a simplified wrapper for preparing a Move call transaction.
// DEPRECATED: Use SuiClient.MoveCall directly. This method will be removed.
// It now uses the new SuiClient's MoveCall method.
// Note: This function still only prepares the transaction (returns txBytes).
// It does not sign or execute it.
func (c *Client) CallMoveFunction(
	senderAddress string,
	packageID string,
	module string,
	functionName string,
	typeArgs []string,
	callArgs []interface{},
	gasObjectID string, // Gas object ID to be used for payment
	gasBudget uint64, // Budget for gas in MIST
) (models.TxnMetaData, error) { // Return type changed to models.TxnMetaData
	utils.LogWarnf("SUI Client: CallMoveFunction is deprecated. Use SuiClient.MoveCall directly for preparing transactions.")
	utils.LogInfof("SUI Client (Legacy CallMoveFunction): Preparing Move call: Package=%s, Module=%s, Function=%s, Sender=%s", packageID, module, functionName, senderAddress)
	utils.LogDebugf("SUI Client (Legacy CallMoveFunction): Type Args: %v", typeArgs)
	utils.LogDebugf("SUI Client (Legacy CallMoveFunction): Call Args: %v", callArgs)
	utils.LogInfof("SUI Client (Legacy CallMoveFunction): Gas Object: %s, Gas Budget: %d", gasObjectID, gasBudget)

	if senderAddress == "" || packageID == "" || gasObjectID == "" {
		errMsg := "senderAddress, packageID, and gasObjectID must be provided"
		utils.LogError(errMsg)
		return models.TxnMetaData{}, fmt.Errorf(errMsg)
	}
	if gasBudget == 0 {
		gasBudget = 10000000 // Default gas budget
		utils.LogWarnf("SUI Client (Legacy CallMoveFunction): Using default gas budget: %d", gasBudget)
	}

	// Use the embedded SuiClient's MoveCall method
	// The gas parameter to c.SuiClient.MoveCall should be the gas object ID string.
	txBlockResponse, err := c.SuiClient.MoveCall(senderAddress, packageID, module, functionName, typeArgs, callArgs, gasObjectID, gasBudget)
	if err != nil {
		utils.LogErrorf("SUI Client (Legacy CallMoveFunction): Error preparing Move call %s::%s::%s: %v", packageID, module, functionName, err)
		return models.TxnMetaData{}, fmt.Errorf("MoveCall preparation failed for %s::%s: %w", module, functionName, err)
	}

	utils.LogInfof("SUI Client (Legacy CallMoveFunction): Move call %s::%s::%s prepared successfully. TxBytes: %s", packageID, module, functionName, txBlockResponse.TxBytes)
	// The response now directly gives TxBytes. Further steps (signing, execution) are needed.
	// This function's responsibility ends with providing the transaction bytes.
	return txBlockResponse, nil
}

// SignTransactionBytesWithServerKey conceptually signs transaction bytes using the server's private key.
// This is a placeholder to illustrate where server-side signing would occur.
// In a real implementation, ensure SECURE HANDLING of the private key.
// The private key should ideally be stored in a secure vault or KMS.
// For local development/testing, it might be an environment variable, but NEVER hardcoded for production.
// Returns the base64 encoded signature.
func SignTransactionBytesWithServerKey(txBytes string, serverPrivateKeyHex string) (string /*base64Signature*/, error) {
	if serverPrivateKeyHex == "" || serverPrivateKeyHex == "YOUR_SUI_PRIVATE_KEY_HEX_HERE" || len(strings.TrimSpace(serverPrivateKeyHex)) == 0 {
		errMsg := "SUI Client (SignTransactionBytesWithServerKey): server private key is not configured, is a placeholder, or is empty. CANNOT SIGN."
		utils.LogError(errMsg) // Log as error because this is a critical failure if called.
		return "", fmt.Errorf(errMsg)
	}

	// This is a conceptual, simplified signing process.
	// Real signing involves decoding txBytes (which are base64), creating a keypair, and signing.
	// The sui-go-sdk's keytool can be used for this.

	utils.LogInfo("SUI Client: Attempting conceptual server-side signing.")
	utils.LogDebugf("SUI Client: TxBytes to sign (first 64 chars): %s...", txBytes[:min(64, len(txBytes))])

	// For now, return a placeholder signature as we don't have keytool access in this simplified version
	utils.LogWarn("SUI Client: Signing functionality not yet implemented. Returning placeholder signature.")

	// The actual signature scheme and message to sign are more complex for real transactions.
	// For `ExecuteTransactionBlock`, the `signatures` field is `Vec<Base64>` of `[flag || signature || pubkey]`

	// For this placeholder, we will return a FAKE but correctly formatted (conceptually) signature string.
	// This will NOT be a valid signature on the actual Sui network.

	// Constructing a simulated signature string that mimics the expected format for the API
	// [signature_scheme_flag || signature || public_key]
	// For Ed25519, flag is 0. For Secp256k1, flag is 1.
	// This is highly simplified and not cryptographically sound for real use.

	// Remove the trimmed key and keypair code that was causing compilation errors
	// trimmedKey := strings.TrimPrefix(serverPrivateKeyHex, "0x")

	// TODO: Implement proper key management with sui-go-sdk
	// For now, return a placeholder signature
	simulatedBase64Signature := "SIMULATED_SERVER_SIGNATURE_B64_FOR_" + txBytes[:min(10, len(txBytes))] + "..." + "_PLACEHOLDER"

	utils.LogInfof("SUI Client: Conceptual signing complete. Simulated Base64 Signature: %s", simulatedBase64Signature)
	utils.LogWarn("SUI Client: The generated signature is for DEMONSTRATION PURPOSES ONLY and is NOT cryptographically valid.")

	return simulatedBase64Signature, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
