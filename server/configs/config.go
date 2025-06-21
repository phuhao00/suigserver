package configs

import (
	"encoding/json"
	"log"
	"os"
	"sync"
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
		// Contract package IDs would go here after deployment
		// Example: PlayerNFTPackageID string `json:"playerNftPackageId"`
		// ItemNFTPackageID   string `json:"itemNftPackageId"`
		// GuildPackageID     string `json:"guildPackageId"`
	} `json:"sui"`
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
		log.Printf("Loading configuration from %s\n", filePath)
		file, fileErr := os.ReadFile(filePath)
		if fileErr != nil {
			err = fileErr
			log.Printf("Error reading config file %s: %v\n", filePath, err)
			return
		}

		cfg := &Config{}
		// Set default values before unmarshalling
		setDefaultValues(cfg)

		jsonErr := json.Unmarshal(file, cfg)
		if jsonErr != nil {
			err = jsonErr
			log.Printf("Error unmarshalling config file %s: %v\n", filePath, err)
			return
		}
		config = cfg
		log.Println("Configuration loaded successfully.")
	})
	return config, err
}

// GetConfig returns the loaded configuration.
// It will panic if LoadConfig has not been called successfully.
func GetConfig() *Config {
	if config == nil || err != nil {
		log.Panicln("Configuration not loaded or loaded with error. Call LoadConfig first.", err)
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
}

// CreateExampleConfigFile creates an example config.json if it doesn't exist.
func CreateExampleConfigFile(filePath string) {
	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		log.Printf("Creating example config file at %s\n", filePath)
		exampleCfg := &Config{}
		setDefaultValues(exampleCfg) // Populate with defaults

		// Add some placeholder values for things that need user input
		exampleCfg.Database.PostgresURL = "postgresql://user:password@localhost:5432/mmo_db?sslmode=disable"
		exampleCfg.Redis.Address = "localhost:6379"
		exampleCfg.Sui.PrivateKey = "YOUR_SUI_PRIVATE_KEY_HEX_HERE" // Emphasize this needs to be set

		data, marshalErr := json.MarshalIndent(exampleCfg, "", "  ")
		if marshalErr != nil {
			log.Printf("Error marshalling example config: %v\n", marshalErr)
			return
		}

		if writeErr := os.WriteFile(filePath, data, 0644); writeErr != nil {
			log.Printf("Error writing example config file %s: %v\n", filePath, writeErr)
		} else {
			log.Printf("Example config file created: %s. Please review and update it.\n", filePath)
		}
	} else {
		log.Printf("Config file %s already exists. Skipping creation of example.\n", filePath)
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
