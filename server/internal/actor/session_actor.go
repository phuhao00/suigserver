package actor

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	// "log" // Replaced by utils.LogX
	"net"  // For basic message parsing, will be replaced by proper protocol
	"time" // For heartbeat

	"github.com/asynkron/protoactor-go/actor"
	"github.com/block-vision/sui-go-sdk/models" // For SUI SDK types
	"github.com/phuhao00/suigserver/server/internal/actor/messages"
	"github.com/phuhao00/suigserver/server/internal/protocol" // For protocol definitions
	"github.com/phuhao00/suigserver/server/internal/sui"      // For SUI client
	"github.com/phuhao00/suigserver/server/internal/utils"    // Logger
)

// PlayerSessionActor manages a single client's connection and game session.
type PlayerSessionActor struct {
	conn            net.Conn
	actorSystem     *actor.ActorSystem // To interact with other actors
	playerID        string             // Set after authentication
	roomPID         *actor.PID         // PID of the room the player is currently in
	roomManagerPID  *actor.PID         // PID of the RoomManagerActor
	worldManagerPID *actor.PID         // PID of the WorldManagerActor, to be injected or discovered
	suiClient       *sui.SuiClient     // SUI client instance
	// Auth Config
	enableDummyAuth bool
	dummyToken      string
	dummyPlayerID   string
	// other player-specific state

	lastActivity    time.Time     // Time of last message from client or significant activity
	heartbeatStopCh chan struct{} // Channel to stop heartbeat goroutine (if any server-side ping)
}

// NewPlayerSessionActor creates a new PlayerSessionActor instance.
// It now also requires a suiClient and dummy auth configurations.
func NewPlayerSessionActor(
	system *actor.ActorSystem,
	roomManagerPID *actor.PID,
	worldManagerPID *actor.PID,
	suiClient *sui.SuiClient,
	enableDummyAuth bool,
	dummyToken string,
	dummyPlayerID string,
) actor.Actor {
	if suiClient == nil {
		utils.LogFatalf("PlayerSessionActor: suiClient cannot be nil")
	}
	if enableDummyAuth && (dummyToken == "" || dummyPlayerID == "") {
		utils.LogFatalf("PlayerSessionActor: Dummy auth enabled but token or PlayerID is empty")
	}
	return &PlayerSessionActor{
		actorSystem:     system,
		roomManagerPID:  roomManagerPID,
		worldManagerPID: worldManagerPID,
		suiClient:       suiClient,
		enableDummyAuth: enableDummyAuth,
		dummyToken:      dummyToken,
		dummyPlayerID:   dummyPlayerID,
		heartbeatStopCh: make(chan struct{}),
	}
}

// PropsForPlayerSession creates actor.Props for a PlayerSessionActor.
// It's a helper to ensure all necessary dependencies are passed for actor creation.
func PropsForPlayerSession(
	system *actor.ActorSystem,
	roomManagerPID *actor.PID,
	worldManagerPID *actor.PID,
	suiClient *sui.SuiClient,
	enableDummyAuth bool,
	dummyToken string,
	dummyPlayerID string,
) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return NewPlayerSessionActor(system, roomManagerPID, worldManagerPID, suiClient, enableDummyAuth, dummyToken, dummyPlayerID)
	})
}

const (
	// clientActivityTimeout is the duration after which a client is disconnected if no messages are received.
	clientActivityTimeout = 90 * time.Second
	// authTimeout is the time allowed for a client to authenticate after connecting.
	authTimeout = 60 * time.Second
)

// TODO: These constants (placeholder...PackageID, placeholder...Module) should be made properly configurable
// via the main config file and passed down to PlayerSessionActor. This is part of the
// "Configuration for New SUI Placeholders" step (Step 4) of the "Server & SUI Logic Enhancement - Phase 2" plan.
const placeholderGameLogicPackageID = "0xPLACEHOLDER_GAME_LOGIC_PACKAGE_ID"       // Used for PERFORM_INGAME_ACTION
const placeholderPlayerObjectPackageID = "0xPLACEHOLDER_PLAYER_OBJECT_PACKAGE_ID" // Used for GET_PLAYER_PROFILE
const placeholderPlayerObjectModule = "player_profile"                            // Used for GET_PLAYER_PROFILE

// Receive is the main message handling loop for the PlayerSessionActor.
func (a *PlayerSessionActor) Receive(ctx actor.Context) {
	actorID := ctx.Self().Id
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		utils.LogInfof("[%s] PlayerSessionActor started.", actorID)
		// An initial timeout for ClientConnected could be set here if TCPServer doesn't guarantee it.
		// For now, we assume ClientConnected will arrive shortly.
		// If ClientConnected doesn't arrive, this actor might become a zombie.
		// Consider a timeout here if ClientConnected is not guaranteed.

	case *actor.Stopping:
		utils.LogInfof("[%s] PlayerSessionActor stopping. PlayerID: %s", actorID, a.playerID)
		if a.conn != nil {
			a.conn.Close() // Ensure connection is closed when actor stops
		}
		a.cleanupResources(ctx) // Cleanup heartbeat resources and notify other systems

	case *actor.Stopped:
		utils.LogInfof("[%s] PlayerSessionActor stopped. PlayerID: %s", actorID, a.playerID)

	case *actor.ReceiveTimeout:
		utils.LogWarnf("[%s] ReceiveTimeout for player %s. No client activity or authentication in time. Stopping session.", actorID, a.playerID)
		if a.conn != nil {
			timeoutMsg := "Timeout due to inactivity."
			if !a.isAuthenticated() {
				timeoutMsg = "Timeout: Authentication not completed in time."
			}
			utils.LogInfof("[%s] Sending timeout disconnect message to player %s: %s", actorID, a.playerID, timeoutMsg)
			a.sendErrorResponse("TIMEOUT", timeoutMsg+" Disconnecting.")
			a.conn.Close()
		}
		ctx.Stop(ctx.Self())

	case *messages.ClientConnected:
		utils.LogInfof("[%s] Received ClientConnected from %s", actorID, msg.Conn.RemoteAddr())
		a.conn = msg.Conn
		a.lastActivity = time.Now()
		ctx.SetReceiveTimeout(authTimeout) // Client has this much time to send auth command

		// Request authentication using the new JSON protocol
		a.sendSimpleMessage("Welcome! Please authenticate. Send JSON: {\"type\":\"AUTH\",\"payload\":{\"token\":\"your_token\"}}")

	case *messages.ClientMessage:
		utils.LogDebugf("[%s] Received ClientMessage from player %s: %s", actorID, a.playerID, string(msg.Payload))
		a.lastActivity = time.Now() // Update last activity time on any client message

		if !a.isAuthenticated() {
			// If not authenticated, keep the authTimeout active.
			// Any message resets it, giving client more time for the 'auth' command.
			ctx.SetReceiveTimeout(authTimeout)
		} else {
			// If authenticated, switch to clientActivityTimeout for general inactivity.
			ctx.SetReceiveTimeout(clientActivityTimeout)
		}
		a.handleClientPayload(ctx, msg.Payload)

	case *messages.ForwardToClient:
		a.handleForwardToClient(msg)

	case *messages.ClientDisconnected:
		utils.LogInfof("[%s] Received ClientDisconnected for player %s: %s. Cleaning up.", actorID, a.playerID, msg.Reason)
		// If in a room, notify the room actor
		if a.roomPID != nil {
			ctx.Send(a.roomPID, &messages.LeaveRoomRequest{PlayerID: a.playerID, PlayerPID: ctx.Self()})
		}
		// Other cleanup is handled in the *actor.Stopping case, which will be triggered by ctx.Stop(ctx.Self())
		if a.conn != nil {
			a.conn.Close() // Ensure conn is closed
		}
		ctx.Stop(ctx.Self()) // Stop this actor instance

	case *messages.AuthenticatePlayer:
		utils.LogInfof("[%s] Authenticating player (from internal msg, token: %s)", actorID, msg.Token)
		// Actual authentication logic placeholder
		// In a real app, this would involve checking against a database or auth service.
		// The token should be securely handled.
		success := false
		// PlayerID from msg.PlayerID is ignored. PlayerID is determined by the validated token.
		if a.enableDummyAuth {
			if msg.Token == a.dummyToken {
				a.playerID = a.dummyPlayerID // Set playerID based on the valid dummy token
				success = true
			}
		} else {
			// If dummy auth is not enabled, no token will be considered valid by this logic.
			// A real auth system would be integrated here.
			utils.LogWarnf("[%s] Dummy authentication is disabled. Player (token: %s) authentication failed.", actorID, msg.Token)
		}

		if success {
			a.lastActivity = time.Now()
			ctx.CancelReceiveTimeout()                   // Authentication successful, cancel auth timeout
			ctx.SetReceiveTimeout(clientActivityTimeout) // Start general client activity timeout
			utils.LogInfof("[%s] Player %s authenticated successfully.", actorID, a.playerID)

			// Notify WorldManager that player has entered
			// The WorldManagerPID should be available to the PlayerSessionActor,
			// e.g., passed during creation or retrieved from a well-known actor registry.
			if a.worldManagerPID != nil {
				utils.LogInfof("[%s] Notifying WorldManager that player %s has entered.", actorID, a.playerID)
				ctx.Send(a.worldManagerPID, &messages.PlayerEnteredWorld{PlayerID: a.playerID, PlayerPID: ctx.Self()})
			} else {
				utils.LogWarnf("[%s] WorldManagerPID not set for player %s. Cannot notify about entering world.", actorID, a.playerID)
			}

		} else {
			utils.LogWarnf("[%s] Player (token: %s) authentication failed (invalid token or dummy auth disabled).", actorID, msg.Token)
			// Error response is now handled by the block sending AuthResponsePayload with Success: false
			ctx.SetReceiveTimeout(authTimeout)
		}

		// PlayerAuthenticated message is internal, not strictly needed if client response is primary.
		// authResponse := &messages.PlayerAuthenticated{
		// 	PlayerID: a.playerID, // use a.playerID as it's now set
		// 	Success:  success,
		// }
		// if ctx.Sender() != nil {
		// 	ctx.Respond(authResponse)
		// }

		// Send JSON response to client
		if success {
			a.sendResponse(protocol.MsgTypeAuthResponse, protocol.AuthResponsePayload{
				PlayerID: a.playerID, // PlayerID is now set on 'a'
				Success:  true,
				Message:  "Authentication successful.",
			})
		} else {
			a.sendResponse(protocol.MsgTypeAuthResponse, protocol.AuthResponsePayload{
				Success: false,
				Message: "Authentication failed. Invalid token or authentication method disabled.",
			})
		}

	case *messages.FindRoomResponse: // Response from RoomManagerActor
		utils.LogInfof("[%s] Player %s received FindRoomResponse: Found=%t, RoomID=%s, RoomPID=%s, Error=%s",
			actorID, a.playerID, msg.Found, msg.RoomID, msg.RoomPID, msg.Error)
		if msg.Found && msg.RoomPID != nil {
			joinReq := &messages.JoinRoomRequest{
				PlayerID:  a.playerID,
				PlayerPID: ctx.Self(),
			}
			ctx.Request(msg.RoomPID, joinReq) // Request to join the actual room
		} else {
			responseMessage := "Error finding room."
			if msg.Error != "" {
				responseMessage = msg.Error
			} else if !msg.Found {
				responseMessage = "Room not found for the given criteria."
			}
			a.sendResponse(protocol.MsgTypeJoinRoomResponse, protocol.JoinRoomResponsePayload{
				Success: false,
				Message: responseMessage,
			})
		}

	case *messages.JoinRoomResponse: // Response from a RoomActor
		if msg.Success {
			a.roomPID = ctx.Sender() // Assume sender is the RoomActor
			utils.LogInfof("[%s] Player %s successfully joined room %s (RoomActor PID: %s)", actorID, a.playerID, msg.RoomID, a.roomPID.Id)
			a.sendResponse(protocol.MsgTypeJoinRoomResponse, protocol.JoinRoomResponsePayload{
				Success: true,
				RoomID:  msg.RoomID,
				Message: "Successfully joined room: " + msg.RoomID,
			})
		} else {
			utils.LogWarnf("[%s] Player %s failed to join room %s: %s", actorID, a.playerID, msg.RoomID, msg.Error)
			a.sendResponse(protocol.MsgTypeJoinRoomResponse, protocol.JoinRoomResponsePayload{
				Success: false,
				RoomID:  msg.RoomID,
				Message: "Failed to join room " + msg.RoomID + ": " + msg.Error,
			})
		}

	case *messages.RoomChatMessage: // Received from a RoomActor to be forwarded to this client
		chatPayload := protocol.ChatMessagePayload{
			SenderName: msg.SenderName,
			Text:       msg.Message,
		}
		a.sendResponse(protocol.MsgTypeNewChatMessage, chatPayload)

	default:
		utils.LogWarnf("[%s] PlayerSessionActor %s received unknown message type %T: %+v", actorID, a.playerID, msg, msg)
	}
}

// cleanupResources performs necessary cleanup when the actor is stopping.
func (a *PlayerSessionActor) cleanupResources(ctx actor.Context) {
	actorID := ctx.Self().Id
	utils.LogInfof("[%s] Cleaning up resources for player %s.", actorID, a.playerID)
	ctx.CancelReceiveTimeout() // Cancel any pending receive timeout

	if a.playerID != "" {
		if a.worldManagerPID != nil {
			utils.LogInfof("[%s] Notifying WorldManager that player %s has left.", actorID, a.playerID)
			ctx.Send(a.worldManagerPID, &messages.PlayerLeftWorld{PlayerID: a.playerID, PlayerPID: ctx.Self()})
		} else {
			utils.LogWarnf("[%s] WorldManagerPID not set for player %s. Cannot notify WorldManager about leaving.", actorID, a.playerID)
		}
		utils.LogInfof("[%s] Player %s disconnected. Placeholder: Trigger save player data mechanism.", actorID, a.playerID)
	}
}

// handleClientPayload parses the raw payload from the client and decides what to do.
func (a *PlayerSessionActor) handleClientPayload(ctx actor.Context, rawPayload []byte) {
	actorID := ctx.Self().Id
	var msg protocol.ClientServerMessage
	if err := json.Unmarshal(rawPayload, &msg); err != nil {
		utils.LogWarnf("[%s] Player %s: Error unmarshaling client message: %v. Payload: '%s'", actorID, a.playerID, err, string(rawPayload))
		a.sendErrorResponse("INVALID_JSON", "Message is not valid JSON.")
		return
	}

	utils.LogDebugf("[%s] Player %s received message type '%s', Payload: %+v", actorID, a.playerID, msg.Type, msg.Payload)

	var payloadMap map[string]interface{}
	if msg.Payload != nil {
		payloadBytes, err := json.Marshal(msg.Payload)
		if err != nil {
			utils.LogErrorf("[%s] Player %s: Error re-marshaling payload for type '%s': %v", actorID, a.playerID, msg.Type, err)
			a.sendErrorResponse("INVALID_PAYLOAD_STRUCTURE", "Cannot process payload structure.")
			return
		}
		if err := json.Unmarshal(payloadBytes, &payloadMap); err != nil {
			utils.LogWarnf("[%s] Player %s: Payload for type '%s' is not a JSON object: %v", actorID, a.playerID, msg.Type, err)
		}
	}

	switch msg.Type {
	case protocol.MsgTypeAuthRequest:
		if a.isAuthenticated() {
			utils.LogWarnf("[%s] Player %s: Already authenticated, received another AUTH request.", actorID, a.playerID)
			a.sendErrorResponse("ALREADY_AUTHENTICATED", "You are already authenticated.")
			return
		}
		var authReqPayload protocol.AuthRequestPayload
		payloadBytes, _ := json.Marshal(msg.Payload)
		if err := json.Unmarshal(payloadBytes, &authReqPayload); err != nil {
			utils.LogWarnf("[%s] Player (no ID yet): Invalid AUTH payload structure: %v", actorID, err)
			a.sendErrorResponse("INVALID_AUTH_PAYLOAD", "Auth payload is malformed.")
			return
		}
		tempPlayerID := "player_awaiting_auth"
		authInternalMsg := &messages.AuthenticatePlayer{
			PlayerID: tempPlayerID,
			Token:    authReqPayload.Token,
		}
		ctx.Request(ctx.Self(), authInternalMsg)

	case protocol.MsgTypeJoinRoomRequest:
		if !a.isAuthenticated() {
			a.sendErrorResponse("NOT_AUTHENTICATED", "Please authenticate first.")
			return
		}
		var joinReqPayload protocol.JoinRoomRequestPayload
		payloadBytes, _ := json.Marshal(msg.Payload)
		if err := json.Unmarshal(payloadBytes, &joinReqPayload); err != nil {
			utils.LogWarnf("[%s] Player %s: Invalid JOIN_ROOM payload: %v", actorID, a.playerID, err)
			a.sendErrorResponse("INVALID_JOIN_PAYLOAD", "Join room payload is malformed.")
			return
		}

		if joinReqPayload.Criteria == "" {
			utils.LogWarnf("[%s] Player %s: Join room criteria is empty.", actorID, a.playerID)
			a.sendErrorResponse("INVALID_JOIN_CRITERIA", "Join room criteria cannot be empty.")
			return
		}

		utils.LogInfof("[%s] Player %s attempting to find and join room with criteria: '%s'", actorID, a.playerID, joinReqPayload.Criteria)
		if a.roomManagerPID == nil {
			utils.LogErrorf("[%s] Player %s: RoomManagerPID not configured. Cannot join room.", actorID, a.playerID)
			a.sendResponse(protocol.MsgTypeJoinRoomResponse, protocol.JoinRoomResponsePayload{
				Success: false,
				Message: "Error: Room manager is not available.",
			})
			return
		}

		ctx.Request(a.roomManagerPID, &messages.FindRoomRequest{
			Criteria:  joinReqPayload.Criteria,
			PlayerPID: ctx.Self(),
		})
		a.sendSimpleMessage(fmt.Sprintf("Attempting to find and join room '%s'...", joinReqPayload.Criteria))

	case protocol.MsgTypeSendChat:
		if !a.isAuthenticated() {
			a.sendErrorResponse("NOT_AUTHENTICATED", "Please authenticate first.")
			return
		}
		if a.roomPID == nil {
			a.sendErrorResponse("NOT_IN_A_ROOM", "You are not in a room. Join a room first.")
			return
		}
		var chatReqPayload protocol.ChatMessagePayload
		payloadBytes, _ := json.Marshal(msg.Payload)
		if err := json.Unmarshal(payloadBytes, &chatReqPayload); err != nil {
			utils.LogWarnf("[%s] Player %s: Invalid SEND_CHAT payload: %v", actorID, a.playerID, err)
			a.sendErrorResponse("INVALID_CHAT_PAYLOAD", "Chat payload is malformed.")
			return
		}
		if chatReqPayload.Text == "" {
			utils.LogWarnf("[%s] Player %s: Empty chat message received.", actorID, a.playerID)
			a.sendErrorResponse("EMPTY_CHAT_MESSAGE", "Chat message cannot be empty.")
			return
		}
		utils.LogInfof("[%s] Player %s sends chat to room %s: %s", actorID, a.playerID, a.roomPID.Id, chatReqPayload.Text)
		roomChatMessageInternal := &messages.RoomChatMessage{
			SenderID:   a.playerID,
			SenderName: a.playerID,
			Message:    chatReqPayload.Text,
		}
		ctx.Send(a.roomPID, &messages.BroadcastToRoom{
			SenderPID:     ctx.Self(),
			ActualMessage: roomChatMessageInternal,
		})

	case protocol.MsgTypePing:
		utils.LogDebugf("[%s] Player %s received PING.", actorID, a.playerID)
		var pingPayload protocol.PingPongPayload
		payloadBytes, _ := json.Marshal(msg.Payload)
		if err := json.Unmarshal(payloadBytes, &pingPayload); err != nil {
			utils.LogWarnf("[%s] Player %s: PING payload malformed: %v", actorID, a.playerID, err)
		}
		a.sendResponse(protocol.MsgTypePong, pingPayload)

	case protocol.MsgTypePlayerAction:
		if !a.isAuthenticated() {
			a.sendErrorResponse("NOT_AUTHENTICATED", "Please authenticate first.")
			return
		}
		var actionPayload protocol.PlayerActionPayload
		payloadBytes, _ := json.Marshal(msg.Payload)
		if err := json.Unmarshal(payloadBytes, &actionPayload); err != nil {
			utils.LogWarnf("[%s] Player %s: Invalid PLAYER_ACTION payload: %v", actorID, a.playerID, err)
			a.sendErrorResponse("INVALID_ACTION_PAYLOAD", "Player action payload is malformed.")
			return
		}

		utils.LogInfof("[%s] Player %s: Received PLAYER_ACTION: Type=%s, Data=%+v. SUI Client available: %t",
			actorID, a.playerID, actionPayload.ActionType, actionPayload.Data, a.suiClient != nil)

		switch actionPayload.ActionType {
		case "GET_PLAYER_PROFILE":
			// Using new constants for placeholder SUI object details
			playerObjectStructName := "PlayerProfile" // Example struct name on SUI, could also be a constant or config

			// Simulate deriving player's SUI object ID.
			// This is a placeholder; actual mechanism might involve a registry contract.
			simulatedPlayerSuiObjectID := fmt.Sprintf("0xSIMULATED_PLAYER_OBJECT_FOR_%s", a.playerID)
			utils.LogInfof("[%s] Player %s: Action %s. Simulating SUI GetObject for object ID: %s",
				actorID, a.playerID, actionPayload.ActionType, simulatedPlayerSuiObjectID)

			// Simulate a successful SUI GetObject call and construct a mock models.SuiObjectResponse.
			// This demonstrates how the server would interact with the SDK types.
			mockSuiObjectData := models.SuiObjectData{
				ObjectId: simulatedPlayerSuiObjectID,
				Version:  "1",
				Digest:   "SIMULATED_OBJECT_DIGEST",
				Type:     fmt.Sprintf("%s::%s::%s", placeholderPlayerObjectPackageID, placeholderPlayerObjectModule, playerObjectStructName),
				Owner:    &models.ObjectOwner{AddressOwner: a.playerID}, // Assuming player owns their profile object
				Content: &models.SuiParsedData{
					DataType: "moveObject",
					// Note: Fields structure may vary - using a generic approach
				},
			}

			// Simulated player data fields that would normally be in the Content.Fields
			simulatedFields := map[string]interface{}{ // These are the SUI object's fields
				"game_player_id": a.playerID, // Field storing the link to the game's internal player ID
				"name":           fmt.Sprintf("Player %s", a.playerID),
				"level":          uint64(15), // Example: SUI often uses u64 for numbers
				"xp":             uint64(5500),
				"health_points":  uint64(120),
				"attack_power":   uint64(25),
				"last_seen_tsms": uint64(time.Now().UnixMilli()), // Example timestamp
			}
			// In a real call: suiObjectResponse, err := a.suiClient.GetObject(context.Background(), simulatedPlayerSuiObjectID)
			// Then check err and process suiObjectResponse.

			// "Parse" the simulated SUI response to populate the data for the client.
			var clientResponseData map[string]interface{}
			if mockSuiObjectData.Content != nil {
				// Use the simulated fields since actual SDK fields structure varies
				clientResponseData = map[string]interface{}{
					"playerId":    simulatedFields["game_player_id"],
					"name":        simulatedFields["name"],
					"level":       simulatedFields["level"],
					"xp":          simulatedFields["xp"],
					"hp":          simulatedFields["health_points"],
					"atk":         simulatedFields["attack_power"],
					"lastSeenMs":  simulatedFields["last_seen_tsms"],
					"suiObjectId": mockSuiObjectData.ObjectId,
					"suiVersion":  mockSuiObjectData.Version,
				}
				utils.LogInfof("[%s] Player %s: Successfully simulated parsing of SUI player profile object for GET_PLAYER_PROFILE.", actorID, a.playerID)
			} else {
				utils.LogWarnf("[%s] Player %s: Simulated SUI player profile object for GET_PLAYER_PROFILE was empty or malformed.", actorID, a.playerID)
				clientResponseData = map[string]interface{}{"error": "Failed to retrieve or parse player SUI data (simulated)."}
			}

			a.sendResponse(protocol.MsgTypePlayerActionResponse, protocol.PlayerActionResponsePayload{
				ActionType: actionPayload.ActionType,
				Status:     "SIMULATED_SUI_GET_OBJECT_SUCCESS",
				Message:    "Player profile data retrieved (simulated SUI GetObject).",
				Data:       clientResponseData,
			})

		case "PERFORM_INGAME_ACTION":
			// Define target module and function for the SUI Move call
			targetModule := "player_actions"
			targetFunction := "execute_game_action"

			// Extract action details from payload.
			// Expecting "action_name": string and "action_params": map[string]interface{} in actionPayload.Data
			actionName, okActionName := actionPayload.Data["action_name"].(string)
			actionParams, okActionParams := actionPayload.Data["action_params"].(map[string]interface{})

			if !okActionName || !okActionParams {
				utils.LogWarnf("[%s] Player %s: PERFORM_INGAME_ACTION payload malformed. Expected 'action_name' (string) and 'action_params' (map). Payload: %+v",
					actorID, a.playerID, actionPayload.Data)
				a.sendResponse(protocol.MsgTypePlayerActionResponse, protocol.PlayerActionResponsePayload{
					ActionType: actionPayload.ActionType,
					Status:     "INVALID_ACTION_DATA",
					Message:    "Action data is malformed. Expected 'action_name' and 'action_params'.",
				})
				return
			}

			utils.LogInfof("[%s] Player %s: Action %s. Preparing simulated SUI MoveCall for action: %s with params: %+v",
				actorID, a.playerID, actionPayload.ActionType, actionName, actionParams)

			// Construct arguments for the SUI Move call based on actionName and actionParams.
			// This is highly dependent on the SUI contract's function signatures.
			// For simulation, we'll create a generic list of arguments.
			// Example: First arg is player ID, second is action name string, third could be serialized params or specific object IDs.
			suiCallArgs := []interface{}{
				// In a real scenario, this might be the player's SUI address or a player capability object ID
				// For simulation, using the game playerID string.
				a.playerID, // This would likely be an ObjectID or address on SUI
				actionName, // The specific action being performed
				// More arguments could be derived from actionParams, e.g., target object IDs, amounts, etc.
				// For example, if params included "target_object_id": "0x...", it would be added here.
				// For now, sending the whole map as a string for simplicity in simulation, though not ideal for real contract calls.
				fmt.Sprintf("%v", actionParams), // Simplistic representation of params
			}
			typeArgs := []string{} // Example: If the Move function has type arguments like T, U...

			// Simulate gas details
			gasObjectID := "0xSIMULATED_GAS_COIN_ID" // Placeholder
			gasBudget := uint64(10000000)            // Example

			utils.LogInfof(
				"[%s] Player %s: SIMULATING SUI MoveCall: PackageID=%s, Module=%s, Function=%s, TypeArgs=%v, Args=%v, GasObj=%s, GasBudget=%d",
				actorID, a.playerID, placeholderGameLogicPackageID, targetModule, targetFunction, typeArgs, suiCallArgs, gasObjectID, gasBudget,
			)

			// Simulate the response from suiClient.MoveCall (which prepares the transaction bytes)
			mockTxBytes := fmt.Sprintf("SIMULATED_TX_BYTES_FOR_%s_ACTION_%s", a.playerID, actionName)
			simulatedMoveCallResponse := models.TxnMetaData{
				TxBytes: mockTxBytes,
				// Other fields like GasUsed, Effects, etc., would be populated after execution.
				// For a `SuiMoveCall` (dry run or build-only), TxBytes is the primary output.
			}
			utils.LogInfof("[%s] Player %s: Simulated SuiMoveCall successful. Received TxBytes: %s",
				actorID, a.playerID, simulatedMoveCallResponse.TxBytes)

			// Log next conceptual steps: signing and execution
			utils.LogInfof("[%s] Player %s: Next conceptual step: Signing TxBytes using server's key (if applicable).",
				actorID, a.playerID)
			// In a real scenario, serverPrivateKeyHex would come from a secure config.
			// For this simulation, we'll just log that it *would* be used.
			// serverPrivateKeyHex := cfg.Sui.PrivateKey // PlayerSessionActor doesn't have cfg.
			// conceptualSignature, signErr := sui.SignTransactionBytesWithServerKey(simulatedMoveCallResponse.TxBytes, serverPrivateKeyHex)
			// if signErr != nil {
			// 	 utils.LogErrorf("[%s] Player %s: Conceptual signing failed: %v", actorID, a.playerID, signErr)
			// } else {
			//	 utils.LogInfof("[%s] Player %s: Conceptual server-side signature obtained: %s", actorID, a.playerID, conceptualSignature)
			// }
			utils.LogInfo("PlayerSessionActor: (Conceptual) Call to sui.SignTransactionBytesWithServerKey would happen here if server needs to sign.")

			utils.LogInfof("[%s] Player %s: Final conceptual step: ExecuteTransactionBlock with TxBytes and signature(s).",
				actorID, a.playerID)

			a.sendResponse(protocol.MsgTypePlayerActionResponse, protocol.PlayerActionResponsePayload{
				ActionType: actionPayload.ActionType,
				Status:     "SIMULATED_SUI_MOVE_CALL_PREPARED",
				Message:    "In-game action prepared for SUI execution (simulated).",
				// Optionally, could return TxBytes or a transaction digest if the simulation went further
			})
		default:
			utils.LogWarnf("[%s] Player %s: Received unknown PLAYER_ACTION type: %s", actorID, a.playerID, actionPayload.ActionType)
			a.sendResponse(protocol.MsgTypePlayerActionResponse, protocol.PlayerActionResponsePayload{
				ActionType: actionPayload.ActionType,
				Status:     "UNKNOWN_ACTION_TYPE",
				Message:    "Server does not understand this player action type.",
			})
		}

	default:
		utils.LogWarnf("[%s] Player %s: Received unhandled message type '%s'", actorID, a.playerID, msg.Type)
		a.sendErrorResponse("UNKNOWN_COMMAND", fmt.Sprintf("Unknown command type: %s", msg.Type))
	}

}

// handleForwardToClient sends a message payload to the connected client.
func (a *PlayerSessionActor) handleForwardToClient(msg *messages.ForwardToClient) {
	if a.conn == nil {
		utils.LogWarnf("PlayerSessionActor %s: No connection available to forward message.", a.playerID)
		return
	}

	payload := msg.Payload
	payloadLen := len(payload)

	buffer := make([]byte, 4+payloadLen)

	binary.BigEndian.PutUint32(buffer[0:4], uint32(payloadLen))

	copy(buffer[4:], payload)

	if _, err := a.conn.Write(buffer); err != nil {
		utils.LogErrorf("PlayerSessionActor %s: Error writing to client %s: %v", a.playerID, a.conn.RemoteAddr(), err)
	} else {
		utils.LogDebugf("PlayerSessionActor %s: Sent %d bytes (len) + %d bytes (payload) to client %s.", a.playerID, 4, payloadLen, a.conn.RemoteAddr())
	}
}

// sendResponse constructs and sends a standard JSON message to the client.
func (a *PlayerSessionActor) sendResponse(msgType string, payload interface{}) {
	response := protocol.ClientServerMessage{
		Type:    msgType,
		Payload: payload,
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		utils.LogErrorf("PlayerSessionActor %s: Error marshaling response type %s: %v", a.playerID, msgType, err)
		errorPayload := protocol.ErrorResponsePayload{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Error creating response: " + err.Error(),
		}
		fallbackResponse := protocol.ClientServerMessage{Type: protocol.MsgTypeError, Payload: errorPayload}
		jsonFallback, _ := json.Marshal(fallbackResponse)
		a.handleForwardToClient(&messages.ForwardToClient{Payload: jsonFallback})
		return
	}
	a.handleForwardToClient(&messages.ForwardToClient{Payload: jsonResponse})
}

// sendErrorResponse sends a structured error message to the client.
func (a *PlayerSessionActor) sendErrorResponse(errCode string, errMsg string) {
	errorPayload := protocol.ErrorResponsePayload{
		Code:    errCode,
		Message: errMsg,
	}
	a.sendResponse(protocol.MsgTypeError, errorPayload)
}

// sendSimpleMessage sends a simple text message to the client, wrapped in standard JSON structure.
func (a *PlayerSessionActor) sendSimpleMessage(message string) {
	payload := protocol.SimpleMessagePayload{
		Message: message,
	}
	a.sendResponse(protocol.MsgTypeSimpleMessage, payload)
}

func (a *PlayerSessionActor) isAuthenticated() bool {
	return a.playerID != ""
}
