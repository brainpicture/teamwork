package internal

import (
	"fmt"
	"strings"
	"time"
)

// PromptModule represents a modular prompt component
type PromptModule struct {
	Name        string
	Content     string
	Priority    int
	Conditional bool
}

// PromptContext contains information for building adaptive prompts
type PromptContext struct {
	UserID         int
	HasProjects    bool
	CurrentProject *Project
	RecentActions  []string
	Capability     string // "basic", "advanced", "expert"
}

// Core prompt modules
var (
	// Base role definition
	BaseRoleModule = PromptModule{
		Name:     "base_role",
		Priority: 1,
		Content: `🤖 ТЫ - УМНЫЙ ПОМОЩНИК ПО УПРАВЛЕНИЮ ПРОЕКТАМИ

🎯 ТВОЯ РОЛЬ:
- Помогать пользователям управлять проектами и задачами
- Обрабатывать естественный язык и выполнять соответствующие действия
- Использовать доступные функции для работы с базой данных
- Предоставлять четкие и полезные ответы`,
	}

	// Function calling rules
	FunctionCallingRulesModule = PromptModule{
		Name:     "function_calling_rules",
		Priority: 2,
		Content: `� ПРАВИЛА ИСПОЛЬЗОВАНИЯ ФУНКЦИЙ:

✅ ВСЕГДА используй доступные функции для выполнения действий:
- Для получения списка проектов: используй list_projects
- Для создания проекта: используй create_project
- Для работы с задачами: используй create_task, list_tasks, update_task
- Для отправки сообщений с кнопками: используй send_message_with_buttons

🎯 СТРАТЕГИЯ ВЫБОРА ФУНКЦИЙ:
- Анализируй запрос пользователя
- Определи нужную функцию
- Подготовь правильные параметры
- Вызови функцию с корректными данными

⚠️ ВАЖНО: Не пытайся выполнить действие без функции!`,
	}

	// Project management API
	ProjectAPIModule = PromptModule{
		Name:     "project_api",
		Priority: 3,
		Content: `� ДОСТУПНЫЕ ФУНКЦИИ ПРОЕКТОВ:

� ПРОСМОТР ДАННЫХ:
- list_projects - получить список проектов (с фильтром по статусу)
- list_tasks - получить список задач (с фильтром по проекту/статусу)
- get_current_project - получить текущий активный проект

✏️ СОЗДАНИЕ И ИЗМЕНЕНИЕ:
- create_project - создать новый проект
- update_project - обновить существующий проект
- delete_project - удалить проект
- create_task - создать новую задачу
- update_task - обновить задачу
- delete_task - удалить задачу

🎛️ УПРАВЛЕНИЕ:
- set_current_project - установить текущий проект
- send_message_with_buttons - отправить сообщение с интерактивными кнопками`,
	}

	// Response formatting
	ResponseFormattingModule = PromptModule{
		Name:     "response_formatting",
		Priority: 4,
		Content: `💬 ПРАВИЛА ОФОРМЛЕНИЯ ОТВЕТОВ:

� СТРУКТУРА ОТВЕТА:
- Используй эмодзи для наглядности
- Группируй информацию логически
- Выделяй важные детали
- Предлагай следующие действия

🔢 ФОРМАТИРОВАНИЕ СПИСКОВ:
- Проекты: название, статус, количество задач
- Задачи: название, статус, приоритет, дедлайн
- Используй соответствующие эмодзи для статусов

🎯 ИНТЕРАКТИВНОСТЬ:
- Предлагай кнопки для частых действий
- Давай четкие инструкции
- Будь проактивным в предложениях`,
	}

	// Error handling
	ErrorHandlingModule = PromptModule{
		Name:     "error_handling",
		Priority: 5,
		Content: `🚨 ОБРАБОТКА ОШИБОК:

✅ ПРОВЕРЯЙ ДАННЫЕ:
- Существование проектов перед созданием задач
- Корректность ID при обновлении
- Права доступа пользователя

� ПОМОЩЬ ПОЛЬЗОВАТЕЛЮ:
- Объясняй ошибки простым языком
- Предлагай альтернативные действия
- Давай конкретные инструкции по исправлению

🔄 ВОССТАНОВЛЕНИЕ:
- Если действие невозможно - предложи альтернативы
- Используй кнопки для быстрого исправления
- Будь терпеливым и понимающим`,
	}

	// Context variables
	ContextModule = PromptModule{
		Name:     "context",
		Priority: 6,
		Content: `🔄 КОНТЕКСТНАЯ ИНФОРМАЦИЯ:

📊 ИСПОЛЬЗУЙ КОНТЕКСТ:
- Историю сообщений для понимания намерений
- Текущий проект для создания задач по умолчанию
- Предыдущие действия для улучшения UX

🎯 АДАПТИВНОСТЬ:
- Подстраивайся под стиль общения пользователя
- Запоминай предпочтения в рамках беседы
- Предлагай действия на основе контекста`,
	}
)

// BuildAdaptivePrompt creates a context-aware system prompt
func BuildAdaptivePrompt(ctx *PromptContext) string {
	var modules []PromptModule

	// Always include base modules
	modules = append(modules, BaseRoleModule, FunctionCallingRulesModule, ProjectAPIModule, ResponseFormattingModule, ErrorHandlingModule, ContextModule)

	// Add conditional modules based on context
	if ctx.Capability == "advanced" || ctx.Capability == "expert" {
		// Add InternetModule
	}

	// Add project-specific context
	if ctx.HasProjects && ctx.CurrentProject != nil {
		projectContextModule := PromptModule{
			Name:     "current_project",
			Priority: 7,
			Content: fmt.Sprintf(`🎯 ТЕКУЩИЙ ПРОЕКТ ПОЛЬЗОВАТЕЛЯ:
- ID: %d
- Название: %s
- Описание: %s
- Статус: %s
- Роль: %s

💡 При создании задач используй этот проект по умолчанию!`,
				ctx.CurrentProject.ID,
				ctx.CurrentProject.Title,
				ctx.CurrentProject.Description,
				ctx.CurrentProject.Status,
				ctx.CurrentProject.UserRole),
		}
		modules = append(modules, projectContextModule)
	}

	// Build final prompt
	var promptParts []string
	for _, module := range modules {
		promptParts = append(promptParts, module.Content)
	}

	return strings.Join(promptParts, "\n\n")
}

// Enhanced template prompts with better structure
const (
	// Improved welcome prompt with personalization
	WelcomePromptV2 = `Создай персонализированное приветствие для пользователя.

👤 ИНФОРМАЦИЯ О ПОЛЬЗОВАТЕЛЕ:
- Имя: %s
- Статус: %s (%s)
- Время: %s
- Проекты: %d
- Текущий проект: %s

🎯 ТРЕБОВАНИЯ:
- Используй эмодзи для дружелюбности
- Адаптируй сообщение под статус пользователя
- Если нет проектов - предложи создать первый
- Если есть проекты - покажи краткую сводку
- Будь мотивирующим и профессиональным`

	// Improved error handling prompt
	ErrorPromptV2 = `Создай умное сообщение об ошибке для пользователя.

🚨 КОНТЕКСТ ОШИБКИ:
- Тип: %s
- Описание: %s
- Пользователь пытался: %s
- Контекст: %s

🎯 ТРЕБОВАНИЯ:
- Объясни ошибку простыми словами
- Предложи конкретные действия для исправления
- Используй эмодзи для смягчения
- Добавь кнопки с альтернативными действиями
- Будь полезным и поддерживающим`

	// Smart project formatting prompt
	ProjectFormattingPromptV2 = `Создай красивый ответ о проектах пользователя.

📊 ДАННЫЕ:
- Проекты: %s
- Количество: %d
- Фильтр: %s
- Сортировка: %s

🎯 ТРЕБОВАНИЯ:
- Группируй по статусам с эмодзи
- Показывай прогресс задач
- Добавляй полезные кнопки действий
- Выделяй активный проект
- Предлагай следующие шаги`
)

// PromptValidator validates prompt quality
type PromptValidator struct {
	MaxLength      int
	RequiredWords  []string
	ForbiddenWords []string
}

// ValidatePrompt checks prompt quality and returns suggestions
func (v *PromptValidator) ValidatePrompt(prompt string) (bool, []string) {
	var issues []string

	// Check length
	if len(prompt) > v.MaxLength {
		issues = append(issues, fmt.Sprintf("Промпт слишком длинный: %d символов (максимум %d)", len(prompt), v.MaxLength))
	}

	// Check for required words
	for _, word := range v.RequiredWords {
		if !strings.Contains(strings.ToLower(prompt), strings.ToLower(word)) {
			issues = append(issues, fmt.Sprintf("Отсутствует обязательное слово: %s", word))
		}
	}

	// Check for forbidden words
	for _, word := range v.ForbiddenWords {
		if strings.Contains(strings.ToLower(prompt), strings.ToLower(word)) {
			issues = append(issues, fmt.Sprintf("Содержит запрещенное слово: %s", word))
		}
	}

	return len(issues) == 0, issues
}

// PromptMetrics tracks prompt performance
type PromptMetrics struct {
	PromptID     string
	UserID       int
	UsageCount   int
	SuccessRate  float64
	AvgTokens    int
	AvgResponse  time.Duration
	LastUsed     time.Time
	ErrorRate    float64
	UserRating   float64
}

// PromptOptimizer optimizes prompts based on usage data
type PromptOptimizer struct {
	metrics map[string]*PromptMetrics
}

// NewPromptOptimizer creates a new prompt optimizer
func NewPromptOptimizer() *PromptOptimizer {
	return &PromptOptimizer{
		metrics: make(map[string]*PromptMetrics),
	}
}

// TrackUsage tracks prompt usage for optimization
func (o *PromptOptimizer) TrackUsage(promptID string, userID int, success bool, tokens int, responseTime time.Duration) {
	if o.metrics[promptID] == nil {
		o.metrics[promptID] = &PromptMetrics{
			PromptID: promptID,
			UserID:   userID,
		}
	}

	metric := o.metrics[promptID]
	metric.UsageCount++
	metric.LastUsed = time.Now()
	metric.AvgTokens = (metric.AvgTokens + tokens) / 2
	metric.AvgResponse = (metric.AvgResponse + responseTime) / 2

	if success {
		metric.SuccessRate = (metric.SuccessRate*float64(metric.UsageCount-1) + 1) / float64(metric.UsageCount)
	} else {
		metric.ErrorRate = (metric.ErrorRate*float64(metric.UsageCount-1) + 1) / float64(metric.UsageCount)
	}
}

// GetOptimizationSuggestions returns suggestions for prompt improvement
func (o *PromptOptimizer) GetOptimizationSuggestions(promptID string) []string {
	metric := o.metrics[promptID]
	if metric == nil {
		return []string{"Недостаточно данных для анализа"}
	}

	var suggestions []string

	if metric.SuccessRate < 0.8 {
		suggestions = append(suggestions, "Низкий процент успешных ответов - упростите инструкции")
	}

	if metric.AvgTokens > 1000 {
		suggestions = append(suggestions, "Слишком много токенов - сократите промпт")
	}

	if metric.AvgResponse > 10*time.Second {
		suggestions = append(suggestions, "Долгое время ответа - оптимизируйте промпт")
	}

	if metric.ErrorRate > 0.2 {
		suggestions = append(suggestions, "Высокий процент ошибок - добавьте примеры")
	}

	return suggestions
}

// Enhanced system prompt builder with version control
func GetSystemPromptV2(ctx *PromptContext) string {
	if ctx == nil {
		// Fallback to basic prompt
		return BuildAdaptivePrompt(&PromptContext{
			Capability: "basic",
		})
	}

	return BuildAdaptivePrompt(ctx)
}

// CreatePromptContext creates context for adaptive prompts
func CreatePromptContext(db *DB, userID int) *PromptContext {
	ctx := &PromptContext{
		UserID:     userID,
		Capability: "basic",
	}

	// Get user's projects
	projects, err := db.GetUserProjects(userID)
	if err == nil {
		ctx.HasProjects = len(projects) > 0
		if len(projects) > 3 {
			ctx.Capability = "advanced"
		}
		if len(projects) > 10 {
			ctx.Capability = "expert"
		}
	}

	// Get current project
	currentProject, err := db.GetUserCurrentProject(userID)
	if err == nil && currentProject != nil {
		ctx.CurrentProject = currentProject
	}

	return ctx
}

// Test function for prompt validation
func TestPromptQuality() map[string][]string {
	validator := &PromptValidator{
		MaxLength:      3000,
		RequiredWords:  []string{"помощник", "функции"},
		ForbiddenWords: []string{},
	}

	results := make(map[string][]string)

	// Test base modules
	modules := []PromptModule{BaseRoleModule, FunctionCallingRulesModule, ProjectAPIModule}
	for _, module := range modules {
		valid, issues := validator.ValidatePrompt(module.Content)
		if !valid {
			results[module.Name] = issues
		}
	}

	return results
}