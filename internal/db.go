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
	ID               int
	TgID             int64
	TgName           string
	Email            string
	Name             string
	CurrentProjectID *int // Pointer to allow NULL values
	TS               time.Time
}

// Message represents a conversation message in the database
type Message struct {
	ID        int
	UserID    int
	ChatID    int64
	Role      string // 'user' or 'assistant'
	Content   string
	CreatedAt time.Time
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
	var currentProjectID sql.NullInt64
	err := db.QueryRow("SELECT id, tg_id, tg_name, email, name, current_project_id, ts FROM users WHERE tg_id = ?", tgID).Scan(
		&user.ID, &user.TgID, &user.TgName, &user.Email, &user.Name, &currentProjectID, &user.TS,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	if currentProjectID.Valid {
		projectID := int(currentProjectID.Int64)
		user.CurrentProjectID = &projectID
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
	_, err := db.Exec("UPDATE users SET tg_name = ?, email = ?, name = ?, current_project_id = ? WHERE tg_id = ?",
		user.TgName, user.Email, user.Name, user.CurrentProjectID, user.TgID)
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
		OpenAIAPIKey:    openAIKey,
		AnthropicAPIKey: getEnvStr("ANTHROPIC_API_KEY", ""),
		AIProvider:      getEnvStr("AI_PROVIDER", "openai"),
		AIEnabled:       aiEnabled,
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

	if config.AIEnabled {
		switch config.AIProvider {
		case "anthropic", "claude":
			if config.AnthropicAPIKey == "" {
				log.Println("Warning: ANTHROPIC_API_KEY not set, AI features will be disabled")
			}
		case "openai", "":
			if config.OpenAIAPIKey == "" {
				log.Println("Warning: OPENAI_API_KEY not set, AI features will be disabled")
			}
		default:
			log.Printf("Warning: Unknown AI provider '%s', defaulting to OpenAI", config.AIProvider)
			if config.OpenAIAPIKey == "" {
				log.Println("Warning: OPENAI_API_KEY not set, AI features will be disabled")
			}
		}
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

// SaveMessage saves a message to the database
func (db *DB) SaveMessage(userID int, chatID int64, role, content string) error {
	_, err := db.Exec(
		"INSERT INTO messages (user_id, chat_id, role, content) VALUES (?, ?, ?, ?)",
		userID, chatID, role, content,
	)
	if err != nil {
		return fmt.Errorf("failed to save message: %v", err)
	}
	return nil
}

// GetRecentMessages retrieves the last N messages for a chat
func (db *DB) GetRecentMessages(chatID int64, limit int) ([]*Message, error) {
	query := `
		SELECT id, user_id, chat_id, role, content, created_at 
		FROM messages 
		WHERE chat_id = ? 
		ORDER BY created_at DESC 
		LIMIT ?
	`

	rows, err := db.Query(query, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %v", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.ChatID, &msg.Role, &msg.Content, &msg.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %v", err)
		}
		messages = append(messages, msg)
	}

	// Reverse the slice to get chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// CleanupOldMessages removes old messages beyond the limit for a chat
func (db *DB) CleanupOldMessages(chatID int64, keepCount int) error {
	query := `
		DELETE FROM messages 
		WHERE chat_id = ? 
		AND id NOT IN (
			SELECT id FROM (
				SELECT id FROM messages 
				WHERE chat_id = ? 
				ORDER BY created_at DESC 
				LIMIT ?
			) AS recent_messages
		)
	`

	_, err := db.Exec(query, chatID, chatID, keepCount)
	if err != nil {
		return fmt.Errorf("failed to cleanup old messages: %v", err)
	}
	return nil
}

// SetUserCurrentProject sets the current project for a user
func (db *DB) SetUserCurrentProject(userID, projectID int) error {
	// First verify that the user has access to this project
	userRole, err := db.GetUserRoleInProject(projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to check project access: %v", err)
	}
	if userRole == "" {
		return fmt.Errorf("user does not have access to this project")
	}

	_, err = db.Exec("UPDATE users SET current_project_id = ? WHERE id = ?", projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to set current project: %v", err)
	}
	return nil
}

// ClearUserCurrentProject clears the current project for a user
func (db *DB) ClearUserCurrentProject(userID int) error {
	_, err := db.Exec("UPDATE users SET current_project_id = NULL WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to clear current project: %v", err)
	}
	return nil
}

// GetUserCurrentProject gets the current project for a user with details
func (db *DB) GetUserCurrentProject(userID int) (*Project, error) {
	query := `
		SELECT p.id, p.title, p.description, p.status, 
		       p.created_at, p.updated_at, pu.role
		FROM users u
		JOIN projects p ON u.current_project_id = p.id
		JOIN project_users pu ON p.id = pu.project_id AND pu.user_id = u.id
		WHERE u.id = ? AND u.current_project_id IS NOT NULL
	`

	project := &Project{}
	err := db.QueryRow(query, userID).Scan(
		&project.ID, &project.Title, &project.Description,
		&project.Status, &project.CreatedAt, &project.UpdatedAt,
		&project.UserRole,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get current project: %v", err)
	}

	return project, nil
}
