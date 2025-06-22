@echo off
REM Combat System Build Script for Windows
REM This script builds and tests the combat system smart contract

echo Building Combat System Smart Contract...

REM Check if sui is installed
where sui >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Error: Sui CLI is not installed or not in PATH
    echo Please install Sui CLI from: https://docs.sui.io/guides/developer/getting-started/sui-install
    exit /b 1
)

REM Navigate to the contract directory
cd /d "%~dp0"

echo Current directory: %CD%
echo Contents:
dir

REM Try to build the contract
echo Attempting to build the contract...
sui move build
if %ERRORLEVEL% EQU 0 (
    echo ✅ Combat system built successfully!
    
    REM Run tests if build succeeded
    echo Running tests...
    sui move test
    if %ERRORLEVEL% EQU 0 (
        echo ✅ All tests passed!
    ) else (
        echo ❌ Some tests failed
        exit /b 1
    )
) else (
    echo ❌ Build failed
    echo This might be due to network issues downloading dependencies.
    echo Please ensure you have a stable internet connection and try again.
    exit /b 1
)

echo 🎉 Combat system is ready for deployment!
