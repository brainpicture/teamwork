package main

import (
	"log"
	"telegram-bot/internal"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Load configuration
	config := internal.LoadConfigForBot()

	// Connect to database
	db, err := internal.ConnectDB(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to database successfully")

	// Initialize AI service
	var aiService *internal.AIService
	if config.AIEnabled {
		switch config.AIProvider {
		case "anthropic", "claude":
			if config.AnthropicAPIKey != "" {
				claudeProvider := internal.NewClaudeProvider(config.AnthropicAPIKey)
				aiService = internal.NewAIService(claudeProvider, true)
				log.Println("AI service initialized with Anthropic Claude-3 Opus")
			} else {
				log.Println("Anthropic API key not provided, AI service disabled")
				aiService = internal.NewAIService(nil, false)
			}
		case "openai", "":
			if config.OpenAIAPIKey != "" {
				openAIProvider := internal.NewOpenAIProvider(config.OpenAIAPIKey)
				aiService = internal.NewAIService(openAIProvider, true)
				log.Println("AI service initialized with OpenAI GPT-4o")
			} else {
				log.Println("OpenAI API key not provided, AI service disabled")
				aiService = internal.NewAIService(nil, false)
			}
		default:
			log.Printf("Unknown AI provider '%s', defaulting to OpenAI", config.AIProvider)
			if config.OpenAIAPIKey != "" {
				openAIProvider := internal.NewOpenAIProvider(config.OpenAIAPIKey)
				aiService = internal.NewAIService(openAIProvider, true)
				log.Println("AI service initialized with OpenAI GPT-4o (fallback)")
			} else {
				log.Println("No valid AI provider available, AI service disabled")
				aiService = internal.NewAIService(nil, false)
			}
		}
	} else {
		aiService = internal.NewAIService(nil, false)
		log.Println("AI service disabled")
	}

	// Initialize Telegram bot
	bot, err := tgbotapi.NewBotAPI(config.TelegramAPIToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	bot.Debug = config.DebugMode
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = config.UpdateTimeout

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			internal.HandleUserMessage(bot, db, aiService, update)
		} else if update.CallbackQuery != nil {
			internal.HandleCallbackQuery(bot, db, update.CallbackQuery)
		}
	}
}
