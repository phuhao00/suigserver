@echo off
echo Building Marketplace System...
sui move build
if %errorlevel% neq 0 (
    echo Build failed!
    exit /b 1
)

echo.
echo Running tests...
sui move test
if %errorlevel% neq 0 (
    echo Tests failed!
    exit /b 1
)

echo.
echo ✅ Build and tests completed successfully!
echo.
echo To deploy to testnet:
echo sui client publish --gas-budget 50000000
echo.
echo To deploy to mainnet:
echo sui client publish --gas-budget 50000000 --network mainnet
