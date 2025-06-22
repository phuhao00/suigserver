#!/bin/bash

# Combat System Build Script
# This script builds and tests the combat system smart contract

echo "Building Combat System Smart Contract..."

# Check if sui is installed
if ! command -v sui &> /dev/null; then
    echo "Error: Sui CLI is not installed or not in PATH"
    echo "Please install Sui CLI from: https://docs.sui.io/guides/developer/getting-started/sui-install"
    exit 1
fi

# Navigate to the contract directory
cd "$(dirname "$0")"

echo "Current directory: $(pwd)"
echo "Contents:"
ls -la

# Try to build the contract
echo "Attempting to build the contract..."
if sui move build; then
    echo "✅ Combat system built successfully!"
    
    # Run tests if build succeeded
    echo "Running tests..."
    if sui move test; then
        echo "✅ All tests passed!"
    else
        echo "❌ Some tests failed"
        exit 1
    fi
else
    echo "❌ Build failed"
    echo "This might be due to network issues downloading dependencies."
    echo "Please ensure you have a stable internet connection and try again."
    exit 1
fi

echo "🎉 Combat system is ready for deployment!"
