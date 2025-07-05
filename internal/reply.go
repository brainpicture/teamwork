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
	"github.com/sashabaranov/go-openai"
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

	// Handle regular text response or function call
	if strings.HasPrefix(aiResponse, "function_call:") {
		// This is a function call, process it
		log.Printf("🔄 PROCESSING FUNCTION CALL for user %d: %s", user.ID, aiResponse)
		
		// Parse function call format: "function_call:functionName:arguments"
		parts := strings.SplitN(aiResponse, ":", 3)
		if len(parts) >= 3 {
			functionName := parts[1]
			arguments := parts[2]
			
			// Create a fake function call for processing
			functionCall := &openai.FunctionCall{
				Name:      functionName,
				Arguments: arguments,
			}
			
			// Process the function call
			pendingOp, err := ProcessGPTFunctionCall(user.ID, update.Message.Chat.ID, functionCall)
			if err != nil {
				if strings.Contains(err.Error(), "_direct") {
					// This is a direct function call (like list_projects, list_tasks)
					functionType := strings.TrimSuffix(err.Error(), "_direct")
					log.Printf("📋 DIRECT FUNCTION CALL: %s for user %d", functionType, user.ID)
					
					// Parse function arguments
					var params map[string]interface{}
					if json.Unmarshal([]byte(arguments), &params) != nil {
						params = make(map[string]interface{})
					}
					
					// Execute direct function
					var result string
					var directErr error
					switch functionType {
					case "list_projects":
						result, directErr = executeListProjects(db, user.ID, params)
					case "list_tasks":
						result, directErr = executeListTasks(db, user.ID, params)
					case "get_current_project":
						result, directErr = executeGetCurrentProject(db, user.ID, params)
					default:
						directErr = fmt.Errorf("неизвестная функция: %s", functionType)
					}
					
					if directErr != nil {
						log.Printf("Error executing direct function %s: %v", functionType, directErr)
						SendReply(bot, update.Message.Chat.ID, fmt.Sprintf("❌ Ошибка выполнения функции %s: %v", functionType, directErr))
					} else {
						// Format the response using AI
						formattedResponse, formatErr := aiService.FormatDataResponse(ctx, messageText, functionType, result)
						if formatErr != nil {
							log.Printf("Error formatting response: %v", formatErr)
							SendReply(bot, update.Message.Chat.ID, result) // Send raw result as fallback
						} else {
							// Check if formatted response is a function call
							if strings.HasPrefix(formattedResponse, "function_call:") {
								// Process the function call from formatting
								formatParts := strings.SplitN(formattedResponse, ":", 3)
								if len(formatParts) >= 3 {
									formatFunctionName := formatParts[1]
									formatArguments := formatParts[2]
									
									if formatFunctionName == "send_message_with_buttons" {
										// Parse button parameters
										var buttonParams map[string]interface{}
										if json.Unmarshal([]byte(formatArguments), &buttonParams) == nil {
											if message, ok := buttonParams["message"].(string); ok {
												if buttons, ok := buttonParams["buttons"].([]interface{}); ok {
													if err := SendMessageWithCustomButtons(bot, update.Message.Chat.ID, message, buttons); err != nil {
														log.Printf("Error sending message with buttons: %v", err)
														SendReply(bot, update.Message.Chat.ID, message) // Send without buttons as fallback
													}
												}
											}
										}
									} else {
										SendReply(bot, update.Message.Chat.ID, formattedResponse)
									}
								}
							} else {
								SendReply(bot, update.Message.Chat.ID, formattedResponse)
							}
						}
						
						// Save response to conversation history
						if err := db.SaveMessage(user.ID, update.Message.Chat.ID, "assistant", result); err != nil {
							log.Printf("Error saving function result to history: %v", err)
						}
					}
				} else {
					log.Printf("Error processing function call: %v", err)
					SendReply(bot, update.Message.Chat.ID, fmt.Sprintf("❌ Ошибка обработки функции: %v", err))
				}
			} else {
				// This requires confirmation, send confirmation message
				log.Printf("🔄 FUNCTION CALL requires confirmation for user %d", user.ID)
				pendingOp.ChatID = update.Message.Chat.ID
				pendingOperations[pendingOp.ID] = pendingOp
				
				confirmationMsg := CreateConfirmationMessage(db, pendingOp)
				if _, err := bot.Send(confirmationMsg); err != nil {
					log.Printf("Error sending confirmation message: %v", err)
					SendReply(bot, update.Message.Chat.ID, "Ошибка отправки подтверждения")
				}
			}
		} else {
			log.Printf("Invalid function call format: %s", aiResponse)
			SendReply(bot, update.Message.Chat.ID, "❌ Неверный формат вызова функции")
		}
	} else {
		// This is a regular text response
		log.Printf("💬 REGULAR TEXT RESPONSE for user %d: %s", user.ID, aiResponse)
		SendReply(bot, update.Message.Chat.ID, aiResponse)
		
		// Save response to conversation history
		if err := db.SaveMessage(user.ID, update.Message.Chat.ID, "assistant", aiResponse); err != nil {
			log.Printf("Error saving bot message: %v", err)
		}
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
