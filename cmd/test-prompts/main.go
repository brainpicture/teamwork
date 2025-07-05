package main

import (
	"fmt"
	"strings"
	"time"

	"telegram-bot/internal"
)

func main() {
	fmt.Println("🔍 Тестирование улучшенной системы промптов")
	fmt.Println(strings.Repeat("=", 50))

	// Тест 1: Проверка определений функций OpenAI
	fmt.Println("\n📋 1. Проверка определений функций OpenAI")
	testGPTFunctions()

	// Тест 2: Проверка модульных промптов
	fmt.Println("\n🧩 2. Проверка модульных промптов")
	testModularPrompts()

	// Тест 3: Валидация качества промптов
	fmt.Println("\n✅ 3. Валидация качества промптов")
	testPromptValidation()

	// Тест 4: Адаптивные промпты
	fmt.Println("\n🎯 4. Тестирование адаптивных промптов")
	testAdaptivePrompts()

	// Тест 5: Метрики производительности
	fmt.Println("\n📊 5. Тестирование метрик")
	testPromptMetrics()

	fmt.Println("\n✅ Все тесты завершены!")
}

func testGPTFunctions() {
	functions := internal.GetGPTFunctions()
	fmt.Printf("   Найдено функций: %d\n", len(functions))

	requiredFunctions := []string{
		"list_projects", "create_project", "update_project", "delete_project",
		"list_tasks", "create_task", "update_task", "delete_task",
		"set_current_project", "get_current_project", "send_message_with_buttons",
	}

	functionMap := make(map[string]bool)
	for _, fn := range functions {
		functionMap[fn.Name] = true
		fmt.Printf("   ✓ %s: %s\n", fn.Name, fn.Description)
	}

	missing := []string{}
	for _, required := range requiredFunctions {
		if !functionMap[required] {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		fmt.Printf("   ❌ Отсутствующие функции: %v\n", missing)
	} else {
		fmt.Println("   ✅ Все необходимые функции определены")
	}
}

func testModularPrompts() {
	// Создаем разные контексты для тестирования
	contexts := []*internal.PromptContext{
		{UserID: 1, Capability: "basic", HasProjects: false},
		{UserID: 2, Capability: "advanced", HasProjects: true},
		{UserID: 3, Capability: "expert", HasProjects: true, CurrentProject: &internal.Project{
			ID: 1, Title: "Тестовый проект", Description: "Описание", Status: "active", UserRole: "owner",
		}},
	}

	for i, ctx := range contexts {
		fmt.Printf("   Контекст %d (%s, проекты: %t):\n", i+1, ctx.Capability, ctx.HasProjects)
		prompt := internal.GetSystemPromptV2(ctx)
		
		lines := strings.Split(prompt, "\n")
		fmt.Printf("     Строк в промпте: %d\n", len(lines))
		fmt.Printf("     Символов: %d\n", len(prompt))
		
		// Проверяем наличие ключевых слов
		keywords := []string{"функции", "проект", "задач", "помощник"}
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(prompt), keyword) {
				fmt.Printf("     ✓ Содержит: %s\n", keyword)
			}
		}
		fmt.Println()
	}
}

func testPromptValidation() {
	validator := &internal.PromptValidator{
		MaxLength:      6000,
		RequiredWords:  []string{"помощник", "функции"},
		ForbiddenWords: []string{},
	}

	// Тестируем базовый промпт
	basicPrompt := internal.GetSystemPromptV2(&internal.PromptContext{Capability: "basic"})
	valid, issues := validator.ValidatePrompt(basicPrompt)

	fmt.Printf("   Базовый промпт валиден: %t\n", valid)
	if !valid {
		fmt.Println("   Проблемы:")
		for _, issue := range issues {
			fmt.Printf("     - %s\n", issue)
		}
	}

	// Тестируем модули
	fmt.Println("   Тестирование модулей:")
	results := internal.TestPromptQuality()
	if len(results) == 0 {
		fmt.Println("     ✅ Все модули прошли валидацию")
	} else {
		for module, issues := range results {
			fmt.Printf("     ❌ %s: %v\n", module, issues)
		}
	}
}

func testAdaptivePrompts() {
	scenarios := []struct {
		name    string
		context *internal.PromptContext
	}{
		{
			"Новый пользователь",
			&internal.PromptContext{UserID: 1, Capability: "basic", HasProjects: false},
		},
		{
			"Опытный пользователь с проектами",
			&internal.PromptContext{UserID: 2, Capability: "advanced", HasProjects: true},
		},
		{
			"Эксперт с текущим проектом",
			&internal.PromptContext{
				UserID: 3, Capability: "expert", HasProjects: true,
				CurrentProject: &internal.Project{ID: 1, Title: "Проект", Status: "active"},
			},
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("   %s:\n", scenario.name)
		prompt := internal.GetSystemPromptV2(scenario.context)
		
		// Анализируем адаптивность
		if scenario.context.HasProjects && strings.Contains(prompt, "ТЕКУЩИЙ ПРОЕКТ") {
			fmt.Println("     ✓ Содержит информацию о текущем проекте")
		}
		
		if scenario.context.Capability == "expert" && len(prompt) > 1000 {
			fmt.Println("     ✓ Расширенный промпт для эксперта")
		}
		
		fmt.Printf("     Длина промпта: %d символов\n", len(prompt))
		fmt.Println()
	}
}

func testPromptMetrics() {
	optimizer := internal.NewPromptOptimizer()
	
	// Симулируем использование промптов
	testData := []struct {
		promptID     string
		userID       int
		success      bool
		tokens       int
		responseTime time.Duration
	}{
		{"basic_prompt", 1, true, 150, 2 * time.Second},
		{"basic_prompt", 1, true, 200, 3 * time.Second},
		{"basic_prompt", 2, false, 300, 8 * time.Second},
		{"advanced_prompt", 3, true, 400, 5 * time.Second},
		{"advanced_prompt", 3, true, 350, 4 * time.Second},
	}

	for _, data := range testData {
		optimizer.TrackUsage(data.promptID, data.userID, data.success, data.tokens, data.responseTime)
	}

	// Получаем предложения по оптимизации
	fmt.Println("   Предложения по оптимизации:")
	for _, promptID := range []string{"basic_prompt", "advanced_prompt"} {
		suggestions := optimizer.GetOptimizationSuggestions(promptID)
		fmt.Printf("     %s:\n", promptID)
		for _, suggestion := range suggestions {
			fmt.Printf("       - %s\n", suggestion)
		}
	}
}

