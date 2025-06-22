# Sui Game Server

A multiplayer game server implementation for the Sui blockchain ecosystem.

## Project Structure

```
suigserver/
├── go.mod                          # Go module definition
├── server/
│   ├── cmd/
│   │   ├── game/main.go           # Full actor-based server (requires external deps)
│   │   └── simple/main.go         # Simple server (no external deps)
│   ├── configs/                   # Configuration management
│   ├── internal/
│   │   ├── actor/                # Actor system implementation
│   │   │   ├── messages/         # Message definitions
│   │   │   ├── room_actor.go     # Room management
│   │   │   ├── room_manager_actor.go
│   │   │   ├── session_actor.go  # Player session handling
│   │   │   └── world_manager_actor.go
│   │   ├── game/                 # Game logic
│   │   ├── model/                # Data models
│   │   ├── network/              # Network layer
│   │   ├── simple/               # Simple server implementation
│   │   └── sui/                  # Sui blockchain integration
├── contracts/                     # Move smart contracts
│   ├── combat_system/
│   ├── game_world/
│   ├── guild_system/
│   ├── items_system/
│   ├── marketplace_system/
│   └── player_system/
└── docs/                         # Documentation
```

## Features

### Current Features
- **Simple Server**: Basic TCP server with room-based chat
- **Actor-based Server**: Advanced server using actor model (requires dependencies)
- **Smart Contracts**: Sui Move contracts for game systems
- **Player Management**: Basic authentication and session handling
- **Room System**: Join/leave rooms and chat functionality
- **Configuration Management**: JSON-based configuration

### Planned Features
- Sui blockchain integration
- NFT-based player items
- Combat system
- Guild system
- Marketplace
- Player progression

## Quick Start

### Option 1: Simple Server (Recommended for testing)

The simple server requires no external dependencies and can be built immediately:

```bash
# Build the simple server
go build -o bin/simple-server.exe server/cmd/simple/main.go

# Run the server
./bin/simple-server.exe -port 8080
```

### Option 2: Full Actor-based Server

The full server uses the Proto.Actor framework but requires external dependencies:

```bash
# Install dependencies (requires internet connection)
go mod tidy

# Build the full server
go build -o bin/game-server.exe server/cmd/game/main.go

# Run the server
./bin/game-server.exe
```

## Configuration

### Simple Server
The simple server uses command-line flags:
- `-port`: Port to run the server on (default: 8080)

### Full Server
The full server uses a JSON configuration file. An example config will be created automatically as `config.json`.

## Client Commands

Connect to the server using telnet or any TCP client:

```bash
telnet localhost 8080
```

Available commands:
- `/auth <name>` - Authenticate with a player name
- `/join <room>` - Join a room (creates if doesn't exist)
- `/say <message>` - Send a message to all players in the current room
- `/quit` - Disconnect from the server

## Example Usage

1. Start the server:
   ```bash
   ./bin/simple-server.exe -port 8080
   ```

2. Connect two clients:
   ```bash
   # Terminal 1
   telnet localhost 8080
   /auth Alice
   /join gameroom
   /say Hello everyone!
   
   # Terminal 2  
   telnet localhost 8080
   /auth Bob
   /join gameroom
   /say Hi Alice!
   ```

## Smart Contracts

Build the smart contracts:

```bash
# Combat System
cd "c:\file\sui_project\gamessever\suigserver\contracts\combat_system"
sui move build --skip-fetch-latest-git-deps
```

## Development

### Building Smart Contracts

Each contract directory has its own build scripts:

```bash
# For Windows
cd contracts/player_system
.\build.bat

# For Linux/Mac
cd contracts/player_system
./build.sh
```

### Adding New Features

1. **Simple Server**: Modify `server/internal/simple/server.go`
2. **Full Server**: Add new actors in `server/internal/actor/`
3. **Smart Contracts**: Add new Move contracts in `contracts/`

## Architecture

### Simple Server
- Single-threaded with goroutines for each client
- In-memory room and player management
- Basic command parsing and message routing

### Full Server (Actor Model)
- `PlayerSessionActor`: Manages individual player connections
- `RoomActor`: Manages game rooms and player interactions
- `RoomManagerActor`: Coordinates room creation and discovery
- `WorldManagerActor`: Handles global game state

### Smart Contracts
- **Player System**: Player NFTs and character data
- **Item System**: Game items and equipment
- **Combat System**: Battle mechanics and results
- **Guild System**: Player organizations
- **Marketplace**: Trading system for game assets

## Troubleshooting

### Network Issues
If you get connection errors when building the full server:
1. Use the simple server instead
2. Check your internet connection
3. Try using Go proxy: `go env -w GOPROXY=direct`

### Port Already in Use
If the port is already in use:
1. Change the port: `./simple-server.exe -port 8081`
2. Or stop the conflicting process

### Build Errors
1. Ensure Go 1.21 or later is installed
2. Run `go mod tidy` to fix module issues
3. Check that all import paths are correct

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test with both simple and full servers
5. Submit a pull request

## License

See LICENSE file for details.