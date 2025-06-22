package configs

import (
	"encoding/json"
	"log" // Standard log for initial messages before custom logger is configured
	"os"
	"sync"
	"github.com/phuhao00/suigserver/server/internal/utils" // Logger
)

// Config holds the application configuration.
type Config struct {
	Server struct {
		Host    string `json:"host"`
		TCPPort int    `json:"tcpPort"`
		HTTPPort int   `json:"httpPort"` // For potential admin/metrics endpoints
		LogLevel string `json:"logLevel"`
	} `json:"server"`
	Database struct {
		PostgresURL string `json:"postgresUrl"`
	} `json:"database"`
	Redis struct {
		Address  string `json:"address"`
		Password string `json:"password"`
		DB       int    `json:"db"`
	} `json:"redis"`
	Sui struct {
		RPCURL         string `json:"rpcUrl"`
		WebsocketURL   string `json:"websocketUrl"` // For event subscriptions
		PrivateKey     string `json:"privateKey"`   // Server's private key for transactions (handle with care!)
		GasBudget      uint64 `json:"gasBudget"`
		// Placeholder Contract package IDs - replace with actual IDs after deployment
		GameLogicPackageID      string `json:"gameLogicPackageId"`
		PlayerRegistryPackageID string `json:"playerRegistryPackageId"`
		ItemSystemPackageID     string `json:"itemSystemPackageId"`
		PlayerObjectPackageID   string `json:"playerObjectPackageId"` // For player profile/data objects
		PlayerObjectModule      string `json:"playerObjectModule"`    // Module name for player profile/data
	} `json:"sui"`
	Auth struct {
		DummyToken      string `json:"dummyToken"`
		DummyPlayerID   string `json:"dummyPlayerId"`
		EnableDummyAuth bool   `json:"enableDummyAuth"` // To easily switch it off
	} `json:"auth"`
	// Potentially add other sections like JWT secrets, external API keys, etc.
}

var (
	once   sync.Once
	config *Config
	err    error
)

// LoadConfig loads the configuration from a file (e.g., config.json).
// It's designed to be called once.
func LoadConfig(filePath string) (*Config, error) {
	once.Do(func() {
		// Use standard log here initially, as our logger's level isn't set yet.
		// Or, accept that these initial logs might not be filtered by level if we used utils.Log directly.
		log.Printf("Loading configuration from %s", filePath) // Standard log
		file, fileErr := os.ReadFile(filePath)
		if fileErr != nil {
			err = fileErr
			log.Printf("Error reading config file %s: %v", filePath, err) // Standard log
			return
		}

		cfg := &Config{}
		// Set default values before unmarshalling
		setDefaultValues(cfg)

		jsonErr := json.Unmarshal(file, cfg)
		if jsonErr != nil {
			err = jsonErr
			log.Printf("Error unmarshalling config file %s: %v", filePath, err) // Standard log
			return
		}
		config = cfg
		log.Println("Configuration loaded successfully.") // Standard log
	})
	return config, err
}

// GetConfig returns the loaded configuration.
// It will panic if LoadConfig has not been called successfully.
func GetConfig() *Config {
	if config == nil || err != nil {
		// Use utils.LogFatalf if available and configured, otherwise standard log.Panicln
		// For simplicity, assuming by the time GetConfig is widely used, logger is set.
		// However, direct calls to log.Panicln are safer if logger setup could fail.
		utils.LogFatalf("Configuration not loaded or loaded with error. Call LoadConfig first. Error: %v", err)
	}
	return config
}

// setDefaultValues sets default configuration values.
func setDefaultValues(cfg *Config) {
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.TCPPort = 8080
	cfg.Server.HTTPPort = 8081
	cfg.Server.LogLevel = "INFO"
	cfg.Sui.GasBudget = 100000000 // Default gas budget (adjust as needed)
	cfg.Sui.RPCURL = "https://fullnode.testnet.sui.io:443" // Default to Sui Testnet
	cfg.Sui.GameLogicPackageID = "0xYOUR_GAME_LOGIC_PACKAGE_ID_HERE"
	cfg.Sui.PlayerRegistryPackageID = "0xYOUR_PLAYER_REGISTRY_PACKAGE_ID_HERE"
	cfg.Sui.ItemSystemPackageID = "0xYOUR_ITEM_SYSTEM_PACKAGE_ID_HERE"
	cfg.Sui.PlayerObjectPackageID = "0xYOUR_PLAYER_OBJECT_PACKAGE_ID_HERE"
	cfg.Sui.PlayerObjectModule = "player_profile" // Example default module name
	// Auth defaults
	cfg.Auth.EnableDummyAuth = true
	cfg.Auth.DummyToken = "fixed_dummy_secret_token_123"
	cfg.Auth.DummyPlayerID = "player_associated_with_dummy_token"
}

// CreateExampleConfigFile creates an example config.json if it doesn't exist.
func CreateExampleConfigFile(filePath string) {
	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		// Using standard log here as well, as this can be called before logger is configured.
		log.Printf("Creating example config file at %s", filePath)
		exampleCfg := &Config{}
		setDefaultValues(exampleCfg) // Populate with defaults

		// Add some placeholder values for things that need user input
		exampleCfg.Database.PostgresURL = "postgresql://user:password@localhost:5432/mmo_db?sslmode=disable"
		exampleCfg.Redis.Address = "localhost:6379"
		exampleCfg.Sui.PrivateKey = "YOUR_SUI_PRIVATE_KEY_HEX_HERE" // Emphasize this needs to be set

		data, marshalErr := json.MarshalIndent(exampleCfg, "", "  ")
		if marshalErr != nil {
			log.Printf("Error marshalling example config: %v", marshalErr)
			return
		}

		if writeErr := os.WriteFile(filePath, data, 0644); writeErr != nil {
			log.Printf("Error writing example config file %s: %v", filePath, writeErr)
		} else {
			log.Printf("Example config file created: %s. Please review and update it.", filePath)
		}
	} else {
		log.Printf("Config file %s already exists. Skipping creation of example.", filePath)
	}
}

/*
Usage in main.go:

import "sui-mmo-server/server/configs"

func main() {
    // Create an example config if it doesn't exist, in the same dir as the executable or a specified path
    configs.CreateExampleConfigFile("config.json")

    // Load the configuration
    cfg, err := configs.LoadConfig("config.json")
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    log.Printf("Server starting on port: %d", cfg.Server.TCPPort)
    // Use cfg.Database.PostgresURL, cfg.Sui.RPCURL etc. throughout the application
    // The GetConfig() function can be used elsewhere to access the loaded config singleton.
}

*/
