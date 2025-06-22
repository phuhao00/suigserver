package sui

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/tidwall/gjson"
)

// GameEventData defines a structure for game events to be logged.
// This should align with what the Move contract expects or how it emits events.
type GameEventData struct {
	EventType      string                 `json:"event_type"`
	Timestamp      int64                  `json:"timestamp"` // Unix timestamp
	EventCreator   string                 `json:"event_creator"` // Address of player/system that triggered event
	RelatedObjects []string               `json:"related_objects"` // e.g., Player IDs, Item IDs, NPC IDs
	Payload        map[string]interface{} `json:"payload"`       // Specific data for this event type
}

// EventLogSuiService is responsible for writing significant game events to the Sui blockchain
// and querying them.
type EventLogSuiService struct {
	suiClient     *Client
	packageID     string // ID of the package containing the event logging module (if events are stored via contract calls)
	moduleName    string // Name of the Move module for event logging
	senderAddress string // Address for sending event logging transactions (if applicable)
	gasObjectID   string // Gas object for these transactions
}

// NewEventLogSuiService creates a new EventLogSuiService.
func NewEventLogSuiService(client *Client, packageID, moduleName, senderAddress, gasObjectID string) *EventLogSuiService {
	log.Println("Initializing Event Log Sui Service...")
	if client == nil {
		log.Panic("EventLogSuiService: Sui client cannot be nil")
	}
	// packageID, moduleName, senderAddress, gasObjectID might be optional if only querying events.
	// For logging events via contract call, they are necessary.
	return &EventLogSuiService{
		suiClient:     client,
		packageID:     packageID,
		moduleName:    moduleName,
		senderAddress: senderAddress,
		gasObjectID:   gasObjectID,
	}
}

// LogGameEventViaCall prepares a transaction to record a game event by calling a Move function.
// This is one way to log events, especially if the event itself triggers state changes or needs on-chain validation.
func (s *EventLogSuiService) LogGameEventViaCall(event GameEventData, gasBudget uint64) (gjson.Result, error) {
	functionName := "log_custom_event" // Example Move function name
	log.Printf("Preparing to log game event via Move call: Type '%s', Creator: %s", event.EventType, event.EventCreator)

	if s.packageID == "" || s.moduleName == "" || s.senderAddress == "" || s.gasObjectID == "" {
		return gjson.Result{}, fmt.Errorf("missing packageID, moduleName, senderAddress, or gasObjectID for LogGameEventViaCall")
	}

	// Serialize payload to JSON string if the Move function expects it that way.
	// Alternatively, flatten into primitive arguments if the function takes them directly.
	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return gjson.Result{}, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	callArgs := []interface{}{
		event.EventType,
		strconv.FormatInt(event.Timestamp, 10),
		event.EventCreator,
		event.RelatedObjects, // Assumes Move function accepts a vector<address> or vector<object_id>
		string(payloadJSON),
	}
	typeArgs := []string{}

	txResult, err := s.suiClient.CallMoveFunction(
		s.senderAddress,
		s.packageID,
		s.moduleName,
		functionName,
		typeArgs,
		callArgs,
		s.gasObjectID,
		gasBudget,
	)

	if err != nil {
		log.Printf("Error preparing LogGameEvent transaction (Type: %s): %v", event.EventType, err)
		return txResult, err
	}

	log.Printf("LogGameEvent transaction prepared (Type: %s). Digest (from unsafe_moveCall): %s",
		event.EventType, txResult.Get("digest").String())
	// Remember: txResult needs to be signed and executed.
	return txResult, nil
}

// QueryGameEvents retrieves past game events using Sui's event querying capabilities.
// eventTypeFilter should be the fully qualified event type string, e.g., "0xPACKAGE::MODULE::EventName"
func (s *EventLogSuiService) QueryGameEvents(
	eventTypeFilter string,
	cursor *EventID, // For pagination, from previous response
	limit uint64,
	descendingOrder bool,
) (gjson.Result, error) {
	log.Printf("Querying game events: Filter='%s', Limit=%d, Descending=%t, Cursor=%v",
		eventTypeFilter, limit, descendingOrder, cursor)

	// TODO: Implement actual Sui event query.
	// This uses the `suix_queryEvents` RPC endpoint via `s.suiClient.QueryEvents`.

	query := map[string]interface{}{
		"MoveEventType": eventTypeFilter,
		// Other filters can be added, e.g., "SenderAddress", "TimeRange"
		// "SenderAddress": "0xSENDER_ADDRESS_IF_NEEDED",
		// "TimeRange": { "startTime": "timestamp_ms_start", "endTime": "timestamp_ms_end" }
	}

	var cursorStr *string
	if cursor != nil {
		// Assuming EventID struct has TxDigest and EventSeq which are needed for cursor
		// The exact format of the cursor string might depend on the API or be opaque.
		// For suix_queryEvents, the cursor is `{"txDigest": string, "eventSeq": string}`.
		// However, the `QueryEvents` method in `client.go` expects a simple string.
		// This needs alignment. For now, let's assume cursor.TxDigest is what's needed or it's a combined string.
		// This part needs to be verified with how `client.QueryEvents` handles the cursor.
		// For now, this is a placeholder.
		tempCursor := fmt.Sprintf(`{"txDigest":"%s","eventSeq":"%s"}`, cursor.TxDigest, cursor.EventSeq)
		cursorStr = &tempCursor
		// If client.QueryEvents expects only txDigest or a specific format, adjust accordingly.
		// cursorStr = &cursor.TxDigest // Example if only TxDigest is used
	}


	if limit == 0 {
		limit = 50 // Default limit
	}
	var limitPtr *uint64 = &limit


	result, err := s.suiClient.QueryEvents(query, cursorStr, limitPtr, descendingOrder)
	if err != nil {
		log.Printf("Error querying events (Filter: %s): %v", eventTypeFilter, err)
		return gjson.Result{}, err
	}

	log.Printf("Successfully queried events (Filter: %s). Found: %d events. HasNextPage: %t",
		eventTypeFilter, len(result.Get("data").Array()), result.Get("hasNextPage").Bool())

	// The result.Get("data") will be an array of events.
	// Each event has fields like: packageId, transactionModule, sender, type, parsedJson, bcs, timestampMs.
	// Example:
	// for _, event := range result.Get("data").Array() {
	// log.Printf("Event Type: %s, Sender: %s, Parsed JSON: %v",
	// event.Get("type").String(),
	// event.Get("sender").String(),
	// event.Get("parsedJson").Value())
	// }
	// nextCursor := result.Get("nextCursor.txDigest").String() // Example, adjust based on actual cursor structure
	// log.Printf("Next cursor for pagination: %s", nextCursor)

	return result, nil
}
