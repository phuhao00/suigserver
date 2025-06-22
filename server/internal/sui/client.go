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
	"github.com/tidwall/gjson"
)

// SuiClient represents a client for interacting with Sui blockchain using sui-go-sdk
type SuiClient struct {
	sdkClient sui.ISuiAPI
	nodeURL   string
}

// NewSuiClient creates a new Sui client using sui-go-sdk
func NewSuiClient(nodeURL string) *SuiClient {
	if nodeURL == "" {
		nodeURL = sui.TestnetRPC // Default to testnet if not specified
	}
	// Initialize the sui-go-sdk client
	// The sdk does not require explicit timeout setting in the same way as resty here.
	// It uses default http client settings which can be customized if needed by creating a custom http.Client.
	opts := sui.DefaultClientOptions()
	opts.RpcUrl = nodeURL

	cli := sui.NewSuiClient(opts)

	return &SuiClient{
		sdkClient: cli,
		nodeURL:   nodeURL,
	}
}

// GetObject retrieves an object from Sui
func (c *SuiClient) GetObject(objectID string) (models.SuiObjectResponse, error) {
	return c.sdkClient.SuiGetObject(context.Background(), models.SuiGetObjectRequest{
		ObjectId: objectID,
		Options: &models.SuiObjectDataOptions{
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
func (c *SuiClient) GetOwnedObjects(address string, objectType *string) (models.SuiGetOwnedObjectsResponse, error) {
	var filter interface{}
	if objectType != nil {
		filter = map[string]interface{}{"StructType": *objectType}
	}

	return c.sdkClient.SuiGetOwnedObjects(context.Background(), models.SuiGetOwnedObjectsRequest{
		Address: address,
		Query: &models.SuiObjectResponseQuery{
			Filter: filter,
			Options: &models.SuiObjectDataOptions{
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
// This function will now return a models.TransactionBlockResponse which contains txBytes.
// The actual execution requires signing and then calling ExecuteTransactionBlock.
func (c *SuiClient) MoveCall(sender, packageID, module, function string, typeArguments []string, arguments []interface{}, gas string, gasBudget uint64) (models.TransactionBlockResponse, error) {
	gasBudgetStr := strconv.FormatUint(gasBudget, 10)
	// gasPriceStr := strconv.FormatUint(1000, 10) // Example gas price

	return c.sdkClient.SuiMoveCall(context.Background(), models.SuiMoveCallRequest{
		Signer:          sender,
		PackageObjectId: packageID,
		Module:          module,
		Function:        function,
		TypeArguments:   typeArguments,
		Arguments:       arguments,
		Gas:             &gas,
		GasBudget:       gasBudgetStr,
		// GasPrice:      gasPriceStr, // GasPrice is often fetched dynamically or set globally
	})
}

// ExecuteTransactionBlock executes a transaction block
func (c *SuiClient) ExecuteTransactionBlock(txBytes string, signatures []string) (models.SuiExecuteTransactionBlockResponse, error) {
	return c.sdkClient.SuiExecuteTransactionBlock(context.Background(), models.SuiExecuteTransactionBlockRequest{
		TxBytes:    txBytes,
		Signatures: signatures,
		Options: &models.SuiTransactionBlockResponseOptions{
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
func (c *SuiClient) QueryEvents(query models.EventFilter, cursor *string, limit *uint64, descendingOrder bool) (models.SuiQueryEventsResponse, error) {
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
			eventSeqUint, err := strconv.ParseUint(parts[1], 10, 64)
			if err == nil {
				actualCursor = &models.EventId{TxDigest: parts[0], EventSeq: eventSeqUint}
			} else {
				utils.LogWarnf("SUI Client: Could not parse EventSeq from cursor '%s': %v", *cursor, err)
				// Decide if this should be a hard error or proceed with nil cursor
			}
		} else {
			utils.LogWarnf("SUI Client: Could not parse cursor string '%s' into TxDigest and EventSeq", *cursor)
			// Decide if this should be a hard error or proceed with nil cursor
		}
	}

	return c.sdkClient.SuiQueryEvents(context.Background(), models.SuiQueryEventsRequest{
		Query:           query,
		Cursor:          actualCursor,
		Limit:           actualLimit,
		DescendingOrder: descendingOrder,
	})
}

// GetCoins retrieves coins owned by an address
func (c *SuiClient) GetCoins(address, coinType string) (models.SuiGetCoinsResponse, error) {
	return c.sdkClient.SuiGetCoins(context.Background(), models.SuiGetCoinsRequest{
		Owner:    address,
		CoinType: &coinType,
	})
}

// GetBalance gets the balance for a specific coin type
func (c *SuiClient) GetBalance(address, coinType string) (models.SuiGetBalanceResponse, error) {
	return c.sdkClient.SuiGetBalance(context.Background(), models.SuiGetBalanceRequest{
		Owner:    address,
		CoinType: &coinType,
	})
}

// Legacy Client struct for backward compatibility.
// It now embeds the new SuiClient which uses sui-go-sdk.
type Client struct {
	*SuiClient
}

// NewClient creates a new legacy client wrapper.
// It initializes SuiClient with a default nodeURL if not specified.
func NewClient() (*Client, error) {
	// Consider making nodeURL configurable, e.g., from environment variables or config file
	suiClient := NewSuiClient(sui.TestnetRPC) // Or your preferred default
	return &Client{
		SuiClient: suiClient,
	}, nil
}

// CallMoveFunction is a simplified wrapper for preparing a Move call transaction.
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
	gasBudget uint64,   // Budget for gas in MIST
) (models.TransactionBlockResponse, error) { // Return type changed to models.TransactionBlockResponse
	utils.LogInfof("SUI Client: Preparing Move call: Package=%s, Module=%s, Function=%s, Sender=%s", packageID, module, functionName, senderAddress)
	utils.LogDebugf("SUI Client: Type Args: %v", typeArgs)
	utils.LogDebugf("SUI Client: Call Args: %v", callArgs)
	utils.LogInfof("SUI Client: Gas Object: %s, Gas Budget: %d", gasObjectID, gasBudget)

	if senderAddress == "" || packageID == "" || gasObjectID == "" {
		errMsg := "senderAddress, packageID, and gasObjectID must be provided"
		utils.LogError(errMsg)
		return models.TransactionBlockResponse{}, fmt.Errorf(errMsg)
	}
	if gasBudget == 0 {
		gasBudget = 10000000 // Default gas budget
		utils.LogWarnf("SUI Client: Using default gas budget: %d", gasBudget)
	}

	// Use the embedded SuiClient's MoveCall method
	// The gas parameter to c.SuiClient.MoveCall should be the gas object ID string.
	txBlockResponse, err := c.SuiClient.MoveCall(senderAddress, packageID, module, functionName, typeArgs, callArgs, gasObjectID, gasBudget)
	if err != nil {
		utils.LogErrorf("SUI Client: Error preparing Move call %s::%s::%s: %v", packageID, module, functionName, err)
		return models.TransactionBlockResponse{}, fmt.Errorf("MoveCall preparation failed for %s::%s: %w", module, functionName, err)
	}

	utils.LogInfof("SUI Client: Move call %s::%s::%s prepared successfully. TxBytes: %s", packageID, module, functionName, txBlockResponse.TxBytes)
	// The response now directly gives TxBytes. Further steps (signing, execution) are needed.
	// This function's responsibility ends with providing the transaction bytes.
	return txBlockResponse, nil
}

// Placeholder for gjson.Result if some legacy code absolutely needs it.
// Ideally, refactor away from gjson.Result towards specific SDK types.
func adaptToGJSON(data interface{}) (gjson.Result, error) {
	// This is a placeholder. Proper adaptation would involve marshalling `data` to JSON
	// and then parsing with gjson, or directly constructing gjson.Result if feasible.
	// For now, it suggests that this adaptation might be complex or lossy.
	utils.LogWarn("SUI Client: adaptToGJSON is a placeholder and may not function correctly.")
	return gjson.Parse(""), fmt.Errorf("adaptToGJSON not fully implemented")
}

// SignTransactionBytesWithServerKey conceptually signs transaction bytes using the server's private key.
// This is a placeholder to illustrate where server-side signing would occur.
// In a real implementation, ensure secure handling of the private key.
func SignTransactionBytesWithServerKey(txBytes string, serverPrivateKeyHex string) (string /*signature*/, error) {
	if serverPrivateKeyHex == "" || serverPrivateKeyHex == "YOUR_SUI_PRIVATE_KEY_HEX_HERE" {
		errMsg := "server private key is not configured or is a placeholder"
		utils.LogWarnf("SUI Client (SignTransactionBytesWithServerKey): %s. Cannot sign.", errMsg)
		return "", fmt.Errorf(errMsg)
	}

	utils.LogInfof("SUI Client: Conceptually signing TxBytes: %s", txBytes)
	utils.LogInfo("SUI Client: Conceptual step: Decoding TxBytes (typically base64).")
	// decodedTxBytes, err := base64.StdEncoding.DecodeString(txBytes)
	// if err != nil { utils.LogErrorf("Failed to decode txBytes: %v", err); return "", err }

	utils.LogInfof("SUI Client: Conceptual step: Creating keypair from server's private key hex: %s", serverPrivateKeyHex[:min(10, len(serverPrivateKeyHex))]+"...") // Log only a prefix
	// kp, err := keytool.NewEd25519KeyPairWithHex(serverPrivateKeyHex)
	// if err != nil { utils.LogErrorf("Failed to create keypair from hex: %v", err); return "", err }

	utils.LogInfo("SUI Client: Conceptual step: Signing decodedTxBytes with keypair.SignSuiMessage(...).")
	// signature := kp.SignSuiMessage(decodedTxBytes)
	// signedTxBase64 := base64.StdEncoding.EncodeToString(signature)

	simulatedSignature := "SIMULATED_SERVER_SIGNATURE_FOR_" + txBytes[:min(10, len(txBytes))]+"..."
	utils.LogInfof("SUI Client: Conceptual signing complete. Simulated Signature: %s", simulatedSignature)

	return simulatedSignature, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
