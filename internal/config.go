package internal

// Config represents application configuration
type Config struct {
	// Telegram settings
	TelegramAPIToken string
	DebugMode        bool
	UpdateTimeout    int

	// Database settings
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// AI settings
	OpenAIAPIKey string
	AIEnabled    bool
}
