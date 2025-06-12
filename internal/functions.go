package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sashabaranov/go-openai"
)

// PendingOperation represents an operation waiting for user confirmation
type PendingOperation struct {
	ID          string                 `json:"id"`
	UserID      int                    `json:"user_id"`
	ChatID      int64                  `json:"chat_id"`
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
	CreatedAt   time.Time              `json:"created_at"`
}

// OperationResult represents the result of executing an operation
type OperationResult struct {
	Success     bool
	Message     string
	ProjectID   *int    // For operations that create/modify projects
	ProjectName *string // For operations that involve projects
}

// PendingOperations stores pending operations in memory
// In production, this should be stored in database
var pendingOperations = make(map[string]*PendingOperation)

// GetGPTFunctions returns all available functions for GPT

// GetGPTFunctions returns all available functions for GPT
// Now returns empty list since all responses are treated as JavaScript code
func GetGPTFunctions() []openai.FunctionDefinition {
	return []openai.FunctionDefinition{}
}

// generateOperationID generates a unique ID for the operation
func generateOperationID() string {
	return fmt.Sprintf("op_%d", time.Now().UnixNano())
}

// handleCreateProject handles the create project function call
func handleCreateProject(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	title, ok := parameters["title"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid title parameter")
	}

	// Description is optional
	description := ""
	if desc, ok := parameters["description"].(string); ok {
		description = desc
	}

	var operationDesc string
	if description != "" {
		operationDesc = fmt.Sprintf("Создать проект '%s'\nОписание: %s", title, description)
	} else {
		operationDesc = fmt.Sprintf("Создать проект '%s'", title)
	}

	operation := &PendingOperation{
		ID:          generateOperationID(),
		UserID:      userID,
		ChatID:      chatID,
		Type:        "create_project",
		Parameters:  parameters,
		Description: operationDesc,
		CreatedAt:   time.Now(),
	}

	pendingOperations[operation.ID] = operation
	return operation, nil
}

// handleUpdateProject handles the update project function call
func handleUpdateProject(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	projectIDFloat, ok := parameters["project_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid project_id parameter")
	}
	projectID := int(projectIDFloat)

	var updates []string
	if title, ok := parameters["title"].(string); ok {
		updates = append(updates, fmt.Sprintf("название: '%s'", title))
	}
	if description, ok := parameters["description"].(string); ok {
		updates = append(updates, fmt.Sprintf("описание: '%s'", description))
	}
	if status, ok := parameters["status"].(string); ok {
		updates = append(updates, fmt.Sprintf("статус: %s", status))
	}

	operation := &PendingOperation{
		ID:          generateOperationID(),
		UserID:      userID,
		ChatID:      chatID,
		Type:        "update_project",
		Parameters:  parameters,
		Description: fmt.Sprintf("Обновить проект #%d (%s)", projectID, fmt.Sprintf("%v", updates)),
		CreatedAt:   time.Now(),
	}

	pendingOperations[operation.ID] = operation
	return operation, nil
}

// handleDeleteProject handles the delete project function call
func handleDeleteProject(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	projectIDFloat, ok := parameters["project_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid project_id parameter")
	}
	projectID := int(projectIDFloat)

	operation := &PendingOperation{
		ID:          generateOperationID(),
		UserID:      userID,
		ChatID:      chatID,
		Type:        "delete_project",
		Parameters:  parameters,
		Description: fmt.Sprintf("Удалить проект #%d", projectID),
		CreatedAt:   time.Now(),
	}

	pendingOperations[operation.ID] = operation
	return operation, nil
}

// handleListProjects handles the list projects function call
func handleListProjects(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	// List projects doesn't need confirmation, we'll handle it differently
	// For now, we'll return an error to indicate this should be handled as a direct response
	return nil, fmt.Errorf("list_projects_direct")
}

// handleCreateTask handles the create task function call
func handleCreateTask(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	// Validate project_id parameter
	if _, ok := parameters["project_id"].(float64); !ok {
		return nil, fmt.Errorf("invalid project_id parameter")
	}

	// Validate title parameter
	title, ok := parameters["title"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid title parameter")
	}

	// Create brief description for now, detailed description will be created in executeCreateTask
	operationDesc := fmt.Sprintf("Создать задачу '%s'", title)

	operation := &PendingOperation{
		ID:          generateOperationID(),
		UserID:      userID,
		ChatID:      chatID,
		Type:        "create_task",
		Parameters:  parameters,
		Description: operationDesc,
		CreatedAt:   time.Now(),
	}

	pendingOperations[operation.ID] = operation
	return operation, nil
}

// handleListTasks handles the list tasks function call
func handleListTasks(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	// List tasks doesn't need confirmation, we'll handle it differently
	return nil, fmt.Errorf("list_tasks_direct")
}

// handleUpdateTask handles the update task function call
func handleUpdateTask(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	taskIDFloat, ok := parameters["task_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid task_id parameter")
	}
	taskID := int(taskIDFloat)

	var updates []string
	if title, ok := parameters["title"].(string); ok {
		updates = append(updates, fmt.Sprintf("название: '%s'", title))
	}
	if description, ok := parameters["description"].(string); ok {
		updates = append(updates, fmt.Sprintf("описание: '%s'", description))
	}
	if status, ok := parameters["status"].(string); ok {
		updates = append(updates, fmt.Sprintf("статус: %s", status))
	}
	if priority, ok := parameters["priority"].(string); ok {
		updates = append(updates, fmt.Sprintf("приоритет: %s", priority))
	}

	operation := &PendingOperation{
		ID:          generateOperationID(),
		UserID:      userID,
		ChatID:      chatID,
		Type:        "update_task",
		Parameters:  parameters,
		Description: fmt.Sprintf("Обновить задачу #%d (%s)", taskID, strings.Join(updates, ", ")),
		CreatedAt:   time.Now(),
	}

	pendingOperations[operation.ID] = operation
	return operation, nil
}

// handleDeleteTask handles the delete task function call
func handleDeleteTask(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	taskIDFloat, ok := parameters["task_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid task_id parameter")
	}
	taskID := int(taskIDFloat)

	operation := &PendingOperation{
		ID:          generateOperationID(),
		UserID:      userID,
		ChatID:      chatID,
		Type:        "delete_task",
		Parameters:  parameters,
		Description: fmt.Sprintf("Удалить задачу #%d", taskID),
		CreatedAt:   time.Now(),
	}

	pendingOperations[operation.ID] = operation
	return operation, nil
}

// CreateConfirmationMessage creates a message with confirmation buttons
func CreateConfirmationMessage(db *DB, operation *PendingOperation) tgbotapi.MessageConfig {
	// For create_task operations, build detailed description
	description := operation.Description
	if operation.Type == "create_task" {
		description = buildDetailedTaskDescription(db, operation)
	}

	msg := tgbotapi.NewMessage(operation.ChatID,
		fmt.Sprintf("🤖 GPT предлагает выполнить операцию:\n\n%s\n\nВы подтверждаете?", description))
	msg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да", fmt.Sprintf("confirm_%s", operation.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет", fmt.Sprintf("cancel_%s", operation.ID)),
		),
	)

	msg.ReplyMarkup = keyboard
	return msg
}

// buildDetailedTaskDescription builds detailed description for task creation
func buildDetailedTaskDescription(db *DB, operation *PendingOperation) string {
	title := operation.Parameters["title"].(string)
	projectID := int(operation.Parameters["project_id"].(float64))

	// Get project name
	project, err := db.GetProjectByIDForUser(projectID, operation.UserID)
	projectName := fmt.Sprintf("#%d", projectID) // fallback to ID
	if err == nil && project != nil {
		projectName = project.Title
	}

	// Get priority (default to medium)
	priority := "medium"
	if prio, ok := operation.Parameters["priority"].(string); ok {
		priority = prio
	}

	// Build description
	description := fmt.Sprintf("Создать задачу '%s'\n📁 Проект: %s\n⚡ Приоритет: %s", title, projectName, priority)

	// Add deadline if specified
	if deadlineStr, ok := operation.Parameters["deadline"].(string); ok && deadlineStr != "" {
		description += fmt.Sprintf("\n⏰ Дедлайн: %s", deadlineStr)
	} else {
		description += "\n⏰ Без дедлайна"
	}

	// Add description if specified
	if desc, ok := operation.Parameters["description"].(string); ok && desc != "" {
		description += fmt.Sprintf("\n📝 Описание: %s", desc)
	}

	return description
}

// HandleCallbackQuery handles button clicks for confirmations
func HandleCallbackQuery(bot *tgbotapi.BotAPI, db *DB, query *tgbotapi.CallbackQuery) {
	data := query.Data
	log.Printf("🔘 CALLBACK QUERY: '%s' from user %d", data, query.From.ID)

	// Handle special create project button
	if data == "create_project_button" {
		log.Printf("🆕 CREATE PROJECT BUTTON clicked by user %d", query.From.ID)
		// Edit the original message to remove the button and show instruction
		editMsg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID,
			"📋 У вас пока нет проектов\n\n💡 Для создания проекта отправьте сообщение в формате:\n\"Создать проект [название]\" или \"Создать проект [название] с описанием [описание]\"")
		editMsg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting
		editMsg.ReplyMarkup = nil
		bot.Send(editMsg)
		bot.Send(tgbotapi.NewCallback(query.ID, "Отправьте название проекта!"))
		return
	}

	// Handle suggested project name buttons
	if strings.HasPrefix(data, "suggest_project_") {
		projectName := strings.TrimPrefix(data, "suggest_project_")

		// Get user from database
		user, err := db.GetUserByTgID(query.From.ID)
		if err != nil {
			log.Printf("Error getting user by TG ID %d: %v", query.From.ID, err)
			bot.Send(tgbotapi.NewCallback(query.ID, "Ошибка при создании проекта"))
			return
		}
		if user == nil {
			log.Printf("User not found for TG ID %d", query.From.ID)
			bot.Send(tgbotapi.NewCallback(query.ID, "Пользователь не найден"))
			return
		}

		// Create project directly (since it's a quick suggestion)
		_, err = db.CreateProject(user.ID, projectName, "")
		if err != nil {
			log.Printf("Error creating suggested project: %v", err)
			bot.Send(tgbotapi.NewCallback(query.ID, "Ошибка при создании проекта"))

			// Edit message to show error
			editMsg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID,
				fmt.Sprintf("❌ Ошибка при создании проекта '%s': %v", projectName, err))
			editMsg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting
			editMsg.ReplyMarkup = nil
			bot.Send(editMsg)
			return
		}

		// Success - edit message and save to history
		successMsg := fmt.Sprintf("✅ Проект '%s' успешно создан!\nМожно добавить в него участников, а также добавлять задачи в этот проект я прослежу чтобы задачи были выполнены.", projectName)
		editMsg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, successMsg)
		editMsg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting
		editMsg.ReplyMarkup = nil
		bot.Send(editMsg)
		bot.Send(tgbotapi.NewCallback(query.ID, "Проект создан!"))

		// Save success message to conversation history
		if err := db.SaveMessage(user.ID, query.Message.Chat.ID, "assistant", successMsg); err != nil {
			log.Printf("Error saving project creation message: %v", err)
		}

		// Cleanup old messages (keep last 50)
		if err := db.CleanupOldMessages(query.Message.Chat.ID, 50); err != nil {
			log.Printf("Error cleaning up old messages: %v", err)
		}

		log.Printf("User %s created suggested project '%s'", user.TgName, projectName)
		return
	}

	// Handle custom buttons
	if strings.HasPrefix(data, "custom_button_") {
		action := strings.TrimPrefix(data, "custom_button_")
		log.Printf("🔘 CUSTOM BUTTON pressed by user %d: %s", query.From.ID, action)

		// Get user from database
		user, err := db.GetUserByTgID(query.From.ID)
		if err != nil {
			log.Printf("Error getting user by TG ID %d: %v", query.From.ID, err)
			bot.Send(tgbotapi.NewCallback(query.ID, "Ошибка при получении пользователя"))
			return
		}
		if user == nil {
			log.Printf("User not found for TG ID %d", query.From.ID)
			bot.Send(tgbotapi.NewCallback(query.ID, "Пользователь не найден"))
			return
		}

		// Save button action as user message to conversation history
		// This way GPT will see the button press in context
		if err := db.SaveMessage(user.ID, query.Message.Chat.ID, "user", action); err != nil {
			log.Printf("Error saving button action message: %v", err)
		}

		// Answer the callback query
		bot.Send(tgbotapi.NewCallback(query.ID, ""))

		log.Printf("✅ Custom button action '%s' saved as user message for user %d", action, user.ID)
		return
	}

	// Parse callback data for confirmation operations
	if len(data) < 7 {
		return
	}

	// Parse callback data: "confirm_operationID" or "cancel_operationID"
	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		log.Printf("Invalid callback data format: %s", data)
		return
	}

	action := parts[0]                          // "confirm" or "cancel"
	operationID := strings.Join(parts[1:], "_") // rejoin in case operation ID contains underscores

	log.Printf("Callback received: action=%s, operationID=%s", action, operationID)

	// Get pending operation
	operation, exists := pendingOperations[operationID]
	if !exists {
		log.Printf("❌ Pending operation %s not found or already processed", operationID)
		bot.Send(tgbotapi.NewCallback(query.ID, "Операция не найдена или уже выполнена"))
		return
	}

	// Check if user has permission
	user, err := db.GetUserByTgID(query.From.ID)
	if err != nil {
		log.Printf("Error getting user by TG ID %d: %v", query.From.ID, err)
		bot.Send(tgbotapi.NewCallback(query.ID, "Ошибка при проверке пользователя"))
		return
	}
	if user == nil {
		log.Printf("User not found for TG ID %d", query.From.ID)
		bot.Send(tgbotapi.NewCallback(query.ID, "Пользователь не найден"))
		return
	}
	if user.ID != operation.UserID {
		log.Printf("Permission denied: user.ID=%d, operation.UserID=%d", user.ID, operation.UserID)
		bot.Send(tgbotapi.NewCallback(query.ID, "Вы не можете подтвердить эту операцию"))
		return
	}

	log.Printf("User %s (ID=%d) processing operation %s with action '%s'", user.TgName, user.ID, operationID, action)

	// Delete the operation from pending
	delete(pendingOperations, operationID)

	// Edit the original message to remove buttons
	editMsg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, query.Message.Text)
	editMsg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting

	log.Printf("Processing action: '%s' (should be 'confirm' or 'cancel')", action)

	if action == "confirm" {
		log.Printf("✅ CONFIRMING OPERATION: %s for user %d", operation.Type, user.ID)
		// Execute the operation
		result := executeOperation(db, operation)
		if result.Success {
			// Handle special case for send_message_with_buttons
			if operation.Type == "send_message_with_buttons" {
				buttons := operation.Parameters["buttons"].([]interface{})
				if err := SendMessageWithCustomButtons(bot, query.Message.Chat.ID, result.Message, buttons); err != nil {
					log.Printf("Error sending message with custom buttons: %v", err)
					editMsg.Text = fmt.Sprintf("❌ Ошибка при отправке сообщения с кнопками: %v", err)
				} else {
					editMsg.Text = "✅ Сообщение с кнопками отправлено!"
				}
			} else {
				editMsg.Text = fmt.Sprintf("✅ %s", result.Message)
			}
			bot.Send(tgbotapi.NewCallback(query.ID, "Операция выполнена!"))

			// Save success message to conversation history
			if err := db.SaveMessage(operation.UserID, operation.ChatID, "assistant", result.Message); err != nil {
				log.Printf("Error saving operation success message: %v", err)
			}

			// Cleanup old messages (keep last 50)
			if err := db.CleanupOldMessages(operation.ChatID, 50); err != nil {
				log.Printf("Error cleaning up old messages: %v", err)
			}
		} else {
			editMsg.Text = fmt.Sprintf("❌ %s", result.Message)
			bot.Send(tgbotapi.NewCallback(query.ID, "Ошибка выполнения операции"))

			// Save error message to conversation history
			if err := db.SaveMessage(operation.UserID, operation.ChatID, "assistant", result.Message); err != nil {
				log.Printf("Error saving operation error message: %v", err)
			}

			// Cleanup old messages (keep last 50)
			if err := db.CleanupOldMessages(operation.ChatID, 50); err != nil {
				log.Printf("Error cleaning up old messages: %v", err)
			}
		}
	} else if action == "cancel" {
		log.Printf("❌ CANCELLING OPERATION: %s for user %d", operation.Type, user.ID)
		cancelMessage := "Операция отменена"
		editMsg.Text = fmt.Sprintf("%s\n\n❌ %s", operation.Description, cancelMessage)
		bot.Send(tgbotapi.NewCallback(query.ID, "Операция отменена"))

		// Save cancellation message to conversation history
		if err := db.SaveMessage(operation.UserID, operation.ChatID, "assistant", cancelMessage); err != nil {
			log.Printf("Error saving operation cancellation message: %v", err)
		}

		// Cleanup old messages (keep last 50)
		if err := db.CleanupOldMessages(operation.ChatID, 50); err != nil {
			log.Printf("Error cleaning up old messages: %v", err)
		}
	}

	editMsg.ReplyMarkup = nil
	bot.Send(editMsg)
}

// executeOperation executes the confirmed operation
func executeOperation(db *DB, operation *PendingOperation) *OperationResult {
	log.Printf("🚀 EXECUTING OPERATION: %s for user %d", operation.Type, operation.UserID)

	switch operation.Type {
	case "create_project":
		return executeCreateProject(db, operation)
	case "update_project":
		return executeUpdateProject(db, operation)
	case "delete_project":
		return executeDeleteProject(db, operation)
	case "create_task":
		return executeCreateTask(db, operation)
	case "update_task":
		return executeUpdateTask(db, operation)
	case "delete_task":
		return executeDeleteTask(db, operation)
	case "set_current_project":
		return executeSetCurrentProject(db, operation)
	case "send_message_with_buttons":
		return executeSendMessageWithButtons(db, operation)
	case "execute_javascript":
		return executeJavaScript(db, operation)
	default:
		log.Printf("❌ Unknown operation type: %s", operation.Type)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Неизвестный тип операции: %s", operation.Type),
		}
	}
}

// executeCreateProject executes the create project operation
func executeCreateProject(db *DB, operation *PendingOperation) *OperationResult {
	title := operation.Parameters["title"].(string)
	log.Printf("🆕 EXECUTING CREATE_PROJECT: '%s' for user %d", title, operation.UserID)

	// Description is optional
	description := ""
	if desc, ok := operation.Parameters["description"].(string); ok {
		description = desc
	}

	_, err := db.CreateProject(operation.UserID, title, description)
	if err != nil {
		log.Printf("❌ Failed to create project '%s' for user %d: %v", title, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при создании проекта: %v", err),
		}
	}

	log.Printf("✅ Successfully created project '%s' for user %d", title, operation.UserID)
	return &OperationResult{
		Success: true,
		Message: fmt.Sprintf("Проект '%s' успешно создан!", title),
	}
}

// executeUpdateProject executes the update project operation
func executeUpdateProject(db *DB, operation *PendingOperation) *OperationResult {
	projectID := int(operation.Parameters["project_id"].(float64))
	log.Printf("✏️ EXECUTING UPDATE_PROJECT: project %d for user %d", projectID, operation.UserID)

	var title, description, status string
	if t, ok := operation.Parameters["title"].(string); ok {
		title = t
	}
	if d, ok := operation.Parameters["description"].(string); ok {
		description = d
	}
	if s, ok := operation.Parameters["status"].(string); ok {
		status = s
	}

	err := db.UpdateProject(projectID, operation.UserID, title, description, ProjectStatus(status))
	if err != nil {
		log.Printf("❌ Failed to update project %d for user %d: %v", projectID, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при обновлении проекта: %v", err),
		}
	}

	log.Printf("✅ Successfully updated project %d for user %d", projectID, operation.UserID)
	return &OperationResult{
		Success: true,
		Message: fmt.Sprintf("Проект #%d успешно обновлен!", projectID),
	}
}

// executeDeleteProject executes the delete project operation
func executeDeleteProject(db *DB, operation *PendingOperation) *OperationResult {
	projectID := int(operation.Parameters["project_id"].(float64))
	log.Printf("🗑️ EXECUTING DELETE_PROJECT: project %d for user %d", projectID, operation.UserID)

	err := db.DeleteProject(projectID, operation.UserID)
	if err != nil {
		log.Printf("❌ Failed to delete project %d for user %d: %v", projectID, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при удалении проекта: %v", err),
		}
	}

	log.Printf("✅ Successfully deleted project %d for user %d", projectID, operation.UserID)
	return &OperationResult{
		Success: true,
		Message: fmt.Sprintf("Проект #%d успешно удален!", projectID),
	}
}

// executeCreateTask executes the create task operation
func executeCreateTask(db *DB, operation *PendingOperation) *OperationResult {
	projectID := int(operation.Parameters["project_id"].(float64))
	title := operation.Parameters["title"].(string)
	log.Printf("📝 EXECUTING CREATE_TASK: '%s' in project %d for user %d", title, projectID, operation.UserID)

	// Get project name for better user experience
	project, err := db.GetProjectByIDForUser(projectID, operation.UserID)
	if err != nil {
		log.Printf("❌ Failed to get project %d for user %d: %v", projectID, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при получении информации о проекте: %v", err),
		}
	}
	if project == nil {
		log.Printf("❌ Project %d not found for user %d", projectID, operation.UserID)
		return &OperationResult{
			Success: false,
			Message: "Проект не найден",
		}
	}

	// Description is optional
	description := ""
	if desc, ok := operation.Parameters["description"].(string); ok {
		description = desc
	}

	// Priority is optional, default to medium
	priority := PriorityMedium
	if prio, ok := operation.Parameters["priority"].(string); ok {
		priority = TaskPriority(prio)
	}

	// Deadline is optional
	var deadline *time.Time
	if deadlineStr, ok := operation.Parameters["deadline"].(string); ok && deadlineStr != "" {
		if t, err := time.Parse("2006-01-02 15:04", deadlineStr); err == nil {
			deadline = &t
		}
	}

	// Create task
	_, err = db.CreateTask(projectID, operation.UserID, title, description, priority, deadline)
	if err != nil {
		log.Printf("❌ Failed to create task '%s' in project %d for user %d: %v", title, projectID, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при создании задачи: %v", err),
		}
	}

	log.Printf("✅ Successfully created task '%s' in project '%s' (ID: %d) for user %d", title, project.Title, projectID, operation.UserID)

	// Build detailed success message
	message := fmt.Sprintf("✅ Задача '%s' успешно создана!\n", title)
	message += fmt.Sprintf("📁 Проект: %s\n", project.Title)
	message += fmt.Sprintf("⚡ Приоритет: %s\n", priority)

	if deadline != nil {
		message += fmt.Sprintf("⏰ Дедлайн: %s", deadline.Format("2006-01-02 15:04"))
	} else {
		message += "⏰ Без дедлайна"
	}

	return &OperationResult{
		Success: true,
		Message: message,
	}
}

// executeListTasks executes list tasks directly (no confirmation needed)
func executeListTasks(db *DB, userID int, parameters map[string]interface{}) (string, error) {
	log.Printf("📝 EXECUTING LIST_TASKS for user %d with params: %v", userID, parameters)

	var tasks []*Task
	var err error

	// Check if project ID filter is provided
	if projectIDFloat, ok := parameters["project_id"].(float64); ok {
		projectID := int(projectIDFloat)
		log.Printf("📝 Filtering tasks by project ID: %d", projectID)
		tasks, err = db.GetProjectTasks(projectID, userID)
	} else if statusStr, ok := parameters["status"].(string); ok {
		log.Printf("📝 Filtering tasks by status: %s", statusStr)
		status := TaskStatus(statusStr)
		tasks, err = db.GetTasksByStatus(userID, status)
	} else {
		log.Printf("📝 Getting all tasks for user")
		tasks, err = db.GetUserTasks(userID)
	}

	if err != nil {
		log.Printf("❌ Failed to get tasks for user %d: %v", userID, err)
		return "", fmt.Errorf("failed to get tasks: %v", err)
	}

	log.Printf("✅ Found %d tasks for user %d", len(tasks), userID)

	// Return JSON data for GPT to format
	result := map[string]interface{}{
		"tasks":   tasks,
		"count":   len(tasks),
		"filters": parameters,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tasks data: %v", err)
	}

	return string(jsonData), nil
}

// getTaskStatusEmoji returns emoji for task status
func getTaskStatusEmoji(status TaskStatus) string {
	switch status {
	case TaskTodo:
		return "📝"
	case TaskInProgress:
		return "🚀"
	case TaskReview:
		return "👀"
	case TaskDone:
		return "✅"
	case TaskCancelled:
		return "❌"
	default:
		return "❓"
	}
}

// getPriorityEmoji returns emoji for task priority
func getPriorityEmoji(priority TaskPriority) string {
	switch priority {
	case PriorityLow:
		return "🟢"
	case PriorityMedium:
		return "🟡"
	case PriorityHigh:
		return "🟠"
	case PriorityUrgent:
		return "🔴"
	default:
		return "⚪"
	}
}

// executeUpdateTask executes the update task operation
func executeUpdateTask(db *DB, operation *PendingOperation) *OperationResult {
	taskID := int(operation.Parameters["task_id"].(float64))
	log.Printf("✏️ EXECUTING UPDATE_TASK: task %d for user %d", taskID, operation.UserID)

	// Get current task data to preserve unchanged fields
	task, err := db.GetTaskByID(taskID, operation.UserID)
	if err != nil {
		log.Printf("❌ Failed to get task %d for user %d: %v", taskID, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при получении задачи: %v", err),
		}
	}
	if task == nil {
		log.Printf("❌ Task %d not found for user %d", taskID, operation.UserID)
		return &OperationResult{
			Success: false,
			Message: "Задача не найдена",
		}
	}

	// Update fields if provided, otherwise keep current values
	title := task.Title
	description := task.Description
	status := task.Status
	priority := task.Priority
	var deadline *time.Time = task.Deadline

	if newTitle, ok := operation.Parameters["title"].(string); ok {
		title = newTitle
	}
	if newDescription, ok := operation.Parameters["description"].(string); ok {
		description = newDescription
	}
	if newStatus, ok := operation.Parameters["status"].(string); ok {
		status = TaskStatus(newStatus)
	}
	if newPriority, ok := operation.Parameters["priority"].(string); ok {
		priority = TaskPriority(newPriority)
	}
	if deadlineStr, ok := operation.Parameters["deadline"].(string); ok && deadlineStr != "" {
		if t, err := time.Parse("2006-01-02 15:04", deadlineStr); err == nil {
			deadline = &t
		}
	}

	err = db.UpdateTask(taskID, operation.UserID, title, description, status, priority, deadline)
	if err != nil {
		log.Printf("❌ Failed to update task %d for user %d: %v", taskID, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при обновлении задачи: %v", err),
		}
	}

	log.Printf("✅ Successfully updated task %d for user %d", taskID, operation.UserID)
	return &OperationResult{
		Success: true,
		Message: fmt.Sprintf("Задача #%d успешно обновлена!", taskID),
	}
}

// executeDeleteTask executes the delete task operation
func executeDeleteTask(db *DB, operation *PendingOperation) *OperationResult {
	taskID := int(operation.Parameters["task_id"].(float64))
	log.Printf("🗑️ EXECUTING DELETE_TASK: task %d for user %d", taskID, operation.UserID)

	err := db.DeleteTask(taskID, operation.UserID)
	if err != nil {
		log.Printf("❌ Failed to delete task %d for user %d: %v", taskID, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при удалении задачи: %v", err),
		}
	}

	log.Printf("✅ Successfully deleted task %d for user %d", taskID, operation.UserID)
	return &OperationResult{
		Success: true,
		Message: fmt.Sprintf("Задача #%d успешно удалена!", taskID),
	}
}

// executeSetCurrentProject executes the set current project operation
func executeSetCurrentProject(db *DB, operation *PendingOperation) *OperationResult {
	projectID := int(operation.Parameters["project_id"].(float64))
	log.Printf("📌 EXECUTING SET_CURRENT_PROJECT: project %d for user %d", projectID, operation.UserID)

	// First, verify that the project exists and belongs to the user
	project, err := db.GetProjectByIDForUser(projectID, operation.UserID)
	if err != nil {
		log.Printf("❌ Failed to get project %d for user %d: %v", projectID, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при получении информации о проекте: %v", err),
		}
	}
	if project == nil {
		log.Printf("❌ Project %d not found for user %d", projectID, operation.UserID)
		return &OperationResult{
			Success: false,
			Message: "Проект не найден",
		}
	}

	// Set as current project
	err = db.SetUserCurrentProject(operation.UserID, projectID)
	if err != nil {
		log.Printf("❌ Failed to set current project %d for user %d: %v", projectID, operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("Ошибка при установке текущего проекта: %v", err),
		}
	}

	log.Printf("✅ Successfully set current project '%s' (ID: %d) for user %d", project.Title, projectID, operation.UserID)
	return &OperationResult{
		Success: true,
		Message: fmt.Sprintf("Проект '%s' установлен как текущий рабочий проект!", project.Title),
	}
}

// executeGetCurrentProject executes get current project directly (no confirmation needed)
func executeGetCurrentProject(db *DB, userID int, parameters map[string]interface{}) (string, error) {
	log.Printf("📁 EXECUTING GET_CURRENT_PROJECT for user %d", userID)

	currentProject, err := db.GetUserCurrentProject(userID)
	if err != nil {
		log.Printf("❌ Failed to get current project for user %d: %v", userID, err)
		return "", fmt.Errorf("failed to get current project: %v", err)
	}

	log.Printf("✅ Found current project for user %d: %v", userID, currentProject != nil)

	// Return JSON data for GPT to format
	result := map[string]interface{}{
		"current_project": currentProject,
		"has_current":     currentProject != nil,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal current project data: %v", err)
	}

	return string(jsonData), nil
}

// ProcessGPTFunctionCall processes a function call from GPT
func ProcessGPTFunctionCall(userID int, chatID int64, functionCall *openai.FunctionCall) (*PendingOperation, error) {
	log.Printf("🔧 GPT FUNCTION CALL: %s for user %d with args: %s", functionCall.Name, userID, functionCall.Arguments)

	// Now we only support execute_javascript function
	if functionCall.Name != "execute_javascript" {
		log.Printf("❌ Unknown function: %s", functionCall.Name)
		return nil, fmt.Errorf("unknown function: %s", functionCall.Name)
	}

	// Parse parameters
	var parameters map[string]interface{}
	if err := json.Unmarshal([]byte(functionCall.Arguments), &parameters); err != nil {
		log.Printf("❌ Failed to parse function arguments for %s: %v", functionCall.Name, err)
		return nil, fmt.Errorf("failed to parse function arguments: %v", err)
	}

	log.Printf("✅ Calling handler for function: %s", functionCall.Name)
	// Call the handler for execute_javascript
	return handleExecuteJavaScript(userID, chatID, parameters)
}

// executeListProjects executes list projects directly (no confirmation needed)
func executeListProjects(db *DB, userID int, parameters map[string]interface{}) (string, error) {
	log.Printf("📋 EXECUTING LIST_PROJECTS for user %d with params: %v", userID, parameters)

	var projects []*Project
	var err error

	// Check if status filter is provided
	if statusStr, ok := parameters["status"].(string); ok {
		log.Printf("📋 Filtering projects by status: %s", statusStr)
		status := ProjectStatus(statusStr)
		projects, err = db.GetUserProjectsByStatus(userID, status)
	} else {
		log.Printf("📋 Getting all projects for user")
		projects, err = db.GetUserProjects(userID)
	}

	if err != nil {
		log.Printf("❌ Failed to get projects for user %d: %v", userID, err)
		return "", fmt.Errorf("failed to get projects: %v", err)
	}

	log.Printf("✅ Found %d projects for user %d", len(projects), userID)

	// Return JSON data for GPT to format
	result := map[string]interface{}{
		"projects": projects,
		"count":    len(projects),
		"filters":  parameters,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal projects data: %v", err)
	}

	return string(jsonData), nil
}

// getStatusEmoji returns emoji for project status
func getStatusEmoji(status ProjectStatus) string {
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

// handleSetCurrentProject handles the set current project function call
func handleSetCurrentProject(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	projectIDFloat, ok := parameters["project_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid project_id parameter")
	}
	projectID := int(projectIDFloat)

	operation := &PendingOperation{
		ID:          generateOperationID(),
		UserID:      userID,
		ChatID:      chatID,
		Type:        "set_current_project",
		Parameters:  parameters,
		Description: fmt.Sprintf("Установить проект #%d как текущий рабочий проект", projectID),
		CreatedAt:   time.Now(),
	}

	pendingOperations[operation.ID] = operation
	return operation, nil
}

// handleGetCurrentProject handles the get current project function call
func handleGetCurrentProject(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	// Get current project doesn't need confirmation, we'll handle it differently
	return nil, fmt.Errorf("get_current_project_direct")
}

// handleSendMessageWithButtons handles the send_message_with_buttons function call
func handleSendMessageWithButtons(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	message, ok := parameters["message"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message parameter")
	}

	buttons, ok := parameters["buttons"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid buttons parameter")
	}

	if len(buttons) > 6 {
		return nil, fmt.Errorf("too many buttons (max 6 allowed)")
	}

	buttonData := make([]string, len(buttons))
	for i, button := range buttons {
		buttonMap, ok := button.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid button format")
		}
		text, ok := buttonMap["text"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid button text format")
		}
		action, ok := buttonMap["action"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid button action format")
		}
		buttonData[i] = fmt.Sprintf("%s|%s", text, action)
	}

	operation := &PendingOperation{
		ID:          generateOperationID(),
		UserID:      userID,
		ChatID:      chatID,
		Type:        "send_message_with_buttons",
		Parameters:  parameters,
		Description: fmt.Sprintf("Отправка сообщения с кнопками: %s", message),
		CreatedAt:   time.Now(),
	}

	pendingOperations[operation.ID] = operation
	return operation, nil
}

// executeSendMessageWithButtons executes sending message with custom buttons
func executeSendMessageWithButtons(db *DB, operation *PendingOperation) *OperationResult {
	message := operation.Parameters["message"].(string)
	buttons := operation.Parameters["buttons"].([]interface{})

	log.Printf("📨 EXECUTING SEND_MESSAGE_WITH_BUTTONS for user %d: %s", operation.UserID, message)

	// Note: This function returns success but the actual message sending is handled separately
	// The bot will send the message with buttons based on this operation result
	buttonTexts := make([]string, len(buttons))
	for i, button := range buttons {
		buttonMap := button.(map[string]interface{})
		buttonTexts[i] = buttonMap["text"].(string)
	}

	log.Printf("✅ Successfully prepared message with %d buttons for user %d", len(buttons), operation.UserID)
	return &OperationResult{
		Success:     true,
		Message:     message, // The message will be sent with buttons
		ProjectID:   nil,
		ProjectName: nil,
	}
}

// SendMessageWithCustomButtons sends a message with custom buttons
func SendMessageWithCustomButtons(bot *tgbotapi.BotAPI, chatID int64, message string, buttons []interface{}) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = tgbotapi.ModeHTML // Enable HTML formatting

	// Build keyboard from buttons
	var keyboardRows [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	for i, button := range buttons {
		buttonMap := button.(map[string]interface{})
		text := buttonMap["text"].(string)
		action := buttonMap["action"].(string)

		// Use action as callback data with special prefix
		callbackData := fmt.Sprintf("custom_button_%s", action)

		btn := tgbotapi.NewInlineKeyboardButtonData(text, callbackData)
		currentRow = append(currentRow, btn)

		// Add row when we have 2 buttons or it's the last button
		if len(currentRow) == 2 || i == len(buttons)-1 {
			keyboardRows = append(keyboardRows, currentRow)
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message with custom buttons: %v", err)
		return err
	}

	return nil
}

// handleExecuteJavaScript handles the execute JavaScript function call
func handleExecuteJavaScript(userID int, chatID int64, parameters map[string]interface{}) (*PendingOperation, error) {
	// JavaScript executes immediately without confirmation
	return nil, fmt.Errorf("execute_javascript_direct")
}

// executeJavaScriptDirect executes JavaScript code directly without requiring confirmation
// validateJavaScriptSyntax performs basic JavaScript syntax validation
func validateJavaScriptSyntax(code string) error {
	// Check for common syntax errors
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Check for missing return in map/filter/forEach callbacks
		if strings.Contains(line, ".map(") && strings.Contains(line, "=> {") {
			// Look for the closing brace and check if there's a return
			braceCount := 0
			hasReturn := false

			for j := i; j < len(lines) && j < i+10; j++ { // Check next 10 lines max
				checkLine := lines[j]
				braceCount += strings.Count(checkLine, "{")
				braceCount -= strings.Count(checkLine, "}")

				if strings.Contains(checkLine, "return ") {
					hasReturn = true
				}

				if braceCount <= 0 && j > i {
					break
				}
			}

			if !hasReturn && braceCount <= 0 {
				return fmt.Errorf("в строке %d: возможно пропущен 'return' в callback функции map()", i+1)
			}
		}

		// Check for unclosed braces (basic check)
		openBraces := strings.Count(trimmed, "{")
		closeBraces := strings.Count(trimmed, "}")
		if openBraces > 0 && closeBraces == 0 && !strings.HasSuffix(trimmed, "{") {
			// This might be a single-line object without proper syntax
			if strings.Contains(trimmed, ":") && !strings.Contains(trimmed, "return") {
				return fmt.Errorf("в строке %d: возможно неправильный синтаксис объекта. Используйте 'return { ... }' или правильные скобки", i+1)
			}
		}
	}

	return nil
}

// autoFixJavaScript attempts to automatically fix common JavaScript errors
func autoFixJavaScript(code string) (string, bool) {
	originalCode := code
	fixed := false

	// Fix missing return in map callbacks
	// Pattern: .map(x => { someProperty: value })
	mapRegex := regexp.MustCompile(`\.map\([^)]*=>\s*\{\s*([^}]+)\s*\}`)
	if mapRegex.MatchString(code) {
		code = mapRegex.ReplaceAllStringFunc(code, func(match string) string {
			// Check if it already has return
			if strings.Contains(match, "return") {
				return match
			}

			// Extract the content inside braces
			braceStart := strings.Index(match, "{")
			braceEnd := strings.LastIndex(match, "}")
			if braceStart != -1 && braceEnd != -1 {
				content := strings.TrimSpace(match[braceStart+1 : braceEnd])

				// If it looks like object properties, wrap with return
				if strings.Contains(content, ":") && !strings.Contains(content, ";") {
					prefix := match[:braceStart+1]
					suffix := match[braceEnd:]
					return prefix + " return { " + content + " }; " + suffix
				}
			}
			return match
		})
		if code != originalCode {
			fixed = true
		}
	}

	// Fix standalone object literals (missing return or assignment)
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for standalone object literal
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") &&
			strings.Contains(trimmed, ":") && !strings.Contains(trimmed, "return") &&
			!strings.Contains(trimmed, "=") {

			// This looks like a standalone object literal, probably should be returned or assigned
			lines[i] = strings.Replace(line, trimmed, "// Fixed: "+trimmed+" (was standalone object)", 1)
			fixed = true
		}
	}

	if fixed {
		code = strings.Join(lines, "\n")
	}

	return code, fixed
}

func executeJavaScriptDirect(db *DB, userID int, parameters map[string]interface{}) (string, error) {
	code, ok := parameters["code"].(string)
	if !ok {
		return "", fmt.Errorf("invalid code parameter")
	}

	// Clean up common issues in the code
	code = strings.TrimSpace(code)

	// Fix common return statement issues in global context
	// Simple approach: remove return statements that are clearly in global scope
	lines := strings.Split(code, "\n")
	var functionDepth int
	var inFunction bool

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track function depth (simple heuristic)
		functionDepth += strings.Count(line, "{")
		functionDepth -= strings.Count(line, "}")

		// Check if we're inside a function
		if strings.Contains(line, "function") && strings.Contains(line, "{") {
			inFunction = true
		}
		if functionDepth <= 0 {
			inFunction = false
			functionDepth = 0
		}

		// Fix standalone return statements in global scope
		if strings.HasPrefix(trimmed, "return ") && !inFunction {
			// Convert "return expr;" to just "expr"
			returnExpr := strings.TrimPrefix(trimmed, "return ")
			returnExpr = strings.TrimSuffix(returnExpr, ";")
			lines[i] = strings.Replace(line, trimmed, returnExpr, 1)
			log.Printf("🔧 Fixed JavaScript return statement: '%s' -> '%s'", trimmed, returnExpr)
		}
	}
	code = strings.Join(lines, "\n")

	// Get timeout, default to 10 seconds
	timeout := 10
	if timeoutFloat, ok := parameters["timeout_seconds"].(float64); ok {
		timeout = int(timeoutFloat)
		if timeout < 1 {
			timeout = 1
		} else if timeout > 60 {
			timeout = 60
		}
	}

	// Get input data if provided
	inputData, _ := parameters["inputData"].(string)

	log.Printf("⚡ EXECUTING JAVASCRIPT (DIRECT) for user %d: %s", userID, code)
	if inputData != "" {
		log.Printf("📥 Input data provided: %.100s...", inputData)
	}

	// Try to auto-fix common JavaScript errors
	fixedCode, wasFixed := autoFixJavaScript(code)
	if wasFixed {
		log.Printf("🔧 Auto-fixed JavaScript code for user %d", userID)
		code = fixedCode
	}

	// Basic JavaScript syntax validation
	if err := validateJavaScriptSyntax(code); err != nil {
		return "", fmt.Errorf("синтаксическая ошибка в JavaScript: %v", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Create Goja runtime
	vm := goja.New()

	// Set up message accumulator for messages to user
	var userMessages []string
	var outputData []string

	// Add message() function to send messages to user
	vm.Set("message", func(call goja.FunctionCall) goja.Value {
		var parts []string
		for _, arg := range call.Arguments {
			if arg.ExportType() == nil {
				parts = append(parts, "undefined")
			} else {
				parts = append(parts, fmt.Sprintf("%v", arg.Export()))
			}
		}
		message := strings.Join(parts, " ")
		userMessages = append(userMessages, message)
		log.Printf("📤 JS Message: %s", message)
		return goja.Undefined()
	})

	// Add output() function to return data to GPT
	vm.Set("output", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			resultValue := call.Arguments[0].Export()
			outputData = append(outputData, fmt.Sprintf("%v", resultValue))
		}
		return goja.Undefined()
	})

	// Add debug() function to inspect objects
	vm.Set("debug", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			value := call.Arguments[0].Export()
			debugStr := fmt.Sprintf("🔍 DEBUG: %+v", value)
			if jsonBytes, err := json.MarshalIndent(value, "", "  "); err == nil {
				debugStr = fmt.Sprintf("🔍 DEBUG JSON:\n%s", string(jsonBytes))
			}
			outputData = append(outputData, debugStr)
			log.Printf("🔍 JavaScript debug: %+v", value)
		}
		return goja.Undefined()
	})

	// Set up enhanced fetch function
	vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("fetch requires at least one argument"))
		}

		url := call.Arguments[0].String()

		// Basic options support
		method := "GET"
		var body string
		headers := make(map[string]string)

		if len(call.Arguments) > 1 {
			if options := call.Arguments[1].ToObject(vm); options != nil {
				if methodVal := options.Get("method"); methodVal != nil {
					method = methodVal.String()
				}
				if bodyVal := options.Get("body"); bodyVal != nil {
					body = bodyVal.String()
				}
				if headersVal := options.Get("headers"); headersVal != nil {
					if headersObj := headersVal.ToObject(vm); headersObj != nil {
						for _, key := range headersObj.Keys() {
							headers[key] = headersObj.Get(key).String()
						}
					}
				}
			}
		}

		// Create HTTP request
		var req *http.Request
		var err error
		if body != "" {
			req, err = http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
		} else {
			req, err = http.NewRequestWithContext(ctx, method, url, nil)
		}

		if err != nil {
			panic(vm.NewTypeError("Failed to create request: " + err.Error()))
		}

		// Set headers
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		// Set default User-Agent if not provided
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "Teamwork-Bot/1.0")
		}

		// Execute request with timeout
		requestTimeout := time.Duration(timeout-1) * time.Second
		if requestTimeout <= 0 {
			requestTimeout = 5 * time.Second
		}
		client := &http.Client{Timeout: requestTimeout}
		resp, err := client.Do(req)
		if err != nil {
			panic(vm.NewTypeError("Fetch failed: " + err.Error()))
		}
		defer resp.Body.Close()

		// Read response body with size limit
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
		if err != nil {
			panic(vm.NewTypeError("Failed to read response: " + err.Error()))
		}
		responseBody := string(bodyBytes)

		// Create response object
		response := vm.NewObject()
		response.Set("status", resp.StatusCode)
		response.Set("statusText", resp.Status)
		response.Set("ok", resp.StatusCode >= 200 && resp.StatusCode < 300)

		// Set response headers
		responseHeaders := vm.NewObject()
		for key, values := range resp.Header {
			if len(values) > 0 {
				responseHeaders.Set(key, values[0])
			}
		}
		response.Set("headers", responseHeaders)

		response.Set("text", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(responseBody)
		})

		response.Set("json", func(call goja.FunctionCall) goja.Value {
			var jsonData interface{}
			err := json.Unmarshal([]byte(responseBody), &jsonData)
			if err != nil {
				panic(vm.NewTypeError("Invalid JSON: " + err.Error()))
			}
			return vm.ToValue(jsonData)
		})

		return response
	})

	// Set up simplified setTimeout
	vm.Set("setTimeout", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("setTimeout requires at least 2 arguments"))
		}

		callback := call.Arguments[0]
		delay := call.Arguments[1].ToInteger()

		go func() {
			time.Sleep(time.Duration(delay) * time.Millisecond)
			if fn, ok := goja.AssertFunction(callback); ok {
				_, err := fn(goja.Undefined())
				if err != nil {
					log.Printf("⚠️ setTimeout callback error: %v", err)
				}
			}
		}()

		return goja.Undefined()
	})

	// Set up basic JSON support
	vm.Set("JSON", vm.NewObject())
	vm.RunString(`
		JSON.parse = function(str) {
			return JSON.parse(str);
		};
		JSON.stringify = function(obj, replacer, space) {
			return JSON.stringify(obj, replacer, space);
		};
	`)

	// Set up Teamwork API object
	teamworkAPI := vm.NewObject()

	// READ FUNCTIONS - execute immediately
	teamworkAPI.Set("listProjects", func(call goja.FunctionCall) goja.Value {
		// Get optional status filter
		var parameters map[string]interface{}
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			if status := call.Arguments[0].String(); status != "" {
				parameters = map[string]interface{}{"status": status}
			}
		}
		if parameters == nil {
			parameters = make(map[string]interface{})
		}

		result, err := executeListProjects(db, userID, parameters)
		if err != nil {
			panic(vm.NewTypeError("Failed to list projects: " + err.Error()))
		}

		// Parse JSON result and extract projects array
		var responseData map[string]interface{}
		if err := json.Unmarshal([]byte(result), &responseData); err != nil {
			panic(vm.NewTypeError("Failed to parse projects data: " + err.Error()))
		}

		// Debug: log what we got
		log.Printf("🔍 Projects API response: %s", result)

		// Return only the projects array, not the whole response object
		projects, ok := responseData["projects"]
		if !ok {
			log.Printf("⚠️ No 'projects' field found in response")
			return vm.ToValue([]interface{}{}) // Return empty array if no projects field
		}

		// Debug: log projects array
		if projectsArray, ok := projects.([]interface{}); ok {
			log.Printf("🔍 Found %d projects in array", len(projectsArray))
			if len(projectsArray) > 0 {
				log.Printf("🔍 First project: %+v", projectsArray[0])
			}
		}

		return vm.ToValue(projects)
	})

	teamworkAPI.Set("listTasks", func(call goja.FunctionCall) goja.Value {
		// Get optional parameters
		parameters := make(map[string]interface{})
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			// If argument is object, use it as parameters
			if obj := call.Arguments[0].ToObject(vm); obj != nil {
				for _, key := range obj.Keys() {
					parameters[key] = obj.Get(key).Export()
				}
			}
		}

		result, err := executeListTasks(db, userID, parameters)
		if err != nil {
			panic(vm.NewTypeError("Failed to list tasks: " + err.Error()))
		}

		// Parse JSON result and extract tasks array
		var responseData map[string]interface{}
		if err := json.Unmarshal([]byte(result), &responseData); err != nil {
			panic(vm.NewTypeError("Failed to parse tasks data: " + err.Error()))
		}

		// Return only the tasks array, not the whole response object
		tasks, ok := responseData["tasks"]
		if !ok {
			return vm.ToValue([]interface{}{}) // Return empty array if no tasks field
		}

		return vm.ToValue(tasks)
	})

	teamworkAPI.Set("getCurrentProject", func(call goja.FunctionCall) goja.Value {
		parameters := make(map[string]interface{})
		result, err := executeGetCurrentProject(db, userID, parameters)
		if err != nil {
			panic(vm.NewTypeError("Failed to get current project: " + err.Error()))
		}

		// Parse JSON result
		var projectData interface{}
		if err := json.Unmarshal([]byte(result), &projectData); err != nil {
			panic(vm.NewTypeError("Failed to parse project data: " + err.Error()))
		}

		return vm.ToValue(projectData)
	})

	// WRITE FUNCTIONS - create pending operations that require confirmation
	// We'll store pending operations in a global map that can be accessed later
	teamworkAPI.Set("createProject", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("createProject requires at least 1 argument (name)"))
		}

		name := call.Arguments[0].String()
		var description string
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			description = call.Arguments[1].String()
		}

		parameters := map[string]interface{}{
			"name":        name,
			"description": description,
		}

		operation, err := handleCreateProject(userID, 0, parameters) // chatID will be set later
		if err != nil {
			panic(vm.NewTypeError("Failed to create project operation: " + err.Error()))
		}

		// Return confirmation requirement
		return vm.ToValue(map[string]interface{}{
			"requiresConfirmation": true,
			"operationID":          operation.ID,
			"description":          operation.Description,
			"type":                 "create_project",
		})
	})

	teamworkAPI.Set("updateProject", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("updateProject requires at least 1 argument (project_id)"))
		}

		projectID := call.Arguments[0].ToFloat()
		parameters := map[string]interface{}{
			"project_id": projectID,
		}

		// Add optional parameters
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			if obj := call.Arguments[1].ToObject(vm); obj != nil {
				for _, key := range obj.Keys() {
					parameters[key] = obj.Get(key).Export()
				}
			}
		}

		operation, err := handleUpdateProject(userID, 0, parameters)
		if err != nil {
			panic(vm.NewTypeError("Failed to create update project operation: " + err.Error()))
		}

		return vm.ToValue(map[string]interface{}{
			"requiresConfirmation": true,
			"operationID":          operation.ID,
			"description":          operation.Description,
			"type":                 "update_project",
		})
	})

	teamworkAPI.Set("deleteProject", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("deleteProject requires 1 argument (project_id)"))
		}

		projectID := call.Arguments[0].ToFloat()
		parameters := map[string]interface{}{
			"project_id": projectID,
		}

		operation, err := handleDeleteProject(userID, 0, parameters)
		if err != nil {
			panic(vm.NewTypeError("Failed to create delete project operation: " + err.Error()))
		}

		return vm.ToValue(map[string]interface{}{
			"requiresConfirmation": true,
			"operationID":          operation.ID,
			"description":          operation.Description,
			"type":                 "delete_project",
		})
	})

	teamworkAPI.Set("createTask", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("createTask requires at least 1 argument (title)"))
		}

		title := call.Arguments[0].String()
		parameters := map[string]interface{}{
			"title": title,
		}

		// Add optional parameters
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			if obj := call.Arguments[1].ToObject(vm); obj != nil {
				for _, key := range obj.Keys() {
					parameters[key] = obj.Get(key).Export()
				}
			}
		}

		operation, err := handleCreateTask(userID, 0, parameters)
		if err != nil {
			panic(vm.NewTypeError("Failed to create task operation: " + err.Error()))
		}

		return vm.ToValue(map[string]interface{}{
			"requiresConfirmation": true,
			"operationID":          operation.ID,
			"description":          operation.Description,
			"type":                 "create_task",
		})
	})

	teamworkAPI.Set("updateTask", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("updateTask requires at least 1 argument (task_id)"))
		}

		taskID := call.Arguments[0].ToFloat()
		parameters := map[string]interface{}{
			"task_id": taskID,
		}

		// Add optional parameters
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			if obj := call.Arguments[1].ToObject(vm); obj != nil {
				for _, key := range obj.Keys() {
					parameters[key] = obj.Get(key).Export()
				}
			}
		}

		operation, err := handleUpdateTask(userID, 0, parameters)
		if err != nil {
			panic(vm.NewTypeError("Failed to create update task operation: " + err.Error()))
		}

		return vm.ToValue(map[string]interface{}{
			"requiresConfirmation": true,
			"operationID":          operation.ID,
			"description":          operation.Description,
			"type":                 "update_task",
		})
	})

	teamworkAPI.Set("deleteTask", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("deleteTask requires 1 argument (task_id)"))
		}

		taskID := call.Arguments[0].ToFloat()
		parameters := map[string]interface{}{
			"task_id": taskID,
		}

		operation, err := handleDeleteTask(userID, 0, parameters)
		if err != nil {
			panic(vm.NewTypeError("Failed to create delete task operation: " + err.Error()))
		}

		return vm.ToValue(map[string]interface{}{
			"requiresConfirmation": true,
			"operationID":          operation.ID,
			"description":          operation.Description,
			"type":                 "delete_task",
		})
	})

	teamworkAPI.Set("setCurrentProject", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {

			panic(vm.NewTypeError("setCurrentProject requires 1 argument (project_id)"))
		}

		projectID := call.Arguments[0].ToFloat()
		parameters := map[string]interface{}{
			"project_id": projectID,
		}

		operation, err := handleSetCurrentProject(userID, 0, parameters)
		if err != nil {
			panic(vm.NewTypeError("Failed to create set current project operation: " + err.Error()))
		}

		return vm.ToValue(map[string]interface{}{
			"requiresConfirmation": true,
			"operationID":          operation.ID,
			"description":          operation.Description,
			"type":                 "set_current_project",
		})
	})

	// Special function to send messages with buttons
	teamworkAPI.Set("sendMessageWithButtons", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("sendMessageWithButtons requires 2 arguments (message, buttons)"))
		}

		message := call.Arguments[0].String()
		buttonsArg := call.Arguments[1]

		// Convert buttons to interface{} slice
		if buttonsArray := buttonsArg.ToObject(vm); buttonsArray != nil {
			var buttons []interface{}
			keys := buttonsArray.Keys()
			for _, key := range keys {
				button := buttonsArray.Get(key).ToObject(vm)
				if button != nil {
					buttonMap := make(map[string]interface{})
					for _, btnKey := range button.Keys() {
						buttonMap[btnKey] = button.Get(btnKey).Export()
					}
					buttons = append(buttons, buttonMap)
				}
			}

			parameters := map[string]interface{}{
				"message": message,
				"buttons": buttons,
			}

			operation, err := handleSendMessageWithButtons(userID, 0, parameters)
			if err != nil {
				panic(vm.NewTypeError("Failed to create send message operation: " + err.Error()))
			}

			return vm.ToValue(map[string]interface{}{
				"requiresConfirmation": true,
				"operationID":          operation.ID,
				"description":          operation.Description,
				"type":                 "send_message_with_buttons",
			})
		}

		panic(vm.NewTypeError("Invalid buttons parameter"))
	})

	vm.Set("teamwork", teamworkAPI)

	// Set inputData variable if provided (legacy support)
	if inputData != "" {
		vm.Set("inputData", inputData)
		log.Printf("📥 Set inputData variable in JavaScript: %.100s...", inputData)
	} else {
		vm.Set("inputData", goja.Undefined())
	}

	// Set prev_output array with data from previous output() calls
	prevOutputArray, hasPrevOutput := parameters["prev_output"].([]interface{})
	if hasPrevOutput && len(prevOutputArray) > 0 {
		// Convert to string array for JavaScript
		var jsArray []string
		for _, item := range prevOutputArray {
			if itemStr, ok := item.(string); ok {
				jsArray = append(jsArray, itemStr)
			} else {
				jsArray = append(jsArray, fmt.Sprintf("%v", item))
			}
		}
		vm.Set("prev_output", jsArray)
		log.Printf("📥 Set prev_output array in JavaScript with %d items", len(jsArray))
	} else {
		vm.Set("prev_output", []string{})
	}

	// Execute code with timeout handling
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic: %v", r)
			}
		}()

		result, err := vm.RunString(code)
		if err != nil {
			errChan <- err
			return
		}

		resultChan <- result.Export()
	}()

	select {
	case <-ctx.Done():
		log.Printf("❌ JavaScript execution timed out for user %d", userID)
		return "", fmt.Errorf("⏰ Вычисление заняло слишком много времени (%d сек)", timeout)
	case err := <-errChan:
		log.Printf("❌ JavaScript execution failed for user %d: %v", userID, err)
		return "", fmt.Errorf("❌ Ошибка вычисления: %v", err)
	case result := <-resultChan:
		log.Printf("✅ JavaScript executed successfully for user %d", userID)

		// Build response structure
		response := map[string]interface{}{}

		// Add user messages if any
		if len(userMessages) > 0 {
			response["messages"] = userMessages
		}

		// Add output data if any (as array, not joined string)
		if len(outputData) > 0 {
			response["output"] = outputData
		}

		// If no specific outputs, include execution result as array for consistency
		if len(outputData) == 0 && len(userMessages) == 0 {
			var resultStr string
			if result == nil {
				resultStr = "undefined"
			} else {
				if resultBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
					resultStr = string(resultBytes)
				} else {
					resultStr = fmt.Sprintf("%v", result)
				}
			}
			response["output"] = []string{resultStr}
		}

		// Return JSON response with messages and/or output
		responseBytes, _ := json.Marshal(response)
		return string(responseBytes), nil
	}
}

// executeJavaScript executes JavaScript code in a secure sandbox using Goja with custom fetch
func executeJavaScript(db *DB, operation *PendingOperation) *OperationResult {
	code := operation.Parameters["code"].(string)

	// Get timeout, default to 10 seconds
	timeout := 10
	if timeoutFloat, ok := operation.Parameters["timeout_seconds"].(float64); ok {
		timeout = int(timeoutFloat)
		if timeout < 1 {
			timeout = 1
		} else if timeout > 60 {
			timeout = 60
		}
	}

	log.Printf("⚡ EXECUTING JAVASCRIPT for user %d: %.100s...", operation.UserID, code)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Create Goja runtime
	vm := goja.New()

	// Set up console object
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		var args []interface{}
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		log.Printf("📋 JS Console: %v", args...)
		return goja.Undefined()
	})
	vm.Set("console", console)

	// Set up enhanced fetch function
	vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("fetch requires at least one argument"))
		}

		url := call.Arguments[0].String()

		// Basic options support
		method := "GET"
		var body string
		headers := make(map[string]string)

		if len(call.Arguments) > 1 {
			if options := call.Arguments[1].ToObject(vm); options != nil {
				if methodVal := options.Get("method"); methodVal != nil {
					method = methodVal.String()
				}
				if bodyVal := options.Get("body"); bodyVal != nil {
					body = bodyVal.String()
				}
				if headersVal := options.Get("headers"); headersVal != nil {
					if headersObj := headersVal.ToObject(vm); headersObj != nil {
						for _, key := range headersObj.Keys() {
							headers[key] = headersObj.Get(key).String()
						}
					}
				}
			}
		}

		// Create HTTP request
		var req *http.Request
		var err error
		if body != "" {
			req, err = http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
		} else {
			req, err = http.NewRequestWithContext(ctx, method, url, nil)
		}

		if err != nil {
			panic(vm.NewTypeError("Failed to create request: " + err.Error()))
		}

		// Set headers
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		// Set default User-Agent if not provided
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "Teamwork-Bot/1.0")
		}

		// Execute request with timeout
		requestTimeout := time.Duration(timeout-1) * time.Second
		if requestTimeout <= 0 {
			requestTimeout = 5 * time.Second
		}
		client := &http.Client{Timeout: requestTimeout}
		resp, err := client.Do(req)
		if err != nil {
			panic(vm.NewTypeError("Fetch failed: " + err.Error()))
		}
		defer resp.Body.Close()

		// Read response body with size limit
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
		if err != nil {
			panic(vm.NewTypeError("Failed to read response: " + err.Error()))
		}
		responseBody := string(bodyBytes)

		// Create response object
		response := vm.NewObject()
		response.Set("status", resp.StatusCode)
		response.Set("statusText", resp.Status)
		response.Set("ok", resp.StatusCode >= 200 && resp.StatusCode < 300)

		// Set response headers
		responseHeaders := vm.NewObject()
		for key, values := range resp.Header {
			if len(values) > 0 {
				responseHeaders.Set(key, values[0])
			}
		}
		response.Set("headers", responseHeaders)

		response.Set("text", func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(responseBody)
		})

		response.Set("json", func(call goja.FunctionCall) goja.Value {
			var jsonData interface{}
			err := json.Unmarshal([]byte(responseBody), &jsonData)
			if err != nil {
				panic(vm.NewTypeError("Invalid JSON: " + err.Error()))
			}
			return vm.ToValue(jsonData)
		})

		return response
	})

	// Set up simplified setTimeout
	vm.Set("setTimeout", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("setTimeout requires at least 2 arguments"))
		}

		callback := call.Arguments[0]
		delay := call.Arguments[1].ToInteger()

		go func() {
			time.Sleep(time.Duration(delay) * time.Millisecond)
			if fn, ok := goja.AssertFunction(callback); ok {
				_, err := fn(goja.Undefined())
				if err != nil {
					log.Printf("⚠️ setTimeout callback error: %v", err)
				}
			}
		}()

		return goja.Undefined()
	})

	// Set up basic JSON support
	vm.Set("JSON", vm.NewObject())
	vm.RunString(`
		JSON.parse = function(str) {
			return JSON.parse(str);
		};
		JSON.stringify = function(obj, replacer, space) {
			return JSON.stringify(obj, replacer, space);
		};
	`)

	// Execute code with timeout handling
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic: %v", r)
			}
		}()

		result, err := vm.RunString(code)
		if err != nil {
			errChan <- err
			return
		}

		resultChan <- result.Export()
	}()

	select {
	case <-ctx.Done():
		log.Printf("❌ JavaScript execution timed out for user %d", operation.UserID)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("⏰ Выполнение JavaScript превысило лимит времени (%d сек)", timeout),
		}
	case err := <-errChan:
		log.Printf("❌ JavaScript execution failed for user %d: %v", operation.UserID, err)
		return &OperationResult{
			Success: false,
			Message: fmt.Sprintf("❌ Ошибка выполнения JavaScript: %v", err),
		}
	case result := <-resultChan:
		log.Printf("✅ JavaScript executed successfully for user %d", operation.UserID)

		// Format result for display
		var resultStr string
		if result == nil {
			resultStr = "undefined"
		} else {
			if resultBytes, err := json.MarshalIndent(result, "", "  "); err == nil {
				resultStr = string(resultBytes)
			} else {
				resultStr = fmt.Sprintf("%v", result)
			}
		}

		message := fmt.Sprintf("✅ JavaScript выполнен успешно!\n\n📊 Результат:\n```\n%s\n```", resultStr)

		return &OperationResult{
			Success: true,
			Message: message,
		}
	}
}
