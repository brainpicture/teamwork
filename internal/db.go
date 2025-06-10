package internal

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB represents a database connection
type DB struct {
	*sql.DB
}

// User represents a user in the database
type User struct {
	ID     int
	TgID   int64
	TgName string
	Email  string
	Name   string
	TS     time.Time
}

// ConnectDB establishes a connection to the database
func ConnectDB(config *Config) (*DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return &DB{db}, nil
}

// GetUserByTgID retrieves a user by their Telegram ID
func (db *DB) GetUserByTgID(tgID int64) (*User, error) {
	user := &User{}
	err := db.QueryRow("SELECT id, tg_id, tg_name, email, name, ts FROM users WHERE tg_id = ?", tgID).Scan(
		&user.ID, &user.TgID, &user.TgName, &user.Email, &user.Name, &user.TS,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}
	return user, nil
}

// CreateUser creates a new user in the database and returns the created user
func (db *DB) CreateUser(tgID int64, tgName, email, name string) (*User, error) {
	result, err := db.Exec("INSERT INTO users (tg_id, tg_name, email, name) VALUES (?, ?, ?, ?)",
		tgID, tgName, email, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	// Get the auto-generated ID
	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %v", err)
	}

	// Return the newly created user
	return &User{
		ID:     int(userID),
		TgID:   tgID,
		TgName: tgName,
		Email:  email,
		Name:   name,
		TS:     time.Now(),
	}, nil
}

// UpdateUser updates an existing user
func (db *DB) UpdateUser(user *User) error {
	_, err := db.Exec("UPDATE users SET tg_name = ?, email = ?, name = ? WHERE tg_id = ?",
		user.TgName, user.Email, user.Name, user.TgID)
	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}
	return nil
}

// GetOrCreateUser gets a user or creates if not exists
// Returns user and a boolean indicating if the user was newly created
func (db *DB) GetOrCreateUser(tgID int64, tgName string) (*User, bool, error) {
	user, err := db.GetUserByTgID(tgID)
	if err != nil {
		return nil, false, err
	}

	if user == nil {
		// User does not exist, create a new one
		user, err = db.CreateUser(tgID, tgName, "", "")
		if err != nil {
			return nil, false, err
		}
		return user, true, nil // true indicates user was newly created
	} else {
		// Update tg_name if it has changed
		if user.TgName != tgName {
			user.TgName = tgName
			err = db.UpdateUser(user)
			if err != nil {
				return nil, false, fmt.Errorf("failed to update user name: %v", err)
			}
		}
		return user, false, nil // false indicates user already existed
	}
}

// LoadConfig loads configuration from environment variables
// This is a universal function that can be used by both bot and database utilities
func LoadConfig() *Config {
	// Get Telegram token (required for bot, optional for db utilities)
	token := os.Getenv("TELEGRAM_API_TOKEN")

	// Get OpenAI settings
	openAIKey := os.Getenv("OPENAI_API_KEY")
	aiEnabled := openAIKey != "" && getEnvBool("AI_ENABLED", true)

	// Default values for database settings
	config := &Config{
		// Telegram settings
		TelegramAPIToken: token,
		DebugMode:        getEnvBool("DEBUG_MODE", true),
		UpdateTimeout:    getEnvInt("UPDATE_TIMEOUT", 60),

		// Database settings (defaults for local development)
		DBHost:     getEnvStr("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 3306),
		DBUser:     getEnvStr("DB_USER", "root"),
		DBPassword: getEnvStr("DB_PASSWORD", ""),
		DBName:     getEnvStr("DB_NAME", "teamwork"),

		// AI settings
		OpenAIAPIKey: openAIKey,
		AIEnabled:    aiEnabled,
	}

	return config
}

// LoadConfigForBot loads configuration for bot with validation
func LoadConfigForBot() *Config {
	config := LoadConfig()

	// Validate required settings for bot
	if config.TelegramAPIToken == "" {
		log.Fatal("TELEGRAM_API_TOKEN is required")
	}

	if config.OpenAIAPIKey == "" {
		log.Println("Warning: OPENAI_API_KEY not set, AI features will be disabled")
	}

	return config
}

// LoadConfigForDB loads configuration for database utilities (minimal validation)
func LoadConfigForDB() *Config {
	return LoadConfig()
}

// getEnvStr reads string environment variable with a default value
func getEnvStr(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvBool reads boolean environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

// getEnvInt reads integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Warning: could not parse %s=%s as integer, using default %d", key, value, defaultValue)
		return defaultValue
	}

	return intValue
}
