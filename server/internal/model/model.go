package model

import "time"

// Player represents a user account in the game.
type Player struct {
	ID            string    `json:"id"` // Could be a UUID or from an auth provider
	SuiAddress    string    `json:"suiAddress"` // Player's Sui wallet address
	Username      string    `json:"username"`
	Email         string    `json:"email"` // Optional, depending on auth
	CreatedAt     time.Time `json:"createdAt"`
	LastLogin     time.Time `json:"lastLogin"`
	CharacterIDs []string  `json:"characterIds"` // IDs of characters owned by this player
}

// Character represents an in-game character controlled by a player.
// Some character data might be mirrored or primarily live on the Player NFT.
type Character struct {
	ID          string    `json:"id"`          // Unique character ID
	PlayerID    string    `json:"playerId"`    // Owning player's ID
	NFTID       string    `json:"nftId"`       // Corresponding Player NFT ID on Sui
	Name        string    `json:"name"`
	Level       int       `json:"level"`
	Experience  int64     `json:"experience"`
	Health      int       `json:"health"`
	Mana        int       `json:"mana"`
	Class       string    `json:"class"` // e.g., Warrior, Mage
	CreatedAt   time.Time `json:"createdAt"`
	// Position    Vector3   `json:"position"` // Example: if tracking off-chain
	// Inventory   []string  `json:"inventory"`  // List of Item IDs or Item NFT IDs
	// Equipment   map[string]string `json:"equipment"` // Slot -> Item ID/NFT ID
}

// Item represents a generic game item.
// This could be an off-chain representation if the item is not an NFT,
// or a cached representation of an Item NFT.
type Item struct {
	ID          string    `json:"id"`          // Unique item ID (could be NFT ID for on-chain items)
	ItemTypeID  string    `json:"itemTypeId"`  // References a definition of the item (e.g., "healing_potion_small")
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsNFT       bool      `json:"isNft"`       // True if this item is represented by an NFT on Sui
	NFTObjectID string    `json:"nftObjectId"` // Sui Object ID if it's an NFT
	Quantity    int       `json:"quantity"`    // For stackable items
	Metadata    map[string]interface{} `json:"metadata"` // Custom attributes
}

// Equipment represents an item that can be equipped by a character.
// Often, these will be NFTs.
type Equipment struct {
	Item            // Embeds Item struct
	Slot         string `json:"slot"`         // e.g., "weapon", "helmet", "chest"
	AttackBonus  int    `json:"attackBonus"`
	DefenseBonus int    `json:"defenseBonus"`
	// Other stat bonuses
}

// Guild represents a player-formed guild or clan.
// Some guild data might be mirrored or primarily live on a Guild NFT/object on Sui.
type Guild struct {
	ID          string    `json:"id"`          // Unique guild ID (could be NFT ID for on-chain guilds)
	SuiObjectID string    `json:"suiObjectId"` // Sui Object ID if the guild is an on-chain entity
	Name        string    `json:"name"`
	LeaderID    string    `json:"leaderId"`    // Player ID of the guild leader
	MemberIDs   []string  `json:"memberIds"`   // List of Player IDs
	CreatedAt   time.Time `json:"createdAt"`
	MOTD        string    `json:"motd"`        // Message of the day
}

// Room represents a game instance or zone.
type Room struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	MaxPlayers  int      `json:"maxPlayers"`
	PlayerIDs   []string `json:"playerIds"` // Players currently in this room
	MapID       string   `json:"mapId"`     // Identifier for the map/scene of this room
	IsPersistent bool     `json:"isPersistent"` // Whether this room stays active even if empty
}

// Session represents an active player connection to the server.
type Session struct {
	ID            string    `json:"id"`         // Unique session ID
	PlayerID      string    `json:"playerId"`
	ClientAddress string    `json:"clientAddress"` // IP address and port
	ConnectedAt   time.Time `json:"connectedAt"`
	LastActivity  time.Time `json:"lastActivity"`
	// Add other session-specific data, e.g., connection object, subscribed channels
}

// Vector3 could be used for positions if needed.
// type Vector3 struct {
// 	X float64 `json:"x"`
// 	Y float64 `json:"y"`
// 	Z float64 `json:"z"`
// }
