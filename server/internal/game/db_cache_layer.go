package game

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// PlayerData represents the structure of data we're storing for a player.
// This is a placeholder; define actual player data structure as needed.
type PlayerData struct {
	ID          string                 `json:"id"`
	DisplayName string                 `json:"displayName"`
	Level       int                    `json:"level"`
	Experience  int                    `json:"experience"`
	Position    map[string]float64     `json:"position"` // e.g., {"x": 0, "y": 0, "z": 0}
	Inventory   map[string]int         `json:"inventory"`  // ItemID -> Quantity
	Attributes  map[string]interface{} `json:"attributes"` // General purpose attributes
	LastLogin   time.Time              `json:"lastLogin"`
}

// DBCacheLayer provides an abstraction for interacting with the database and caching layer (Redis).
type DBCacheLayer struct {
	db          *sql.DB
	redisClient *redis.Client
	ctx         context.Context // Context for Redis operations
}

// DBConfig holds database connection parameters.
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// NewDBCacheLayer creates a new DBCacheLayer.
// Actual configuration should be passed in, not hardcoded.
func NewDBCacheLayer(dbCfg DBConfig, redisCfg RedisConfig) (*DBCacheLayer, error) {
	log.Println("Initializing DB Cache Layer...")

	// Initialize PostgreSQL connection
	// Example: "host=localhost port=5432 user=user password=password dbname=gamedb sslmode=disable"
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.DBName, dbCfg.SSLMode)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Printf("Error opening PostgreSQL connection: %v", err)
		return nil, fmt.Errorf("sql.Open failed: %w", err)
	}

	// Initialize Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisCfg.Addr,     // e.g., "localhost:6379"
		Password: redisCfg.Password, // no password set if empty
		DB:       redisCfg.DB,       // use default DB
	})

	return &DBCacheLayer{
		db:          db,
		redisClient: rdb,
		ctx:         context.Background(), // Or a more specific context
	}, nil
}

// Start initializes and tests the DB and cache connections.
func (dbcl *DBCacheLayer) Start() error {
	log.Println("Starting DB Cache Layer...")
	// Ping DB
	if err := dbcl.db.Ping(); err != nil {
		log.Printf("Error pinging PostgreSQL: %v", err)
		return fmt.Errorf("db.Ping failed: %w", err)
	}
	log.Println("PostgreSQL connection successful.")

	// Ping Redis
	if _, err := dbcl.redisClient.Ping(dbcl.ctx).Result(); err != nil {
		log.Printf("Error pinging Redis: %v", err)
		return fmt.Errorf("redis.Ping failed: %w", err)
	}
	log.Println("Redis connection successful.")
	log.Println("DB Cache Layer started successfully.")
	return nil
}

// Stop closes database and cache connections.
func (dbcl *DBCacheLayer) Stop() {
	log.Println("Stopping DB Cache Layer...")
	if dbcl.db != nil {
		if err := dbcl.db.Close(); err != nil {
			log.Printf("Error closing PostgreSQL connection: %v", err)
		} else {
			log.Println("PostgreSQL connection closed.")
		}
	}
	if dbcl.redisClient != nil {
		if err := dbcl.redisClient.Close(); err != nil {
			log.Printf("Error closing Redis connection: %v", err)
		} else {
			log.Println("Redis connection closed.")
		}
	}
	log.Println("DB Cache Layer stopped.")
}

// GetPlayerData retrieves player data using a cache-aside strategy.
func (dbcl *DBCacheLayer) GetPlayerData(playerID string) (*PlayerData, error) {
	cacheKey := fmt.Sprintf("player:%s", playerID)

	// 1. Try to fetch from Redis (cache)
	val, err := dbcl.redisClient.Get(dbcl.ctx, cacheKey).Result()
	if err == nil { // Cache hit
		log.Printf("Cache hit for player %s", playerID)
		var playerData PlayerData
		if err := json.Unmarshal([]byte(val), &playerData); err != nil {
			log.Printf("Error unmarshaling player data from Redis for %s: %v", playerID, err)
			// Cache data might be corrupted, proceed to fetch from DB
		} else {
			return &playerData, nil // Successfully retrieved from cache
		}
	} else if err != redis.Nil { // Actual Redis error (not just cache miss)
		log.Printf("Error fetching player data from Redis for %s: %v", playerID, err)
		// Depending on strategy, might still try DB or return error
	} else {
		log.Printf("Cache miss for player %s", playerID)
	}

	// 2. If cache miss or error, fetch from PostgreSQL (DB)
	log.Printf("Fetching player data from DB for %s", playerID)
	// This is a placeholder. Real implementation needs a proper DB schema and query.
	// Example: Assume a table `players` with columns `id` (TEXT PRIMARY KEY) and `data` (JSONB).
	// row := dbcl.db.QueryRow("SELECT data FROM players WHERE id = $1", playerID)
	// var jsonData []byte
	// if err := row.Scan(&jsonData); err != nil {
	// 	if err == sql.ErrNoRows {
	// 		log.Printf("Player %s not found in DB.", playerID)
	// 		return nil, fmt.Errorf("player not found: %s", playerID) // Or return nil, nil if not finding is not an error
	// 	}
	// 	log.Printf("Error scanning player data from DB for %s: %v", playerID, err)
	// 	return nil, fmt.Errorf("db scan failed for %s: %w", playerID, err)
	// }
	//
	// var playerData PlayerData
	// if err := json.Unmarshal(jsonData, &playerData); err != nil {
	// 	log.Printf("Error unmarshaling player data from DB for %s: %v", playerID, err)
	// 	return nil, fmt.Errorf("db data unmarshal failed for %s: %w", playerID, err)
	// }

	// Placeholder: Simulate DB fetch
	if playerID == "player123" { // Simulate finding a player
		playerData := PlayerData{
			ID:          playerID,
			DisplayName: "MockPlayer",
			Level:       10,
			Experience:  1000,
			LastLogin:   time.Now(),
			Position:    map[string]float64{"x": 10, "y": 5},
			Inventory:   map[string]int{"sword": 1, "potion": 5},
		}
		jsonData, _ := json.Marshal(playerData) // Error handling omitted for brevity

		// 3. Store the fetched data back into Redis for future requests
		// Use a reasonable expiration time, e.g., 1 hour
		err = dbcl.redisClient.Set(dbcl.ctx, cacheKey, jsonData, 1*time.Hour).Err()
		if err != nil {
			log.Printf("Error setting player data to Redis for %s: %v", playerID, err)
			// Non-critical error, data was still fetched from DB
		} else {
			log.Printf("Player data for %s cached in Redis.", playerID)
		}
		return &playerData, nil
	}

	log.Printf("Player %s not found (simulated DB miss).", playerID)
	return nil, fmt.Errorf("player %s not found", playerID) // Or a specific error like ErrPlayerNotFound
}

// SavePlayerData saves player data to the DB and updates/invalidates the cache.
func (dbcl *DBCacheLayer) SavePlayerData(playerID string, data *PlayerData) error {
	if data == nil {
		return fmt.Errorf("cannot save nil player data for %s", playerID)
	}
	log.Printf("Saving player data for %s to DB...", playerID)
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling player data for %s: %v", playerID, err)
		return fmt.Errorf("marshal player data failed for %s: %w", playerID, err)
	}

	// 1. Save to PostgreSQL (DB)
	// Placeholder: Assume an UPSERT operation.
	// Example: "INSERT INTO players (id, data) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET data = $2"
	// _, err = dbcl.db.Exec("YOUR_UPSERT_QUERY_HERE", playerID, jsonData)
	// if err != nil {
	// 	log.Printf("Error saving player data to DB for %s: %v", playerID, err)
	// 	return fmt.Errorf("db save failed for %s: %w", playerID, err)
	// }
	log.Printf("Simulated: Player data for %s saved to DB.", playerID)

	// 2. Update or Invalidate Redis cache
	cacheKey := fmt.Sprintf("player:%s", playerID)
	// Option A: Update cache with new data
	err = dbcl.redisClient.Set(dbcl.ctx, cacheKey, jsonData, 1*time.Hour).Err()
	if err != nil {
		log.Printf("Error updating player data in Redis for %s: %v", playerID, err)
		// This could be a critical error if strong cache consistency is needed,
		// or just logged if eventual consistency is acceptable.
	} else {
		log.Printf("Player data for %s updated in Redis cache.", playerID)
	}
	// Option B: Invalidate cache (delete the key)
	// if err := dbcl.redisClient.Del(dbcl.ctx, cacheKey).Err(); err != nil {
	//     log.Printf("Error deleting player data from Redis for %s: %v", playerID, err)
	// } else {
	//     log.Printf("Player data for %s invalidated in Redis cache.", playerID)
	// }

	return nil
}
