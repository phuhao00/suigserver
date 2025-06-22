package sui

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/block-vision/sui-go-sdk/models"
)

// GameEventData defines a structure for game events to be logged.
// This should align with what the Move contract expects or how it emits events.
type GameEventData struct {
	EventType      string                 `json:"event_type"`
	Timestamp      int64                  `json:"timestamp"`       // Unix timestamp
	EventCreator   string                 `json:"event_creator"`   // Address of player/system that triggered event
	RelatedObjects []string               `json:"related_objects"` // e.g., Player IDs, Item IDs, NPC IDs
	Payload        map[string]interface{} `json:"payload"`         // Specific data for this event type
}

// EventLogSuiService is responsible for writing significant game events to the Sui blockchain
// and querying them.
type EventLogSuiService struct {
	suiClient     *SuiClient // Use the refactored SuiClient
	packageID     string     // ID of the package containing the event logging module (if events are stored via contract calls)
	moduleName    string     // Name of the Move module for event logging
	senderAddress string     // Address for sending event logging transactions (if applicable)
	gasObjectID   string     // Gas object for these transactions
}

// NewEventLogSuiService creates a new EventLogSuiService.
func NewEventLogSuiService(client *SuiClient, packageID, moduleName, senderAddress, gasObjectID string) *EventLogSuiService {
	log.Println("Initializing Event Log Sui Service...")
	if client == nil {
		log.Panic("EventLogSuiService: Sui client cannot be nil")
	}
	return &EventLogSuiService{
		suiClient:     client,
		packageID:     packageID,
		moduleName:    moduleName,
		senderAddress: senderAddress,
		gasObjectID:   gasObjectID,
	}
}

// LogGameEventViaCall prepares a transaction to record a game event by calling a Move function.
// Returns TransactionBlockResponse for subsequent signing and execution.
func (s *EventLogSuiService) LogGameEventViaCall(event GameEventData, gasBudget uint64) (models.TransactionBlockResponse, error) {
	functionName := "log_custom_event" // Example Move function name
	log.Printf("Preparing to log game event via Move call: Type '%s', Creator: %s", event.EventType, event.EventCreator)

	if s.packageID == "" || s.moduleName == "" || s.senderAddress == "" || s.gasObjectID == "" {
		return models.TransactionBlockResponse{}, fmt.Errorf("missing packageID, moduleName, senderAddress, or gasObjectID for LogGameEventViaCall")
	}

	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return models.TransactionBlockResponse{}, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	callArgs := []interface{}{
		event.EventType,
		strconv.FormatInt(event.Timestamp, 10),
		event.EventCreator,
		event.RelatedObjects,
		string(payloadJSON),
	}
	typeArgs := []string{}

	txBlockResponse, err := s.suiClient.MoveCall(
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
		return models.TransactionBlockResponse{}, err
	}

	log.Printf("LogGameEvent transaction prepared (Type: %s). TxBytes: %s",
		event.EventType, txBlockResponse.TxBytes)
	return txBlockResponse, nil
}

// QueryGameEvents retrieves past game events using Sui's event querying capabilities.
// eventTypeFilter should be the fully qualified event type string, e.g., "0xPACKAGE::MODULE::EventName"
func (s *EventLogSuiService) QueryGameEvents(
	eventTypeFilter string,
	cursorTxDigest *string, // Transaction digest for pagination cursor
	cursorEventSeq *uint64, // Event sequence number for pagination cursor
	limit uint64,
	descendingOrder bool,
) (models.SuiQueryEventsResponse, error) {
	log.Printf("Querying game events: Filter='%s', Limit=%d, Descending=%t, CursorTx=%v, CursorSeq=%v",
		eventTypeFilter, limit, descendingOrder, cursorTxDigest, cursorEventSeq)

	query := models.EventFilter{
		MoveEventType: &eventTypeFilter,
		// Other filters like Sender, TimeRange can be added to models.EventFilter if needed
	}

	var actualCursor *models.EventId
	if cursorTxDigest != nil && cursorEventSeq != nil {
		actualCursor = &models.EventId{
			TxDigest: *cursorTxDigest,
			EventSeq: *cursorEventSeq,
		}
	}

	if limit == 0 {
		limit = 50 // Default limit
	}

	// The client.QueryEvents was already updated to take *string for cursor,
	// but sui-go-sdk takes *models.EventId. We need to adjust the client.go or here.
	// For now, let's assume client.QueryEvents is called directly if it was adapted,
	// or we pass the structured cursor if we are calling the SDK method directly here.
	// The current plan step is to update THIS file (event_logs.go) to use the SDK.
	// So, we should use the SDK's QueryEvents method signature.

	// The `s.suiClient.QueryEvents` method in `client.go` was refactored to accept `models.EventFilter`
	// and a string cursor. This means `client.go` needs to parse the string cursor into `models.EventId`.
	// Let's construct the string cursor as expected by our refactored `client.go`'s `QueryEvents` method.
	var cursorStr *string
	if actualCursor != nil {
		str := fmt.Sprintf("%s:%d", actualCursor.TxDigest, actualCursor.EventSeq)
		cursorStr = &str
	}

	// We need to pass a pointer to limit for the client.QueryEvents method
	var limitPtr *uint64
	if limit > 0 {
		limitPtr = &limit
	}

	resp, err := s.suiClient.QueryEvents(query, cursorStr, limitPtr, descendingOrder)
	if err != nil {
		log.Printf("Error querying events (Filter: %s): %v", eventTypeFilter, err)
		return models.SuiQueryEventsResponse{}, err
	}

	log.Printf("Successfully queried events (Filter: %s). Found: %d events. HasNextPage: %t",
		eventTypeFilter, len(resp.Data), resp.HasNextPage)

	// Example of accessing next cursor from SDK response
	// if resp.HasNextPage && resp.NextCursor != nil {
	// 	log.Printf("Next cursor for pagination: TxDigest=%s, EventSeq=%d", resp.NextCursor.TxDigest, resp.NextCursor.EventSeq)
	// }

	return resp, nil
}
