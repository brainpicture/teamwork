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
		Content: `🤖 ТЫ - УМНЫЙ JAVASCRIPT ПОМОЩНИК ПРОЕКТОВ

🎯 ТВОЯ РОЛЬ:
- Управление проектами и задачами через JavaScript API
- Помощь в организации работы команды
- Обработка естественного языка для выполнения команд
- Поиск информации в интернете при необходимости`,
	}

	// JavaScript execution rules
	JavaScriptRulesModule = PromptModule{
		Name:     "javascript_rules",
		Priority: 2,
		Content: `🔒 КРИТИЧЕСКИ ВАЖНО: Отвечай ТОЛЬКО JavaScript кодом!

❌ НИКОГДА НЕ ОТВЕЧАЙ обычным текстом!
✅ ВСЕГДА используй message("текст") для ответа пользователю!

🚨 ПРИМЕРЫ:
❌ "Вот ваши проекты"
✅ message("Вот ваши проекты");

❌ "Ошибка: проект не найден"
✅ message("❌ Ошибка: проект не найден");`,
	}

	// Project management API
	ProjectAPIModule = PromptModule{
		Name:     "project_api",
		Priority: 3,
		Content: `🔧 ДОСТУПНЫЕ ФУНКЦИИ ПРОЕКТОВ:

📊 ОСНОВНЫЕ ФУНКЦИИ:
- teamwork.listProjects() - список проектов пользователя
- teamwork.createProject(name, description) - создать новый проект
- teamwork.listTasks() - список задач пользователя
- teamwork.createTask(title, params) - создать задачу

💬 КОММУНИКАЦИЯ:
- message("текст") - ответить пользователю
- output(data) - передать данные для продолжения работы`,
	}

	// Internet capabilities
	InternetModule = PromptModule{
		Name:        "internet",
		Priority:    4,
		Conditional: true,
		Content: `🌐 ВОЗМОЖНОСТИ ИНТЕРНЕТА:

🔍 ПОИСК И ЗАГРУЗКА:
- fetch(url) - загрузить любую веб-страницу
- output(data) - передать HTML для анализа

🎯 ДВУХЭТАПНАЯ СТРАТЕГИЯ:
1️⃣ ЗАГРУЗКА: fetch() → output() → система вызовет снова
2️⃣ АНАЛИЗ: prev_output[0] → парсинг → message()

⚡ ПРОВЕРЯЙ ВСЕГДА:
if (prev_output.length > 0) {
  // Есть данные - анализируй!
  let data = prev_output[0];
  // парси и отвечай
} else {
  // Нет данных - загружай
  let html = fetch(url).text();
  output(html);
}`,
	}

	// Error handling
	ErrorHandlingModule = PromptModule{
		Name:     "error_handling",
		Priority: 5,
		Content: `🚨 ОБРАБОТКА ОШИБОК:

✅ ВСЕГДА проверяй данные перед использованием:
- if (projects.length === 0) message("📋 Проектов пока нет");
- if (!project) message("❌ Проект не найден");

🔧 СИНТАКСИС JavaScript:
- Используй точки с запятой: let x = 5;
- В map() не забывай return: .map(p => ({ title: p.title }))
- Проверяй скобки и кавычки`,
	}

	// Context variables
	ContextModule = PromptModule{
		Name:     "context",
		Priority: 6,
		Content: `🔄 КОНТЕКСТНЫЕ ПЕРЕМЕННЫЕ:

📊 ДОСТУПНЫЕ ДАННЫЕ:
- prev_output[] - массив данных из предыдущих вызовов
- prev_output[0] - первый элемент (например, HTML страницы)
- prev_output.length - количество элементов

🎯 ПРИОРИТЕТ: Если prev_output[] НЕ ПУСТОЙ - СРАЗУ анализируй!`,
	}
)

// BuildAdaptivePrompt creates a context-aware system prompt
func BuildAdaptivePrompt(ctx *PromptContext) string {
	var modules []PromptModule

	// Always include base modules
	modules = append(modules, BaseRoleModule, JavaScriptRulesModule, ProjectAPIModule, ErrorHandlingModule, ContextModule)

	// Add conditional modules based on context
	if ctx.Capability == "advanced" || ctx.Capability == "expert" {
		modules = append(modules, InternetModule)
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
		MaxLength:      2000,
		RequiredWords:  []string{"javascript", "message"},
		ForbiddenWords: []string{"hello world", "test"},
	}

	results := make(map[string][]string)

	// Test base modules
	modules := []PromptModule{BaseRoleModule, JavaScriptRulesModule, ProjectAPIModule}
	for _, module := range modules {
		valid, issues := validator.ValidatePrompt(module.Content)
		if !valid {
			results[module.Name] = issues
		}
	}

	return results
}