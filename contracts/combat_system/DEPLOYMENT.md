# Combat System Deployment Guide

This guide provides step-by-step instructions for deploying the Combat System smart contract to the Sui blockchain.

## Prerequisites

1. **Sui CLI Installation**
   ```bash
   # Install Sui CLI (visit https://docs.sui.io/guides/developer/getting-started/sui-install for latest instructions)
   cargo install --locked --git https://github.com/MystenLabs/sui.git --branch devnet sui
   ```

2. **Wallet Setup**
   ```bash
   # Create a new wallet or import existing one
   sui client new-address ed25519
   
   # Check your address
   sui client active-address
   
   # Get testnet SUI tokens (for testing)
   curl --location --request POST 'https://faucet.devnet.sui.io/gas' \
   --header 'Content-Type: application/json' \
   --data-raw '{
       "FixedAmountRequest": {
           "recipient": "YOUR_ADDRESS_HERE"
       }
   }'
   ```

3. **Network Configuration**
   ```bash
   # Switch to devnet (for testing)
   sui client switch --env devnet
   
   # Or switch to testnet
   sui client switch --env testnet
   
   # For mainnet deployment
   sui client switch --env mainnet
   ```

## Build and Test

1. **Navigate to Contract Directory**
   ```bash
   cd contracts/combat_system
   ```

2. **Build the Contract**
   ```bash
   # For Unix/Linux/macOS
   ./build.sh
   
   # For Windows
   build.bat
   
   # Or manually
   sui move build
   ```

3. **Run Tests**
   ```bash
   sui move test
   ```

## Deployment Steps

### 1. Deploy to Devnet (Testing)

```bash
# Build and publish the contract
sui client publish --gas-budget 20000000
```

This will output:
- Package ID: The unique identifier for your deployed contract
- Object IDs: For created objects like AdminCap and CombatRegistry

### 2. Deploy to Testnet

```bash
# Switch to testnet
sui client switch --env testnet

# Ensure you have testnet SUI tokens
sui client gas

# Deploy
sui client publish --gas-budget 20000000
```

### 3. Deploy to Mainnet (Production)

⚠️ **Important**: Only deploy to mainnet after thorough testing on devnet and testnet.

```bash
# Switch to mainnet
sui client switch --env mainnet

# Ensure you have mainnet SUI tokens
sui client gas

# Deploy with higher gas budget for mainnet
sui client publish --gas-budget 30000000
```

## Post-Deployment Setup

### 1. Save Important Information

After deployment, save these critical details:

```bash
# Package ID (the deployed contract address)
export PACKAGE_ID="0x..."

# AdminCap Object ID (for administrative functions)
export ADMIN_CAP_ID="0x..."

# CombatRegistry Object ID (shared object for global state)
export COMBAT_REGISTRY_ID="0x..."
```

### 2. Verify Deployment

```bash
# Check the deployed package
sui client object $PACKAGE_ID

# Check AdminCap ownership
sui client object $ADMIN_CAP_ID

# Check CombatRegistry (shared object)
sui client object $COMBAT_REGISTRY_ID
```

### 3. Test Basic Functionality

```bash
# Example: Create a test combat session
sui client call \
  --package $PACKAGE_ID \
  --module combat \
  --function initiate_combat \
  --args $ADMIN_CAP_ID 1 1 '["0xparticipant1", "0xparticipant2"]' 1000 '["test combat"]' \
  --gas-budget 5000000
```

## Integration with Game Server

### 1. Update Game Server Configuration

Update your Go game server configuration with the deployed contract details:

```go
// config/config.go
type ContractConfig struct {
    PackageID         string `yaml:"package_id"`
    AdminCapID        string `yaml:"admin_cap_id"`
    CombatRegistryID  string `yaml:"combat_registry_id"`
    NetworkURL        string `yaml:"network_url"`
}
```

### 2. Initialize Sui Client in Game Server

```go
// internal/sui/client.go
func NewSuiClient(config ContractConfig) *SuiClient {
    return &SuiClient{
        PackageID:        config.PackageID,
        AdminCapID:       config.AdminCapID,
        CombatRegistryID: config.CombatRegistryID,
        NetworkURL:       config.NetworkURL,
    }
}
```

## Security Considerations

### 1. AdminCap Management

- **Store AdminCap securely**: The AdminCap object ID should be kept secure as it provides administrative access
- **Use multi-sig for production**: Consider using a multi-signature wallet for AdminCap on mainnet
- **Implement access controls**: Add additional access control layers in your game server

### 2. Private Key Security

- **Hardware wallets**: Use hardware wallets for mainnet deployments
- **Key rotation**: Implement key rotation policies
- **Environment variables**: Store sensitive data in environment variables, not in code

### 3. Gas Management

- **Monitor gas usage**: Track gas consumption for contract interactions
- **Set appropriate limits**: Configure reasonable gas budgets for different operations
- **Optimize transactions**: Batch operations when possible to reduce gas costs

## Monitoring and Maintenance

### 1. Event Monitoring

Set up monitoring for contract events:

```bash
# Subscribe to combat events
sui client subscribe-event \
  --package $PACKAGE_ID \
  --module combat
```

### 2. Health Checks

Implement health checks in your game server:

```go
func (c *SuiClient) HealthCheck() error {
    // Check contract accessibility
    _, err := c.GetCombatRegistry()
    return err
}
```

### 3. Upgrade Planning

- **Plan for upgrades**: Design your system to handle contract upgrades
- **Version management**: Keep track of deployed contract versions
- **Migration strategies**: Plan data migration for contract upgrades

## Troubleshooting

### Common Issues

1. **Gas Budget Too Low**
   ```bash
   # Increase gas budget
   --gas-budget 30000000
   ```

2. **Insufficient Balance**
   ```bash
   # Check balance
   sui client gas
   
   # Request more tokens (testnet/devnet)
   sui client faucet
   ```

3. **Network Issues**
   ```bash
   # Check network connectivity
   sui client show-rpc-url
   
   # Switch networks if needed
   sui client switch --env devnet
   ```

4. **Object Not Found**
   ```bash
   # Verify object exists
   sui client object OBJECT_ID
   
   # Check if object was consumed/transferred
   sui client object OBJECT_ID --verbose
   ```

## Support and Resources

- **Sui Documentation**: https://docs.sui.io/
- **Sui Discord**: https://discord.gg/sui
- **GitHub Issues**: Report bugs and issues in the project repository
- **Community Forums**: Engage with the Sui developer community

## Deployment Checklist

- [ ] Sui CLI installed and configured
- [ ] Wallet created and funded
- [ ] Contract built and tested locally
- [ ] Deployed to devnet and tested
- [ ] Deployed to testnet and tested
- [ ] Game server configuration updated
- [ ] AdminCap secured
- [ ] Monitoring systems in place
- [ ] Documentation updated
- [ ] Team notified of deployment details
