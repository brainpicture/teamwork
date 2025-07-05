package internal

import (
	"context"
	"encoding/json"
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
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, an internal error occurred.")
		msg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting
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

	// Handle voice/audio messages
	if update.Message.Voice != nil || update.Message.Audio != nil {
		log.Printf("[%s] (ID: %d) sent audio message", tgName, tgID)
		handleAudioMessage(bot, db, aiService, update, user)
		return
	}

	// Get message text
	messageText := strings.TrimSpace(update.Message.Text)
	log.Printf("Processing message: '%s', isNewUser: %t", messageText, isNewUser)

	// Send welcome message for new users OR /start command
	if isNewUser {
		log.Printf("Sending welcome message to NEW USER: %s", user.TgName)
		SendWelcomeMessageWithTyping(bot, db, aiService, update.Message.Chat.ID, user.TgName, user.ID, true)
		return
	}

	if messageText == "/start" {
		log.Printf("Sending welcome message for /start command: %s", user.TgName)
		SendWelcomeMessageWithTyping(bot, db, aiService, update.Message.Chat.ID, user.TgName, user.ID, false)
		return
	}

	// Process text message
	processTextMessage(bot, db, aiService, update, user, messageText)
}

// handleAudioMessage processes voice and audio messages
func handleAudioMessage(bot *tgbotapi.BotAPI, db *DB, aiService *AIService, update tgbotapi.Update, user *User) {
	// Check if AI service is enabled
	if !aiService.IsEnabled() {
		SendReply(bot, update.Message.Chat.ID, "🎤 Получил аудиосообщение, но функция транскрипции недоступна. Пожалуйста, отправьте текстовое сообщение.")
		return
	}

	// Create context with timeout for audio processing
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Longer timeout for audio
	defer cancel()

	// Start typing indicator
	SendTypingWithContext(bot, update.Message.Chat.ID, ctx)

	var fileID string
	var fileName string

	// Get file info based on message type
	if update.Message.Voice != nil {
		fileID = update.Message.Voice.FileID
		fileName = "voice.ogg" // Telegram voice messages are in OGG format
		log.Printf("Processing voice message: duration=%ds", update.Message.Voice.Duration)
	} else if update.Message.Audio != nil {
		fileID = update.Message.Audio.FileID
		fileName = update.Message.Audio.FileName
		if fileName == "" {
			fileName = "audio.mp3" // Default name if not provided
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

// processTextMessage processes a text message (extracted from HandleUserMessage)
func processTextMessage(bot *tgbotapi.BotAPI, db *DB, aiService *AIService, update tgbotapi.Update, user *User, messageText string) {
	// Save user message to database
	if err := db.SaveMessage(user.ID, update.Message.Chat.ID, "user", messageText); err != nil {
		log.Printf("Error saving user message: %v", err)
	}

	// Load recent conversation history (last 50 messages)
	history, err := db.GetRecentMessages(update.Message.Chat.ID, 50)
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
		currentProject = nil // Continue without current project context
	}

	// Generate AI response with conversation context and current project using adaptive prompts
	aiResponse, err := aiService.GenerateResponseWithContextAndProject(ctx, messageText, history, currentProject, "Привет! Я помощник команды разработчиков. Как дела? 👋")

	// Handle AI service errors
	if err != nil {
		// Check if this is a function call
		if strings.HasPrefix(err.Error(), "function_call:") {
			aiResponse = err.Error() // Use the function call as response
		} else {
			// Real error - inform user and save error message
			errorMsg := fmt.Sprintf("❌ Произошла ошибка при обработке запроса: %v", err)
			log.Printf("AI generation error: %v", err)

			// Save error response to database
			if saveErr := db.SaveMessage(user.ID, update.Message.Chat.ID, "assistant", errorMsg); saveErr != nil {
				log.Printf("Error saving bot error response: %v", saveErr)
			}

			SendReply(bot, update.Message.Chat.ID, errorMsg)
			return
		}
	}

	// All AI responses are now treated as JavaScript code
	log.Printf("🔄 EXECUTING JAVASCRIPT for user %d: %s", user.ID, aiResponse)

	parameters := map[string]interface{}{
		"code": aiResponse,
	}

	jsResult, err := executeJavaScriptDirect(db, user.ID, parameters)
	if err != nil {
		log.Printf("Error executing JavaScript: %v", err)

		// Check if this looks like plain text instead of JavaScript
		if !strings.Contains(aiResponse, "message(") && !strings.Contains(aiResponse, "teamwork.") &&
			!strings.Contains(aiResponse, "let ") && !strings.Contains(aiResponse, "const ") &&
			!strings.Contains(aiResponse, "var ") && !strings.Contains(aiResponse, "function") {

			// This looks like plain text, give GPT a detailed hint
			errorMsg := fmt.Sprintf(`🚨 СИСТЕМНАЯ ОШИБКА: Обнаружен обычный текст вместо JavaScript!

❌ Ваш ответ: "%s"

✅ Правильный формат:
message("%s");

🤖 НАПОМИНАНИЕ: Вы - JavaScript интерпретатор!
Вы можете отвечать ТОЛЬКО кодом JavaScript!

📋 Примеры правильных ответов:
• message("Привет! 👋");
• let projects = teamwork.listProjects();
• message("У вас " + projects.length + " проектов");

🔄 Попробуйте еще раз с JavaScript кодом!`, aiResponse, aiResponse)

			SendReply(bot, update.Message.Chat.ID, errorMsg)
		} else {
			// This is a JavaScript syntax error, provide specific help
			jsErrorMsg := fmt.Sprintf(`🚨 ОШИБКА JAVASCRIPT: %v

❌ Ваш код:
%s

🔧 ЧАСТЫЕ ОШИБКИ И ИСПРАВЛЕНИЯ:

1️⃣ Пропущен return в map():
❌ projects.map(p => { title: p.title })
✅ projects.map(p => ({ title: p.title }))
✅ projects.map(p => { return { title: p.title }; })

2️⃣ Неправильный синтаксис объекта:
❌ { title: project.title, count: tasks.length }
✅ let obj = { title: project.title, count: tasks.length };
✅ return { title: project.title, count: tasks.length };

3️⃣ Забыли точку с запятой:
❌ let x = 5
✅ let x = 5;

🔄 Исправьте синтаксис и попробуйте снова!`, err, aiResponse)
			SendReply(bot, update.Message.Chat.ID, jsErrorMsg)

			// Save the error to context so GPT learns
			systemError := fmt.Sprintf("КРИТИЧЕСКАЯ ОШИБКА JAVASCRIPT: GPT написал код с синтаксической ошибкой '%s'. ОБЯЗАТЕЛЬНО проверять синтаксис JavaScript! Частые ошибки: пропущен return в map(), неправильные объекты, забытые точки с запятой.", aiResponse)
			if err := db.SaveMessage(user.ID, update.Message.Chat.ID, "system", systemError); err != nil {
				log.Printf("Error saving JavaScript error to history: %v", err)
			}
		}
		return
	}

	// Parse JavaScript result
	var resultObj map[string]interface{}
	if json.Unmarshal([]byte(jsResult), &resultObj) == nil {
		// Check if result contains pending operations (JSON with requiresConfirmation)
		if requiresConfirmation, ok := resultObj["requiresConfirmation"].(bool); ok && requiresConfirmation {
			// This is a pending operation, handle it normally
			operationID := resultObj["operationID"].(string)
			if pendingOp, exists := pendingOperations[operationID]; exists {
				pendingOp.ChatID = update.Message.Chat.ID // Set correct chat ID
				pendingOperations[operationID] = pendingOp

				confirmationMsg := CreateConfirmationMessage(db, pendingOp)
				if _, err := bot.Send(confirmationMsg); err != nil {
					log.Printf("Error sending confirmation message: %v", err)
					SendReply(bot, update.Message.Chat.ID, "Ошибка отправки подтверждения")
				}
				return
			}
		}

		// Handle messages and output from JavaScript
		messages, hasMessages := resultObj["messages"].([]interface{})
		outputArray, hasOutput := resultObj["output"].([]interface{})

		// Send messages to user if any
		if hasMessages && len(messages) > 0 {
			for _, msg := range messages {
				if msgStr, ok := msg.(string); ok && msgStr != "" {
					SendReply(bot, update.Message.Chat.ID, msgStr)
					// Save each message to history
					if err := db.SaveMessage(user.ID, update.Message.Chat.ID, "assistant", msgStr); err != nil {
						log.Printf("Error saving bot message: %v", err)
					}
				}
			}
		}

		// If there's output data, pass it back to GPT for continuation
		if hasOutput && len(outputArray) > 0 {
			log.Printf("🔄 JavaScript returned %d output items, continuing GPT conversation", len(outputArray))

			// Convert output array to strings and join for context message
			var outputStrings []string
			for _, item := range outputArray {
				if itemStr, ok := item.(string); ok {
					outputStrings = append(outputStrings, itemStr)
				} else {
					outputStrings = append(outputStrings, fmt.Sprintf("%v", item))
				}
			}
			outputData := strings.Join(outputStrings, "\n")

			// Add detailed output data to conversation context
			outputMessage := fmt.Sprintf("Результат выполнения JavaScript кода:\n\nВызванный код вернул следующие данные через output():\n%s\n\nПроанализируй эти данные и продолжи диалог с пользователем.", outputData)
			if err := db.SaveMessage(user.ID, update.Message.Chat.ID, "system", outputMessage); err != nil {
				log.Printf("Error saving JavaScript output to history: %v", err)
			}

			// Generate new AI response based on the output
			messages, err := db.GetRecentMessages(update.Message.Chat.ID, 10)
			if err != nil {
				log.Printf("Error getting recent messages for continuation: %v", err)
				return
			}

			// Send typing indicator while generating response
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			SendTypingWithContext(bot, update.Message.Chat.ID, ctx)

			// Generate AI response with the new context - GPT should generate NEW JavaScript code
			continueResponse, err := aiService.GenerateResponseWithContext(ctx, "Проанализируй данные из output() и сгенерируй НОВЫЙ JavaScript код для обработки этих данных", messages, "")
			if err != nil {
				log.Printf("Error generating continuation response: %v", err)
				return
			}

			// Execute the NEW JavaScript code generated by GPT with prev_output array
			log.Printf("🔄 EXECUTING NEW JS CODE generated by GPT for user %d", user.ID)
			recParams := map[string]interface{}{
				"code":        continueResponse,
				"prev_output": outputArray, // Передаем массив output данных
			}
			recResult, err := executeJavaScriptDirect(db, user.ID, recParams)
			if err == nil {
				// Handle recursive result
				var recObj map[string]interface{}
				if json.Unmarshal([]byte(recResult), &recObj) == nil {
					if recMessages, ok := recObj["messages"].([]interface{}); ok {
						for _, msg := range recMessages {
							if msgStr, ok := msg.(string); ok && msgStr != "" {
								SendReply(bot, update.Message.Chat.ID, msgStr)
								if err := db.SaveMessage(user.ID, update.Message.Chat.ID, "assistant", msgStr); err != nil {
									log.Printf("Error saving recursive bot message: %v", err)
								}
							}
						}
					}
				}
			}
			return
		}

		// If only messages were sent (no output), we're done
		if hasMessages {
			return
		}
	}

	// Fallback: if no messages were sent, this might be an error or unexpected result
	if jsResult != "" {
		log.Printf("⚠️ JavaScript executed but no messages sent to user. Result: %s", jsResult)
		SendReply(bot, update.Message.Chat.ID, "Код выполнен, но результат не был отправлен через message()")
	}

	// Cleanup old messages (keep last 50)
	if err := db.CleanupOldMessages(update.Message.Chat.ID, 50); err != nil {
		log.Printf("Error cleaning up old messages: %v", err)
	}
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
		// Generate AI welcome message for new users
		status := "новый пользователь"
		timestamp := time.Now().Format("15:04, 2 January 2006")

		if hasProjects {
			welcomeText = aiService.GenerateWelcomeMessage(
				ctx,
				userName,
				status,
				timestamp,
				"🎉 Добро пожаловать, "+userName+"!\n\nЭто ваш первый раз здесь. Рады видеть вас!\nЯ готов помочь вам с задачами команды.\n\nИспользуйте команды для взаимодействия со мной.",
			)
		} else {
			// Suggest creating first project for new users with no projects
			welcomeText = fmt.Sprintf("🎉 Добро пожаловать, %s!\n\nРады видеть вас в первый раз! Я помощник для управления проектами команды.\n\n🚀 Давайте создадим ваш первый проект! Выберите один из популярных вариантов ниже или введите свое название:\n\n💡 Пример: \"Создай проект Интернет-магазин\"", userName)
		}
		log.Printf("Sending NEW USER welcome message to %s (hasProjects: %t)", userName, hasProjects)
	} else {
		// Generate AI welcome message for /start command
		status := "возвращающийся пользователь"
		timestamp := time.Now().Format("15:04, 2 January 2006")

		if hasProjects {
			welcomeText = aiService.GenerateWelcomeMessage(
				ctx,
				userName,
				status,
				timestamp,
				"👋 Привет снова, "+userName+"!\n\nРад видеть вас! Чем могу помочь?\n\nИспользуйте команды для взаимодействия со мной.",
			)
		} else {
			// Suggest creating first project for returning users with no projects
			welcomeText = fmt.Sprintf("👋 Привет снова, %s!\n\nЯ заметил, что у вас пока нет проектов. Давайте исправим это!\n\n🚀 Выберите один из популярных типов проектов ниже или создайте свой:\n\n💡 Просто скажите: \"Создай проект [ваше название]\"", userName)
		}
		log.Printf("Sending /start welcome message to %s (hasProjects: %t)", userName, hasProjects)
	}

	// Send message with create project button if no projects exist
	if !hasProjects {
		SendMessageWithCreateProjectButton(bot, chatID, welcomeText)
	} else {
		// Send regular message if user has projects
		msg := tgbotapi.NewMessage(chatID, welcomeText)
		msg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Failed to send welcome message: %v", err)
		} else {
			log.Printf("Welcome message sent successfully to %s", userName)
		}
	}
}

// SendReply sends a reply message to the user
func SendReply(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting
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
