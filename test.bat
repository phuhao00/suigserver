@echo off
echo Testing Sui Game Server...

REM Check if server is built
if not exist "bin\simple-server.exe" (
    echo Error: Server not found. Run build.bat first.
    exit /b 1
)

if not exist "bin\client.exe" (
    echo Building test client...
    go build -o bin/client.exe tools/client/main.go
)

echo.
echo Starting server on port 8082...
start "Game Server" cmd /k "bin\simple-server.exe -port 8082"

REM Wait a moment for server to start
timeout /t 2 /nobreak >nul

echo.
echo Starting test client...
echo Instructions:
echo 1. Type: /auth TestPlayer
echo 2. Type: /join testroom
echo 3. Type: /say Hello World!
echo 4. Type: /quit or 'exit' to quit
echo.

bin\client.exe -port 8082

echo.
echo Test completed. Server is still running in the background.
echo Close the server window when done testing.
