package protocol

// ClientServerMessage defines the standard structure for messages exchanged
// between client and server.
type ClientServerMessage struct {
	Type    string      `json:"type"`    // Defines the kind of message, e.g., "AUTH", "PLAYER_ACTION"
	Payload interface{} `json:"payload"` // Data specific to the message type
}

// AuthRequestPayload is the payload for an "AUTH" request from the client.
type AuthRequestPayload struct {
	Token string `json:"token"`
}

// AuthResponsePayload is the payload for an "AUTH_SUCCESS" or "AUTH_FAILURE" response.
type AuthResponsePayload struct {
	PlayerID string `json:"playerId,omitempty"` // Included on success
	Success  bool   `json:"success"`
	Message  string `json:"message"` // e.g., "Authentication successful" or error message
}

// ErrorResponsePayload is a generic payload for error messages.
type ErrorResponsePayload struct {
	Code    string `json:"code"` // e.g., "INVALID_COMMAND", "NOT_AUTHENTICATED"
	Message string `json:"message"`
}

// SimpleMessagePayload is for simple text messages to the client (e.g., welcome, usage)
type SimpleMessagePayload struct {
	Message string `json:"message"`
}

// JoinRoomRequestPayload is for "JOIN_ROOM" request from client
type JoinRoomRequestPayload struct {
	Criteria string `json:"criteria"` // e.g., room ID or type
}

// JoinRoomResponsePayload is for "ROOM_JOINED" or "JOIN_ROOM_FAILED"
type JoinRoomResponsePayload struct {
	Success bool   `json:"success"`
	RoomID  string `json:"roomId,omitempty"`
	Message string `json:"message"`
}

// ChatMessagePayload is for "SEND_CHAT" from client or "NEW_CHAT_MESSAGE" to client
type ChatMessagePayload struct {
	SenderName string `json:"senderName,omitempty"` // Server populates this for NEW_CHAT_MESSAGE
	Text       string `json:"text"`
}

// PingPongPayload can be empty or contain a timestamp, used for "PING" and "PONG"
type PingPongPayload struct {
	Timestamp int64 `json:"timestamp,omitempty"`
}

// PlayerActionPayload is a generic payload for player actions.
// The specific content of "Data" would vary based on the action.
type PlayerActionPayload struct {
	ActionType string                 `json:"actionType"` // e.g., "MOVE", "USE_ITEM"
	Data       map[string]interface{} `json:"data"`       // Action-specific data
}

// PlayerActionResponsePayload is for acknowledging player actions.
type PlayerActionResponsePayload struct {
	ActionType string                 `json:"actionType"`
	Status     string                 `json:"status"` // e.g., "SUCCESS", "FAILURE", "PENDING"
	Message    string                 `json:"message,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"` // For returning data, e.g., from GET_PLAYER_PROFILE
}

// Constants for message types
const (
	MsgTypeError                = "ERROR"
	MsgTypeSimpleMessage        = "SIMPLE_MESSAGE"
	MsgTypeAuthRequest          = "AUTH"
	MsgTypeAuthResponse         = "AUTH_RESPONSE"
	MsgTypeJoinRoomRequest      = "JOIN_ROOM"
	MsgTypeJoinRoomResponse     = "JOIN_ROOM_RESPONSE"
	MsgTypeSendChat             = "SEND_CHAT"
	MsgTypeNewChatMessage       = "NEW_CHAT_MESSAGE"
	MsgTypePing                 = "PING"
	MsgTypePong                 = "PONG"
	MsgTypePlayerAction         = "PLAYER_ACTION"
	MsgTypePlayerActionResponse = "PLAYER_ACTION_RESPONSE"
)
