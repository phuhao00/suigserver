@echo off
echo Building Sui Game Server...

REM Create bin directory if it doesn't exist
if not exist "bin" mkdir bin

echo.
echo Building Simple Server (no external dependencies)...
go build -o bin/simple-server.exe server/cmd/simple/main.go
if %ERRORLEVEL% EQU 0 (
    echo ✓ Simple server built successfully: bin/simple-server.exe
) else (
    echo ✗ Failed to build simple server
    exit /b 1
)

echo.
echo Building Actor-based Server (requires dependencies)...
REM Check if we can build the actor version
go build -o bin/game-server.exe server/cmd/game/main.go 2>nul
if %ERRORLEVEL% EQU 0 (
    echo ✓ Actor-based server built successfully: bin/game-server.exe
) else (
    echo ⚠ Actor-based server build failed (missing dependencies)
    echo   To build the actor version:
    echo   1. Ensure internet connection
    echo   2. Run: copy go.mod.actor go.mod
    echo   3. Run: go mod tidy
    echo   4. Run: go build -o bin/game-server.exe server/cmd/game/main.go
)

echo.
echo Build Summary:
echo ✓ Simple Server: bin/simple-server.exe (ready to use)
if exist "bin/game-server.exe" (
    echo ✓ Actor Server: bin/game-server.exe (ready to use)
) else (
    echo ⚠ Actor Server: Not built (dependencies required)
)

echo.
echo Usage:
echo   Simple Server: bin/simple-server.exe -port 8080
echo   Actor Server:  bin/game-server.exe
echo.
echo Connect with: telnet localhost 8080
