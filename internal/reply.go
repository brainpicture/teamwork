package internal

import (
	"context"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendTypingAction sends "typing..." indicator to the chat
func SendTypingAction(bot *tgbotapi.BotAPI, chatID int64) {
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	if _, err := bot.Request(action); err != nil {
		log.Printf("Failed to send typing action: %v", err)
	}
}

// SendTypingWithContext sends typing action and maintains it during context execution
func SendTypingWithContext(bot *tgbotapi.BotAPI, chatID int64, ctx context.Context) {
	// Send initial typing action
	SendTypingAction(bot, chatID)

	// Continue sending typing action every 4 seconds while context is active
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				SendTypingAction(bot, chatID)
			}
		}
	}()
}

// HandleUserMessage processes incoming user messages and handles database operations
func HandleUserMessage(bot *tgbotapi.BotAPI, db *DB, aiService *AIService, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	// Get Telegram user ID and name
	tgID := update.Message.From.ID
	tgName := update.Message.From.UserName
	if tgName == "" {
		tgName = update.Message.From.FirstName
		if update.Message.From.LastName != "" {
			tgName += " " + update.Message.From.LastName
		}
	}

	log.Printf("[%s] (ID: %d) %s", tgName, tgID, update.Message.Text)

	// Store or update user in database
	user, isNewUser, err := db.GetOrCreateUser(tgID, tgName)
	if err != nil {
		log.Printf("Error handling user in database: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, an internal error occurred.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Failed to send error message: %v", err)
		}
		return
	}

	if user.ID == 0 {
		log.Printf("Warning: User ID is 0, database operation may have failed.")
	} else {
		if isNewUser {
			log.Printf("NEW USER created in database: %s (DB_ID: %d, TG_ID: %d)", user.TgName, user.ID, user.TgID)
		} else {
			log.Printf("Existing user found in database: %s (DB_ID: %d, TG_ID: %d)", user.TgName, user.ID, user.TgID)
		}
	}

	// Get message text
	messageText := strings.TrimSpace(update.Message.Text)
	log.Printf("Processing message: '%s', isNewUser: %t", messageText, isNewUser)

	// Send welcome message for new users OR /start command
	if isNewUser {
		log.Printf("Sending welcome message to NEW USER: %s", user.TgName)
		SendWelcomeMessageWithTyping(bot, aiService, update.Message.Chat.ID, user.TgName, true)
		return
	}

	if messageText == "/start" {
		log.Printf("Sending welcome message for /start command: %s", user.TgName)
		SendWelcomeMessageWithTyping(bot, aiService, update.Message.Chat.ID, user.TgName, false)
		return
	}

	// Generate AI response for regular messages with typing indicator
	log.Printf("Generating AI response for: %s", user.TgName)

	// Create context with timeout for AI generation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start typing indicator
	SendTypingWithContext(bot, update.Message.Chat.ID, ctx)

	// Generate AI response
	aiResponse := aiService.GenerateResponse(ctx, messageText, "Привет! Я помощник команды разработчиков. Как дела? 👋")

	SendReply(bot, update.Message.Chat.ID, aiResponse)
}

// SendWelcomeMessageWithTyping sends a welcome message with typing indicator
func SendWelcomeMessageWithTyping(bot *tgbotapi.BotAPI, aiService *AIService, chatID int64, userName string, isNewUser bool) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Start typing indicator
	SendTypingWithContext(bot, chatID, ctx)

	var welcomeText string

	if isNewUser {
		// Generate AI welcome message for new users
		status := "новый пользователь"
		timestamp := time.Now().Format("15:04, 2 January 2006")

		welcomeText = aiService.GenerateWelcomeMessage(
			ctx,
			userName,
			status,
			timestamp,
			"🎉 Добро пожаловать, "+userName+"!\n\nЭто ваш первый раз здесь. Рады видеть вас!\nЯ готов помочь вам с задачами команды.\n\nИспользуйте команды для взаимодействия со мной.",
		)
		log.Printf("Sending NEW USER welcome message to %s", userName)
	} else {
		// Generate AI welcome message for /start command
		status := "возвращающийся пользователь"
		timestamp := time.Now().Format("15:04, 2 January 2006")

		welcomeText = aiService.GenerateWelcomeMessage(
			ctx,
			userName,
			status,
			timestamp,
			"👋 Привет снова, "+userName+"!\n\nРад видеть вас! Чем могу помочь?\n\nИспользуйте команды для взаимодействия со мной.",
		)
		log.Printf("Sending /start welcome message to %s", userName)
	}

	msg := tgbotapi.NewMessage(chatID, welcomeText)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send welcome message: %v", err)
	} else {
		log.Printf("Welcome message sent successfully to %s", userName)
	}
}

// SendWelcomeMessage sends a welcome message to new users or /start command (legacy function)
func SendWelcomeMessage(bot *tgbotapi.BotAPI, aiService *AIService, chatID int64, userName string, isNewUser bool) {
	SendWelcomeMessageWithTyping(bot, aiService, chatID, userName, isNewUser)
}

// SendReply sends a reply message to the user
func SendReply(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}
