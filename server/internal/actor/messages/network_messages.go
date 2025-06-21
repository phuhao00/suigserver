package messages

import (
	"net"

	"github.com/asynkron/protoactor-go/actor"
)

// ClientConnected is sent to a PlayerSessionActor when a new client connects.
type ClientConnected struct {
	Conn   net.Conn    // The raw network connection.
	SelfPID *actor.PID // PID of the TCPServer's connection handling goroutine/actor, if needed for replies.
}

// ClientDisconnected is sent when a client connection is lost or closed.
type ClientDisconnected struct {
	Reason string
}

// ClientMessage is sent from the network layer to the PlayerSessionActor, containing raw data from the client.
type ClientMessage struct {
	Payload []byte
}

// ForwardToClient is sent from an actor (e.g., PlayerSessionActor) to the network layer (or a specific connection actor)
// to send data to the client.
type ForwardToClient struct {
	Payload []byte
}

// TerminateSession is a message that can be sent to a PlayerSessionActor to instruct it to shut down.
type TerminateSession struct {
	Reason string
}

// SentFromNetwork indicates a message originating from the network layer to an actor.
// It helps actors understand the source and context of the message.
// type SentFromNetwork struct {
// 	ConnectionID string // Unique identifier for the client connection
// 	ActualMessage interface{} // The actual message payload (e.g., a parsed command)
// }

// SendToNetwork is a message an actor sends when it wants to transmit data over the network.
// type SendToNetwork struct {
// 	ConnectionID string // Unique identifier for the client connection
// 	Payload      []byte // Raw data to be sent
// }
