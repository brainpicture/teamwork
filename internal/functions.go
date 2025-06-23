package internal

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)



// HandleCallbackQuery handles button clicks for simple operations
func HandleCallbackQuery(bot *tgbotapi.BotAPI, db *DB, query *tgbotapi.CallbackQuery) {
	data := query.Data
	log.Printf("Callback query: '%s' from user %d", data, query.From.ID)

	// Handle create project button
	if data == "create_project_button" {
		log.Printf("Create project button clicked by user %d", query.From.ID)
		editMsg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID,
			"📋 У вас пока нет проектов\n\n💡 Для создания проекта используйте команду:\n/project_add")
		bot.Send(editMsg)
		bot.Send(tgbotapi.NewCallback(query.ID, "Используйте /project_add для создания проекта"))
		return
	}

	// Handle suggested project name buttons
	if strings.HasPrefix(data, "suggest_project_") {
		projectName := strings.TrimPrefix(data, "suggest_project_")

		// Get user from database
		user, err := db.GetUserByTgID(query.From.ID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			bot.Send(tgbotapi.NewCallback(query.ID, "Ошибка при создании проекта"))
			return
		}
		if user == nil {
			bot.Send(tgbotapi.NewCallback(query.ID, "Пользователь не найден"))
			return
		}

		// Create project directly
		_, err = db.CreateProject(user.ID, projectName, "")
		if err != nil {
			log.Printf("Error creating project: %v", err)
			bot.Send(tgbotapi.NewCallback(query.ID, "Ошибка при создании проекта"))
			return
		}

		// Success
		successMsg := fmt.Sprintf("✅ Проект '%s' успешно создан!", projectName)
		editMsg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, successMsg)
		bot.Send(editMsg)
		bot.Send(tgbotapi.NewCallback(query.ID, "Проект создан!"))

		log.Printf("User %s created project '%s'", user.TgName, projectName)
		return
	}

	// Default callback response
	bot.Send(tgbotapi.NewCallback(query.ID, ""))
}


