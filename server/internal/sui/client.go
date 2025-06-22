package sui

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

// SuiClient represents a client for interacting with Sui blockchain
type SuiClient struct {
	client  *resty.Client
	nodeURL string
}

// TransactionRequest represents a Sui transaction request
type TransactionRequest struct {
	Sender           string                 `json:"sender"`
	PackageObjectID  string                 `json:"packageObjectId"`
	Module           string                 `json:"module"`
	Function         string                 `json:"function"`
	TypeArguments    []string               `json:"typeArguments,omitempty"`
	Arguments        []interface{}          `json:"arguments"`
	Gas              string                 `json:"gas,omitempty"`
	GasBudget        string                 `json:"gasBudget"`
	GasPrice         string                 `json:"gasPrice,omitempty"`
}

// TransactionResponse represents a Sui transaction response
type TransactionResponse struct {
	TxBytes    string `json:"txBytes"`
	Gas        Gas    `json:"gas"`
	InputObjects []InputObject `json:"inputObjects"`
}

type Gas struct {
	ObjectID string `json:"objectId"`
	Version  int    `json:"version"`
	Digest   string `json:"digest"`
}

type InputObject struct {
	ObjectType string `json:"objectType"`
	ObjectID   string `json:"objectId"`
	Version    int    `json:"version"`
	Digest     string `json:"digest"`
}

// ExecuteTransactionResponse represents the response from executing a transaction
type ExecuteTransactionResponse struct {
	Digest          string `json:"digest"`
	Transaction     Transaction `json:"transaction"`
	Effects         Effects `json:"effects"`
	Events          []Event `json:"events"`
	ObjectChanges   []ObjectChange `json:"objectChanges"`
	BalanceChanges  []BalanceChange `json:"balanceChanges"`
	TimestampMs     string `json:"timestampMs"`
	Checkpoint      string `json:"checkpoint"`
}

type Transaction struct {
	Data   TransactionData `json:"data"`
	TxSignatures []string `json:"txSignatures"`
}

type TransactionData struct {
	MessageVersion string `json:"messageVersion"`
	Transaction    TransactionDetails `json:"transaction"`
	Sender         string `json:"sender"`
	GasData        GasData `json:"gasData"`
}

type TransactionDetails struct {
	Kind    string `json:"kind"`
	Inputs  []Input `json:"inputs"`
	Transactions []TransactionCall `json:"transactions"`
}

type Input struct {
	Type      string `json:"type"`
	ValueType string `json:"valueType,omitempty"`
	Value     string `json:"value,omitempty"`
}

type TransactionCall struct {
	MoveCall *MoveCall `json:"MoveCall,omitempty"`
}

type MoveCall struct {
	Package   string        `json:"package"`
	Module    string        `json:"module"`
	Function  string        `json:"function"`
	Arguments []interface{} `json:"arguments"`
}

type GasData struct {
	Payment []PaymentObject `json:"payment"`
	Owner   string          `json:"owner"`
	Price   string          `json:"price"`
	Budget  string          `json:"budget"`
}

type PaymentObject struct {
	ObjectID string `json:"objectId"`
	Version  int    `json:"version"`
	Digest   string `json:"digest"`
}

type Effects struct {
	MessageVersion string `json:"messageVersion"`
	Status         Status `json:"status"`
	ExecutedEpoch  string `json:"executedEpoch"`
	GasUsed        GasUsed `json:"gasUsed"`
	ModifiedAtVersions []ModifiedAtVersion `json:"modifiedAtVersions"`
	TransactionDigest string `json:"transactionDigest"`
	Created           []Created `json:"created"`
	Mutated           []Mutated `json:"mutated"`
	Deleted           []Deleted `json:"deleted"`
}

type Status struct {
	Status string `json:"status"`
}

type GasUsed struct {
	ComputationCost string `json:"computationCost"`
	StorageCost     string `json:"storageCost"`
	StorageRebate   string `json:"storageRebate"`
	NonRefundableStorageFee string `json:"nonRefundableStorageFee"`
}

type ModifiedAtVersion struct {
	ObjectID       string `json:"objectId"`
	SequenceNumber string `json:"sequenceNumber"`
}

type Created struct {
	Owner     Owner  `json:"owner"`
	Reference Reference `json:"reference"`
}

type Mutated struct {
	Owner     Owner  `json:"owner"`
	Reference Reference `json:"reference"`
}

type Deleted struct {
	ObjectID string `json:"objectId"`
	Version  int    `json:"version"`
	Digest   string `json:"digest"`
}

type Owner struct {
	AddressOwner string `json:"AddressOwner,omitempty"`
	ObjectOwner  string `json:"ObjectOwner,omitempty"`
	Shared       *SharedOwner `json:"Shared,omitempty"`
}

type SharedOwner struct {
	InitialSharedVersion string `json:"initial_shared_version"`
}

type Reference struct {
	ObjectID string `json:"objectId"`
	Version  int    `json:"version"`
	Digest   string `json:"digest"`
}

type Event struct {
	ID               EventID `json:"id"`
	PackageID        string  `json:"packageId"`
	TransactionModule string `json:"transactionModule"`
	Sender           string  `json:"sender"`
	Type             string  `json:"type"`
	ParsedJSON       interface{} `json:"parsedJson"`
	Bcs              string  `json:"bcs"`
	TimestampMs      string  `json:"timestampMs"`
}

type EventID struct {
	TxDigest string `json:"txDigest"`
	EventSeq string `json:"eventSeq"`
}

type ObjectChange struct {
	Type         string `json:"type"`
	Sender       string `json:"sender"`
	Owner        Owner  `json:"owner"`
	ObjectType   string `json:"objectType"`
	ObjectID     string `json:"objectId"`
	Version      string `json:"version"`
	PreviousVersion string `json:"previousVersion,omitempty"`
	Digest       string `json:"digest"`
}

type BalanceChange struct {
	Owner    Owner  `json:"owner"`
	CoinType string `json:"coinType"`
	Amount   string `json:"amount"`
}

// NewSuiClient creates a new Sui client
func NewSuiClient(nodeURL string) *SuiClient {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("Content-Type", "application/json")
	
	if nodeURL == "" {
		nodeURL = "https://fullnode.testnet.sui.io:443"
	}
	
	return &SuiClient{
		client:  client,
		nodeURL: nodeURL,
	}
}

// makeRPCCall makes a JSON-RPC call to the Sui node
func (c *SuiClient) makeRPCCall(method string, params []interface{}) (gjson.Result, error) {
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	resp, err := c.client.R().
		SetBody(requestBody).
		Post(c.nodeURL)

	if err != nil {
		return gjson.Result{}, fmt.Errorf("failed to make RPC call: %w", err)
	}

	if resp.StatusCode() != 200 {
		return gjson.Result{}, fmt.Errorf("RPC call failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	result := gjson.Parse(resp.String())
	
	if result.Get("error").Exists() {
		return gjson.Result{}, fmt.Errorf("RPC error: %s", result.Get("error").String())
	}

	return result.Get("result"), nil
}

// GetObject retrieves an object from Sui
func (c *SuiClient) GetObject(objectID string) (gjson.Result, error) {
	params := []interface{}{
		objectID,
		map[string]interface{}{
			"showType":    true,
			"showOwner":   true,
			"showPreviousTransaction": true,
			"showDisplay": false,
			"showContent": true,
			"showBcs":     false,
			"showStorageRebate": true,
		},
	}

	return c.makeRPCCall("sui_getObject", params)
}

// GetOwnedObjects retrieves objects owned by an address
func (c *SuiClient) GetOwnedObjects(address string, objectType *string) (gjson.Result, error) {
	params := []interface{}{
		address,
		map[string]interface{}{
			"filter": map[string]interface{}{
				"StructType": objectType,
			},
			"options": map[string]interface{}{
				"showType":    true,
				"showOwner":   true,
				"showPreviousTransaction": true,
				"showDisplay": false,
				"showContent": true,
				"showBcs":     false,
				"showStorageRebate": true,
			},
		},
	}

	return c.makeRPCCall("sui_getOwnedObjects", params)
}

// MoveCall executes a Move function call
func (c *SuiClient) MoveCall(sender, packageID, module, function string, typeArguments []string, arguments []interface{}, gas string, gasBudget uint64) (gjson.Result, error) {
	params := []interface{}{
		sender,
		packageID,
		module,
		function,
		typeArguments,
		arguments,
		gas,
		strconv.FormatUint(gasBudget, 10),
		strconv.FormatUint(1000, 10), // gas price
	}

	return c.makeRPCCall("unsafe_moveCall", params)
}

// ExecuteTransactionBlock executes a transaction block
func (c *SuiClient) ExecuteTransactionBlock(txBytes string, signatures []string) (gjson.Result, error) {
	params := []interface{}{
		txBytes,
		signatures,
		map[string]interface{}{
			"showInput":         true,
			"showRawInput":      false,
			"showEffects":       true,
			"showEvents":        true,
			"showObjectChanges": true,
			"showBalanceChanges": true,
		},
		"WaitForLocalExecution",
	}

	return c.makeRPCCall("sui_executeTransactionBlock", params)
}

// QueryEvents queries events from Sui
func (c *SuiClient) QueryEvents(query map[string]interface{}, cursor *string, limit *uint64, descendingOrder bool) (gjson.Result, error) {
	params := []interface{}{
		query,
	}
	
	if cursor != nil {
		params = append(params, *cursor)
	} else {
		params = append(params, nil)
	}
	
	if limit != nil {
		params = append(params, *limit)
	} else {
		params = append(params, 50) // default limit
	}
	
	params = append(params, descendingOrder)

	return c.makeRPCCall("suix_queryEvents", params)
}

// GetCoins retrieves coins owned by an address
func (c *SuiClient) GetCoins(address, coinType string) (gjson.Result, error) {
	params := []interface{}{
		address,
		coinType,
	}

	return c.makeRPCCall("suix_getCoins", params)
}

// GetBalance gets the balance for a specific coin type
func (c *SuiClient) GetBalance(address, coinType string) (gjson.Result, error) {
	params := []interface{}{
		address,
		coinType,
	}

	return c.makeRPCCall("suix_getBalance", params)
}

// Legacy Client struct for backward compatibility
type Client struct {
	*SuiClient
}

func NewClient() (*Client, error) {
	suiClient := NewSuiClient("https://fullnode.testnet.sui.io:443")
	
	return &Client{
		SuiClient: suiClient,
	}, nil
}

// CallMoveFunction is a simplified wrapper around unsafe_moveCall.
// It requires senderAddress, packageID, and gasObjectID to be configured or passed.
// This function is a placeholder and needs proper parameters for actual use.
func (c *Client) CallMoveFunction(
	senderAddress string, // Example: "0xYOUR_SENDER_ADDRESS"
	packageID string,     // Example: "0xYOUR_PACKAGE_ID"
	module string,
	functionName string,
	typeArgs []string,    // Specific type arguments for the Move function, if any
	callArgs []interface{}, // Arguments for the Move function
	gasObjectID string,   // Object ID of the gas coin to be used for payment
	gasBudget uint64,     // Budget for gas in MIST
) (gjson.Result, error) {
	log.Printf("Calling Move function: Package=%s, Module=%s, Function=%s, Sender=%s", packageID, module, functionName, senderAddress)
	log.Printf("Type Args: %v", typeArgs)
	log.Printf("Call Args: %v", callArgs)
	log.Printf("Gas Object: %s, Gas Budget: %d", gasObjectID, gasBudget)

	if senderAddress == "" || packageID == "" || gasObjectID == "" {
		errMsg := "senderAddress, packageID, and gasObjectID must be provided for CallMoveFunction"
		log.Println(errMsg)
		return gjson.Result{}, fmt.Errorf(errMsg)
	}
	if gasBudget == 0 {
		gasBudget = 10000000 // Default gas budget if not specified, e.g., 0.01 SUI
		log.Printf("Using default gas budget: %d", gasBudget)
	}

	// The `unsafe_moveCall` expects arguments to be in specific formats (e.g., strings for object IDs).
	// Ensure `callArgs` are properly formatted based on the target Move function's signature.
	// For example, object IDs should be strings, numbers as strings or numbers based on type, etc.

	// This uses the embedded SuiClient's MoveCall method.
	result, err := c.SuiClient.MoveCall(senderAddress, packageID, module, functionName, typeArgs, callArgs, gasObjectID, gasBudget)
	if err != nil {
		log.Printf("Error calling Move function %s::%s::%s: %v. Raw Response: %s", packageID, module, functionName, err, result.Raw)
		return result, fmt.Errorf("MoveCall failed for %s::%s: %w", module, functionName, err)
	}

	log.Printf("Move function %s::%s::%s called successfully. Digest: %s", packageID, module, functionName, result.Get("digest").String())
	// The result from unsafe_moveCall is a JSON object containing txBytes, gas summary, and input objects.
	// To get the actual effects or created objects, this transaction needs to be signed and executed
	// via `sui_executeTransactionBlock`. This function only prepares the transaction.
	// For a complete flow, one might:
	// 1. Call `unsafe_moveCall` (or `sui_DevInspectTransactionBlock` for simulation).
	// 2. Get `txBytes` from the response.
	// 3. Sign `txBytes` using the sender's private key (client-side or secure enclave).
	// 4. Call `sui_executeTransactionBlock` with signed transaction.
	// This simplified `CallMoveFunction` does not perform signing or execution.
	return result, nil
}