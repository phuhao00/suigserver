#!/bin/bash
echo "Building Sui Game Server..."

# Create bin directory if it doesn't exist
mkdir -p bin

echo
echo "Building Simple Server (no external dependencies)..."
go build -o bin/simple-server server/cmd/simple/main.go
if [ $? -eq 0 ]; then
    echo "✓ Simple server built successfully: bin/simple-server"
else
    echo "✗ Failed to build simple server"
    exit 1
fi

echo
echo "Building Actor-based Server (requires dependencies)..."
# Check if we can build the actor version
go build -o bin/game-server server/cmd/game/main.go 2>/dev/null
if [ $? -eq 0 ]; then
    echo "✓ Actor-based server built successfully: bin/game-server"
else
    echo "⚠ Actor-based server build failed (missing dependencies)"
    echo "  To build the actor version:"
    echo "  1. Ensure internet connection"
    echo "  2. Run: cp go.mod.actor go.mod"
    echo "  3. Run: go mod tidy"
    echo "  4. Run: go build -o bin/game-server server/cmd/game/main.go"
fi

echo
echo "Build Summary:"
echo "✓ Simple Server: bin/simple-server (ready to use)"
if [ -f "bin/game-server" ]; then
    echo "✓ Actor Server: bin/game-server (ready to use)"
else
    echo "⚠ Actor Server: Not built (dependencies required)"
fi

echo
echo "Usage:"
echo "  Simple Server: ./bin/simple-server -port 8080"
echo "  Actor Server:  ./bin/game-server"
echo
echo "Connect with: telnet localhost 8080"
