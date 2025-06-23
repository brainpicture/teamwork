package internal

// Simple system prompt for basic bot functionality
func GetSystemPrompt() string {
	return `Ты - дружелюбный помощник команды разработчиков в Telegram боте.

Твои функции:
- Отвечать на вопросы пользователей
- Помогать с управлением проектами
- Быть полезным и профессиональным

Отвечай кратко, дружелюбно, используй подходящие эмодзи. 
Если пользователь спрашивает о проектах, предложи использовать команды:
/projects - список проектов
/project_add - добавить проект

Всегда отвечай на русском языке.`
}

// WelcomePromptTemplate template for generating personalized welcome messages
const WelcomePromptTemplate = `Создай краткое приветственное сообщение для пользователя.

Информация о пользователе:
- Имя: %s
- Статус: %s
- Время: %s

Сделай сообщение дружелюбным (2-3 предложения), используй эмодзи, упомяни что это бот команды.`

// ErrorPromptTemplate template for generating user-friendly error messages
const ErrorPromptTemplate = `Создай дружелюбное сообщение об ошибке.

Контекст: %s

Сделай сообщение понятным, без технических деталей, предложи что делать дальше.`

// ProjectListPromptTemplate template for generating project list responses
const ProjectListPromptTemplate = `Создай красивое сообщение со списком проектов пользователя.

Данные проектов: %s
Количество проектов: %d

Требования:
- Покажи проекты в удобном формате
- Группируй по статусам если нужно
- Используй эмодзи для статусов (🔵 planning, 🟢 active, ⏸️ paused, ✅ completed, ❌ cancelled)
- Добавь краткие инструкции по управлению проектами
- Если проектов нет, предложи создать первый`

// ProjectHelpPromptTemplate template for project management help
const ProjectHelpPromptTemplate = `Создай справочное сообщение о командах управления проектами.

Требования:
- Покажи доступные команды для работы с проектами
- Объясни каждую команду кратко
- Используй эмодзи для наглядности
- Добавь примеры использования
- Будь дружелюбным и понятным`

// ProjectCreatedPromptTemplate template for project creation confirmation
const ProjectCreatedPromptTemplate = `Создай сообщение подтверждения создания проекта.

Данные проекта: %s

Требования:
- Поздравь с созданием проекта
- Покажи основную информацию о проекте
- Используй подходящие эмодзи
- Предложи следующие шаги
- Будь мотивирующим`
