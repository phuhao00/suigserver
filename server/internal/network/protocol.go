package network

// This file now only imports the protocol package
// All protocol-related types have been moved to server/internal/protocol

import (
	"github.com/phuhao00/suigserver/server/internal/protocol"
)

// Re-export commonly used types for backward compatibility
// These can be removed once all code is updated to import protocol directly
type ClientServerMessage = protocol.ClientServerMessage
type AuthRequestPayload = protocol.AuthRequestPayload
type AuthResponsePayload = protocol.AuthResponsePayload
type ErrorResponsePayload = protocol.ErrorResponsePayload
type SimpleMessagePayload = protocol.SimpleMessagePayload
type JoinRoomRequestPayload = protocol.JoinRoomRequestPayload
type JoinRoomResponsePayload = protocol.JoinRoomResponsePayload
type ChatMessagePayload = protocol.ChatMessagePayload
type PingPongPayload = protocol.PingPongPayload
type PlayerActionPayload = protocol.PlayerActionPayload
type PlayerActionResponsePayload = protocol.PlayerActionResponsePayload

// Re-export constants for backward compatibility
const (
	MsgTypeError                = protocol.MsgTypeError
	MsgTypeSimpleMessage        = protocol.MsgTypeSimpleMessage
	MsgTypeAuthRequest          = protocol.MsgTypeAuthRequest
	MsgTypeAuthResponse         = protocol.MsgTypeAuthResponse
	MsgTypeJoinRoomRequest      = protocol.MsgTypeJoinRoomRequest
	MsgTypeJoinRoomResponse     = protocol.MsgTypeJoinRoomResponse
	MsgTypeSendChat             = protocol.MsgTypeSendChat
	MsgTypeNewChatMessage       = protocol.MsgTypeNewChatMessage
	MsgTypePing                 = protocol.MsgTypePing
	MsgTypePong                 = protocol.MsgTypePong
	MsgTypePlayerAction         = protocol.MsgTypePlayerAction
	MsgTypePlayerActionResponse = protocol.MsgTypePlayerActionResponse
)
