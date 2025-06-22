package sui

import (
	"encoding/json"
	"fmt"
	"log"

	// "log" // Will be replaced by utils
	"strconv"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/phuhao00/suigserver/server/internal/utils" // For logging
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
func NewEventLogSuiService(suiClient *SuiClient, packageID, moduleName, senderAddress, gasObjectID string) *EventLogSuiService {
	utils.LogInfo("Initializing Event Log Sui Service...")
	if suiClient == nil {
		log.Panic("EventLogSuiService: SuiClient cannot be nil")
	}
	// packageID, moduleName, senderAddress, gasObjectID can be empty if only QueryGameEvents is used.
	// However, if LogGameEventViaCall is a primary function, these should be validated.
	// For now, allow them to be potentially empty, LogGameEventViaCall will validate if needed.
	return &EventLogSuiService{
		suiClient:     suiClient,
		packageID:     packageID,
		moduleName:    moduleName,
		senderAddress: senderAddress,
		gasObjectID:   gasObjectID,
	}
}

// LogGameEventViaCall prepares a transaction to record a game event by calling a Move function.
// Returns TxnMetaData for subsequent signing and execution.
// This uses the service's configured senderAddress and gasObjectID.
func (s *EventLogSuiService) LogGameEventViaCall(event GameEventData, gasBudget uint64) (models.TxnMetaData, error) {
	functionName := "log_custom_event" // Example Move function name
	utils.LogInfof("EventLogSuiService: Preparing to log game event via Move call: Type '%s', Creator: %s. GasBudget: %d",
		event.EventType, event.EventCreator, gasBudget)

	if s.packageID == "" || s.moduleName == "" || s.senderAddress == "" || s.gasObjectID == "" {
		errMsg := "missing packageID, moduleName, senderAddress, or gasObjectID for LogGameEventViaCall in EventLogSuiService config"
		utils.LogError("EventLogSuiService: " + errMsg)
		return models.TxnMetaData{}, fmt.Errorf(errMsg)
	}
	if event.EventType == "" || event.EventCreator == "" {
		errMsg := "EventType and EventCreator must be provided in GameEventData"
		utils.LogError("EventLogSuiService: " + errMsg)
		return models.TxnMetaData{}, fmt.Errorf(errMsg)
	}

	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		utils.LogErrorf("EventLogSuiService: Failed to marshal event payload for event type %s: %v", event.EventType, err)
		return models.TxnMetaData{}, fmt.Errorf("failed to marshal event payload: %w", err)
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
		utils.LogErrorf("EventLogSuiService: Error preparing LogGameEvent transaction (Type: %s): %v", event.EventType, err)
		return models.TxnMetaData{}, fmt.Errorf("MoveCall failed for LogGameEvent (Type: %s): %w", event.EventType, err)
	}

	utils.LogInfof("EventLogSuiService: LogGameEvent transaction prepared (Type: %s). TxBytes: %s",
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
) (models.PaginatedEventsResponse, error) {
	utils.LogInfof("EventLogSuiService: Querying game events: Filter='%s', Limit=%d, Descending=%t, CursorTx=%v, CursorSeq=%v",
		eventTypeFilter, limit, descendingOrder, cursorTxDigest, cursorEventSeq)

	if eventTypeFilter == "" {
		utils.LogError("EventLogSuiService: eventTypeFilter must be provided for QueryGameEvents")
		return models.PaginatedEventsResponse{}, fmt.Errorf("eventTypeFilter must be provided")
	}

	query := models.SuiEventFilter{
		// MoveEventType: &eventTypeFilter, // TODO: Fix field name - currently undefined
		// Other filters like Sender, TimeRange can be added to models.SuiEventFilter if needed
	}

	// Construct string cursor as expected by client.QueryEvents, which handles parsing.
	var cursorStrPointer *string
	if cursorTxDigest != nil && cursorEventSeq != nil {
		str := fmt.Sprintf("%s:%d", *cursorTxDigest, *cursorEventSeq)
		cursorStrPointer = &str
		utils.LogDebugf("EventLogSuiService: Using cursor string for query: %s", str)
	}

	var actualLimit uint64 = 50 // Default limit for query
	if limit > 0 {
		actualLimit = limit
	}
	limitPtr := &actualLimit

	resp, err := s.suiClient.QueryEvents(query, cursorStrPointer, limitPtr, descendingOrder)
	if err != nil {
		utils.LogErrorf("EventLogSuiService: Error querying events (Filter: %s): %v", eventTypeFilter, err)
		return models.PaginatedEventsResponse{}, fmt.Errorf("QueryEvents failed for filter %s: %w", eventTypeFilter, err)
	}

	utils.LogInfof("EventLogSuiService: Successfully queried events (Filter: %s). Found: %d events. HasNextPage: %t",
		eventTypeFilter, len(resp.Data), resp.HasNextPage)

	if resp.HasNextPage {
		utils.LogDebugf("EventLogSuiService: Next cursor for pagination available")
	}

	return resp, nil
}
