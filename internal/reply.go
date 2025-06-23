package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
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
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Извините, произошла внутренняя ошибка.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Failed to send error message: %v", err)
		}
		return
	}

	if user.ID == 0 {
		log.Printf("Warning: User ID is 0, database operation may have failed.")
	} else {
		if isNewUser {
			log.Printf("NEW USER created: %s (DB_ID: %d, TG_ID: %d)", user.TgName, user.ID, user.TgID)
		} else {
			log.Printf("Existing user found: %s (DB_ID: %d, TG_ID: %d)", user.TgName, user.ID, user.TgID)
		}
	}

	// Handle voice/audio messages
	if update.Message.Voice != nil || update.Message.Audio != nil {
		log.Printf("[%s] sent audio message", tgName)
		handleAudioMessage(bot, db, aiService, update, user)
		return
	}

	// Get message text
	messageText := strings.TrimSpace(update.Message.Text)
	log.Printf("Processing message: '%s', isNewUser: %t", messageText, isNewUser)

	// Send welcome message for new users OR /start command
	if isNewUser || messageText == "/start" {
		log.Printf("Sending welcome message to: %s", user.TgName)
		SendWelcomeMessageWithTyping(bot, db, aiService, update.Message.Chat.ID, user.TgName, user.ID, isNewUser)
		return
	}

	// Handle project management commands
	if strings.HasPrefix(messageText, "/") {
		handleCommand(bot, db, user, messageText, update.Message.Chat.ID)
		return
	}

	// Process regular text message
	processTextMessage(bot, db, aiService, update, user, messageText)
}

// handleCommand processes bot commands
func handleCommand(bot *tgbotapi.BotAPI, db *DB, user *User, command string, chatID int64) {
	switch command {
	case "/projects":
		handleProjectsCommand(bot, db, user, chatID)
	case "/project_add":
		handleProjectAddCommand(bot, db, user, chatID)
	case "/help":
		handleHelpCommand(bot, chatID)
	default:
		SendReply(bot, chatID, "Неизвестная команда. Используйте /help для списка доступных команд.")
	}
}

// handleProjectsCommand shows user's projects
func handleProjectsCommand(bot *tgbotapi.BotAPI, db *DB, user *User, chatID int64) {
	projects, err := db.GetUserProjects(user.ID)
	if err != nil {
		log.Printf("Error getting projects: %v", err)
		SendReply(bot, chatID, "Ошибка при получении списка проектов.")
		return
	}

	if len(projects) == 0 {
		SendMessageWithCreateProjectButton(bot, chatID, "📋 У вас пока нет проектов.\n\nВыберите тип проекта для быстрого создания или нажмите кнопку для создания собственного:")
		return
	}

	// Build projects list
	var message strings.Builder
	message.WriteString("📋 Ваши проекты:\n\n")
	
	for i, project := range projects {
		emoji := getProjectStatusEmoji(project.Status)
		message.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, emoji, project.Title))
		if project.Description != "" {
			message.WriteString(fmt.Sprintf("   📝 %s\n", project.Description))
		}
		message.WriteString(fmt.Sprintf("   📊 Статус: %s\n\n", project.Status))
	}

	SendReply(bot, chatID, message.String())
}

// getProjectStatusEmoji returns emoji for project status
func getProjectStatusEmoji(status ProjectStatus) string {
	switch status {
	case StatusPlanning:
		return "📝"
	case StatusActive:
		return "🚀"
	case StatusPaused:
		return "⏸️"
	case StatusCompleted:
		return "✅"
	case StatusCancelled:
		return "❌"
	default:
		return "❓"
	}
}

// handleProjectAddCommand handles project creation
func handleProjectAddCommand(bot *tgbotapi.BotAPI, db *DB, user *User, chatID int64) {
	SendReply(bot, chatID, "📋 Создание нового проекта\n\nОтправьте название проекта или используйте формат:\n\"Название проекта | Описание проекта\"")
	
	// For simplicity, we'll just show instructions
	// In a more complex implementation, you could track user state
}

// handleHelpCommand shows help information
func handleHelpCommand(bot *tgbotapi.BotAPI, chatID int64) {
	helpText := `🤖 Доступные команды:

📋 Проекты:
/projects - показать ваши проекты
/project_add - создать новый проект

ℹ️ Справка:
/help - показать это сообщение

💬 Также вы можете просто написать мне любое сообщение, и я отвечу!
🎤 Поддерживаются голосовые сообщения (будут преобразованы в текст).`

	SendReply(bot, chatID, helpText)
}

// handleAudioMessage processes voice and audio messages
func handleAudioMessage(bot *tgbotapi.BotAPI, db *DB, aiService *AIService, update tgbotapi.Update, user *User) {
	// Check if AI service is enabled
	if !aiService.IsEnabled() {
		SendReply(bot, update.Message.Chat.ID, "🎤 Получил аудиосообщение, но функция транскрипции недоступна. Пожалуйста, отправьте текстовое сообщение.")
		return
	}

	// Create context with timeout for audio processing
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start typing indicator
	SendTypingWithContext(bot, update.Message.Chat.ID, ctx)

	var fileID string
	var fileName string

	// Get file info based on message type
	if update.Message.Voice != nil {
		fileID = update.Message.Voice.FileID
		fileName = "voice.ogg"
		log.Printf("Processing voice message: duration=%ds", update.Message.Voice.Duration)
	} else if update.Message.Audio != nil {
		fileID = update.Message.Audio.FileID
		fileName = update.Message.Audio.FileName
		if fileName == "" {
			fileName = "audio.mp3"
		}
		log.Printf("Processing audio message: duration=%ds, filename=%s", update.Message.Audio.Duration, fileName)
	}

	// Download the audio file from Telegram
	audioData, err := downloadTelegramFile(bot, fileID)
	if err != nil {
		log.Printf("Error downloading audio file: %v", err)
		SendReply(bot, update.Message.Chat.ID, "❌ Ошибка при скачивании аудиофайла")
		return
	}

	// Transcribe audio
	transcribedText, err := aiService.TranscribeAudio(ctx, audioData, fileName)
	if err != nil {
		log.Printf("Error transcribing audio: %v", err)
		SendReply(bot, update.Message.Chat.ID, "❌ Ошибка при распознавании речи")
		return
	}

	log.Printf("Audio transcribed: %s", transcribedText)

	// Process the transcribed text as a regular message
	if transcribedText != "" {
		processTextMessage(bot, db, aiService, update, user, transcribedText)
	}
}

// downloadTelegramFile downloads a file from Telegram
func downloadTelegramFile(bot *tgbotapi.BotAPI, fileID string) (io.Reader, error) {
	// Get file info from Telegram
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := bot.GetFile(fileConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	// Download file
	fileURL := file.Link(bot.Token)
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %v", err)
	}

	return resp.Body, nil
}

// processTextMessage processes a text message
func processTextMessage(bot *tgbotapi.BotAPI, db *DB, aiService *AIService, update tgbotapi.Update, user *User, messageText string) {
	// Save user message to database
	if err := db.SaveMessage(user.ID, update.Message.Chat.ID, "user", messageText); err != nil {
		log.Printf("Error saving user message: %v", err)
	}

	// Check if message looks like project creation
	if strings.Contains(strings.ToLower(messageText), "создай проект") || 
	   strings.Contains(strings.ToLower(messageText), "создать проект") ||
	   strings.Contains(strings.ToLower(messageText), "новый проект") {
		handleProjectCreationMessage(bot, db, user, update.Message.Chat.ID, messageText)
		return
	}

	// Load recent conversation history (last 20 messages)
	history, err := db.GetRecentMessages(update.Message.Chat.ID, 20)
	if err != nil {
		log.Printf("Error loading conversation history: %v", err)
		history = []*Message{} // Use empty history on error
	}

	// Create context with timeout for AI generation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start typing indicator
	SendTypingWithContext(bot, update.Message.Chat.ID, ctx)

	// Get user's current project for context
	currentProject, err := db.GetUserCurrentProject(user.ID)
	if err != nil {
		log.Printf("Error getting current project for user %d: %v", user.ID, err)
		currentProject = nil
	}

	// Generate AI response with conversation context and current project
	fallbackMessage := "Привет! Я помощник команды разработчиков. Как дела? 👋"
	aiResponse, err := aiService.GenerateResponseWithContextAndProject(ctx, messageText, history, currentProject, fallbackMessage)

	if err != nil {
		log.Printf("AI generation error: %v", err)
		aiResponse = fallbackMessage
	}

	// Send response to user
	SendReply(bot, update.Message.Chat.ID, aiResponse)

	// Save AI response to database
	if err := db.SaveMessage(user.ID, update.Message.Chat.ID, "assistant", aiResponse); err != nil {
		log.Printf("Error saving bot response: %v", err)
	}

	// Cleanup old messages (keep last 50)
	if err := db.CleanupOldMessages(update.Message.Chat.ID, 50); err != nil {
		log.Printf("Error cleaning up old messages: %v", err)
	}
}

// handleProjectCreationMessage handles project creation from natural language
func handleProjectCreationMessage(bot *tgbotapi.BotAPI, db *DB, user *User, chatID int64, messageText string) {
	// Simple parsing - look for project name after "проект"
	var projectName, projectDescription string
	
	// Try to extract project name and description
	text := strings.ToLower(messageText)
	
	// Look for patterns like "создай проект [название]" or "создать проект [название] с описанием [описание]"
	if strings.Contains(text, "создай проект") {
		parts := strings.Split(messageText, "создай проект")
		if len(parts) > 1 {
			remaining := strings.TrimSpace(parts[1])
			if strings.Contains(remaining, " с описанием ") {
				projectParts := strings.Split(remaining, " с описанием ")
				projectName = strings.TrimSpace(projectParts[0])
				if len(projectParts) > 1 {
					projectDescription = strings.TrimSpace(projectParts[1])
				}
			} else {
				projectName = remaining
			}
		}
	} else if strings.Contains(text, "создать проект") {
		parts := strings.Split(messageText, "создать проект")
		if len(parts) > 1 {
			remaining := strings.TrimSpace(parts[1])
			if strings.Contains(remaining, " с описанием ") {
				projectParts := strings.Split(remaining, " с описанием ")
				projectName = strings.TrimSpace(projectParts[0])
				if len(projectParts) > 1 {
					projectDescription = strings.TrimSpace(projectParts[1])
				}
			} else {
				projectName = remaining
			}
		}
	}

	// Clean up project name
	projectName = strings.Trim(projectName, "\"'")
	projectDescription = strings.Trim(projectDescription, "\"'")

	if projectName == "" {
		SendReply(bot, chatID, "Не могу определить название проекта. Попробуйте:\n\"Создай проект Мой сайт\" или \"Создать проект Мой сайт с описанием Простой сайт для компании\"")
		return
	}

	// Create project
	_, err := db.CreateProject(user.ID, projectName, projectDescription)
	if err != nil {
		log.Printf("Error creating project: %v", err)
		SendReply(bot, chatID, "❌ Ошибка при создании проекта. Попробуйте позже.")
		return
	}

	// Success message
	var successMsg string
	if projectDescription != "" {
		successMsg = fmt.Sprintf("✅ Проект '%s' успешно создан!\n📝 Описание: %s\n\nТеперь вы можете управлять проектом с помощью команд или просто пишите мне.", projectName, projectDescription)
	} else {
		successMsg = fmt.Sprintf("✅ Проект '%s' успешно создан!\n\nТеперь вы можете управлять проектом с помощью команд или просто пишите мне.", projectName)
	}

	SendReply(bot, chatID, successMsg)

	// Save success message to history
	if err := db.SaveMessage(user.ID, chatID, "assistant", successMsg); err != nil {
		log.Printf("Error saving project creation message: %v", err)
	}

	log.Printf("User %s created project '%s'", user.TgName, projectName)
}

// SendWelcomeMessageWithTyping sends a welcome message with typing indicator
func SendWelcomeMessageWithTyping(bot *tgbotapi.BotAPI, db *DB, aiService *AIService, chatID int64, userName string, userID int, isNewUser bool) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Start typing indicator
	SendTypingWithContext(bot, chatID, ctx)

	// Check if user has any projects
	projects, err := db.GetUserProjects(userID)
	hasProjects := err == nil && len(projects) > 0

	var welcomeText string

	if isNewUser {
		status := "новый пользователь"
		timestamp := time.Now().Format("15:04, 2 January 2006")

		if hasProjects {
			welcomeText = aiService.GenerateWelcomeMessage(
				ctx,
				userName,
				status,
				timestamp,
				fmt.Sprintf("🎉 Добро пожаловать, %s!\n\nЭто ваш первый раз здесь. Рады видеть вас!\nЯ готов помочь вам с управлением проектами.\n\nИспользуйте /help для списка команд.", userName),
			)
		} else {
			welcomeText = fmt.Sprintf("🎉 Добро пожаловать, %s!\n\nРады видеть вас в первый раз! Я помощник для управления проектами команды.\n\n🚀 Давайте создадим ваш первый проект! Выберите один из популярных вариантов ниже или просто скажите: \"Создай проект [название]\"", userName)
		}
		log.Printf("Sending NEW USER welcome message to %s", userName)
	} else {
		status := "возвращающийся пользователь"
		timestamp := time.Now().Format("15:04, 2 January 2006")

		if hasProjects {
			welcomeText = aiService.GenerateWelcomeMessage(
				ctx,
				userName,
				status,
				timestamp,
				fmt.Sprintf("👋 Привет снова, %s!\n\nРад видеть вас! Чем могу помочь?\n\nИспользуйте /help для списка команд.", userName),
			)
		} else {
			welcomeText = fmt.Sprintf("👋 Привет снова, %s!\n\nЯ заметил, что у вас пока нет проектов. Давайте исправим это!\n\n🚀 Выберите один из популярных типов проектов ниже или просто скажите: \"Создай проект [название]\"", userName)
		}
		log.Printf("Sending /start welcome message to %s", userName)
	}

	// Send message with create project button if no projects exist
	if !hasProjects {
		SendMessageWithCreateProjectButton(bot, chatID, welcomeText)
	} else {
		// Send regular message if user has projects
		SendReply(bot, chatID, welcomeText)
	}
}

// SendReply sends a reply message to the user
func SendReply(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// SendMessageWithCreateProjectButton sends a message with "Create Project" inline button
func SendMessageWithCreateProjectButton(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting

	// Add inline keyboard with "Create Project" button and suggested project names
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Создать проект", "create_project_button"),
		),
		// thats an name suggesion
		/*tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💻 Веб-приложение", "suggest_project_Веб-приложение"),
		),*/
	)

	msg.ReplyMarkup = keyboard
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message with create project button: %v", err)
	}
}
