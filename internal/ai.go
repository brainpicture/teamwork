package internal

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/sashabaranov/go-openai"
	anthropic "github.com/unfunco/anthropic-sdk-go"
)

// AIProvider interface for different AI providers
type AIProvider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	GenerateResponseWithContext(ctx context.Context, prompt string, history []*Message) (string, error)
	GenerateWelcomeMessage(ctx context.Context, userName, status, timestamp string) (string, error)
	GenerateErrorMessage(ctx context.Context, errorContext string) (string, error)
	TranscribeAudio(ctx context.Context, audioData io.Reader, filename string) (string, error)
	GenerateResponseWithContextAndProject(ctx context.Context, prompt string, history []*Message, currentProject *Project) (string, error)
}

// OpenAIProvider implementation for OpenAI ChatGPT
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// ClaudeProvider implementation for Anthropic Claude
type ClaudeProvider struct {
	client *anthropic.Client
	model  string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	client := openai.NewClient(apiKey)
	return &OpenAIProvider{
		client: client,
		model:  openai.GPT4o, // Using GPT-4o as requested
	}
}

// NewClaudeProvider creates a new Anthropic Claude provider
func NewClaudeProvider(apiKey string) *ClaudeProvider {
	transport := &anthropic.Transport{APIKey: apiKey}
	client := anthropic.NewClient(transport.Client())
	return &ClaudeProvider{
		client: client,
		model:  string(anthropic.Claude3Opus20240229), // Using Claude-3 Opus (most powerful available)
	}
}

// TranscribeAudio transcribes audio using OpenAI Whisper API
func (p *OpenAIProvider) TranscribeAudio(ctx context.Context, audioData io.Reader, filename string) (string, error) {
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   audioData,
		FilePath: filename, // Add filename so OpenAI can determine format
	}

	resp, err := p.client.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Whisper API error: %v", err)
	}

	log.Printf("Audio transcribed successfully: %d characters", len(resp.Text))
	return resp.Text, nil
}

// GenerateResponse generates a response using OpenAI ChatGPT
func (p *OpenAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// Get available functions
	openAIFunctions := GetGPTFunctions()

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: p.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: GetSystemPrompt(),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Functions:   openAIFunctions,
			MaxTokens:   500,
			Temperature: 0.7,
		},
	)

	if err != nil {
		return "", fmt.Errorf("ChatGPT API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from ChatGPT")
	}

	choice := resp.Choices[0]

	// Check if GPT wants to call a function
	if choice.Message.FunctionCall != nil {
		return "", fmt.Errorf("function_call:%s:%s", choice.Message.FunctionCall.Name, choice.Message.FunctionCall.Arguments)
	}

	response := choice.Message.Content
	log.Printf("AI Response generated: %d characters", len(response))
	return response, nil
}

// GenerateWelcomeMessage generates a personalized welcome message
func (p *OpenAIProvider) GenerateWelcomeMessage(ctx context.Context, userName, status, timestamp string) (string, error) {
	prompt := fmt.Sprintf(WelcomePromptTemplate, userName, status, timestamp)
	return p.GenerateResponse(ctx, prompt)
}

// GenerateErrorMessage generates a user-friendly error message
func (p *OpenAIProvider) GenerateErrorMessage(ctx context.Context, errorContext string) (string, error) {
	prompt := fmt.Sprintf(ErrorPromptTemplate, errorContext)
	return p.GenerateResponse(ctx, prompt)
}

// GenerateResponseWithContext generates a response using OpenAI ChatGPT with conversation history
func (p *OpenAIProvider) GenerateResponseWithContext(ctx context.Context, prompt string, history []*Message) (string, error) {
	// Get available functions
	openAIFunctions := GetGPTFunctions()

	// Build message history
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: GetSystemPrompt(),
		},
	}

	// Add conversation history
	for _, msg := range history {
		role := openai.ChatMessageRoleUser
		if msg.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	})

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       p.model,
			Messages:    messages,
			Functions:   openAIFunctions,
			MaxTokens:   500,
			Temperature: 0.7,
		},
	)

	if err != nil {
		return "", fmt.Errorf("ChatGPT API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from ChatGPT")
	}

	choice := resp.Choices[0]

	// Check if GPT wants to call a function
	if choice.Message.FunctionCall != nil {
		return "", fmt.Errorf("function_call:%s:%s", choice.Message.FunctionCall.Name, choice.Message.FunctionCall.Arguments)
	}

	response := choice.Message.Content
	log.Printf("AI Response with context generated: %d characters, history: %d messages", len(response), len(history))
	return response, nil
}

// GenerateResponseWithContextAndProject generates a response using OpenAI ChatGPT with conversation history and current project context
func (p *OpenAIProvider) GenerateResponseWithContextAndProject(ctx context.Context, prompt string, history []*Message, currentProject *Project) (string, error) {
	// Get available functions
	openAIFunctions := GetGPTFunctions()

	// Build enhanced system prompt with current project info
	systemPrompt := GetSystemPrompt()
	if currentProject != nil {
		projectInfo := fmt.Sprintf("\n\nТЕКУЩИЙ ПРОЕКТ ПОЛЬЗОВАТЕЛЯ:\n- ID: %d\n- Название: %s\n- Описание: %s\n- Статус: %s\n- Роль пользователя: %s\n\nПри создании задач используй этот проект по умолчанию, если пользователь не указал другой проект явно.",
			currentProject.ID, currentProject.Title, currentProject.Description, currentProject.Status, currentProject.UserRole)
		systemPrompt += projectInfo
	}

	// Build message history
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	// Add conversation history
	for _, msg := range history {
		role := openai.ChatMessageRoleUser
		if msg.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	})

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       p.model,
			Messages:    messages,
			Functions:   openAIFunctions,
			MaxTokens:   500,
			Temperature: 0.7,
		},
	)

	if err != nil {
		return "", fmt.Errorf("ChatGPT API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from ChatGPT")
	}

	choice := resp.Choices[0]

	// Check if GPT wants to call a function
	if choice.Message.FunctionCall != nil {
		return "", fmt.Errorf("function_call:%s:%s", choice.Message.FunctionCall.Name, choice.Message.FunctionCall.Arguments)
	}

	response := choice.Message.Content
	log.Printf("AI Response with project context generated: %d characters, project: %s", len(response), currentProject.Title)
	return response, nil
}

// AIService manages AI providers and provides high-level AI functionality
type AIService struct {
	provider AIProvider
	enabled  bool
}

// NewAIService creates a new AI service
func NewAIService(provider AIProvider, enabled bool) *AIService {
	return &AIService{
		provider: provider,
		enabled:  enabled,
	}
}

// IsEnabled returns whether AI service is enabled
func (s *AIService) IsEnabled() bool {
	return s.enabled && s.provider != nil
}

// TranscribeAudio transcribes audio if enabled, otherwise returns error
func (s *AIService) TranscribeAudio(ctx context.Context, audioData io.Reader, filename string) (string, error) {
	if !s.IsEnabled() {
		return "", fmt.Errorf("AI service is disabled")
	}

	return s.provider.TranscribeAudio(ctx, audioData, filename)
}

// GenerateResponse generates an AI response if enabled, otherwise returns fallback
func (s *AIService) GenerateResponse(ctx context.Context, prompt string, fallback string) string {
	if !s.IsEnabled() {
		log.Printf("AI service disabled, using fallback response")
		return fallback
	}

	response, err := s.provider.GenerateResponse(ctx, prompt)
	if err != nil {
		log.Printf("AI generation failed, using fallback: %v", err)
		return fallback
	}

	return response
}

// GenerateWelcomeMessage generates a welcome message or returns fallback
func (s *AIService) GenerateWelcomeMessage(ctx context.Context, userName, status, timestamp, fallback string) string {
	if !s.IsEnabled() {
		return fallback
	}

	response, err := s.provider.GenerateWelcomeMessage(ctx, userName, status, timestamp)
	if err != nil {
		log.Printf("AI welcome generation failed, using fallback: %v", err)
		return fallback
	}

	return response
}

// GenerateResponseWithContext generates an AI response with conversation history if enabled, otherwise returns fallback
func (s *AIService) GenerateResponseWithContext(ctx context.Context, prompt string, history []*Message, fallback string) (string, error) {
	if !s.IsEnabled() {
		return fallback, nil
	}

	response, err := s.provider.GenerateResponseWithContext(ctx, prompt, history)
	if err != nil {
		return "", err
	}

	return response, nil
}

// GenerateResponseWithContextAndProject generates a response with conversation history and current project context
func (s *AIService) GenerateResponseWithContextAndProject(ctx context.Context, prompt string, history []*Message, currentProject *Project, fallback string) (string, error) {
	if !s.IsEnabled() {
		return fallback, nil
	}

	// Check if provider supports project context
	if provider, ok := s.provider.(*OpenAIProvider); ok {
		response, err := provider.GenerateResponseWithContextAndProject(ctx, prompt, history, currentProject)
		if err != nil {
			return "", err
		}
		return response, nil
	}

	// Fallback to regular context if provider doesn't support project context
	response, err := s.provider.GenerateResponseWithContext(ctx, prompt, history)
	if err != nil {
		return "", err
	}

	return response, nil
}

// FormatDataResponse uses GPT to format raw data response
func (s *AIService) FormatDataResponse(ctx context.Context, userQuery string, functionType string, jsonData string) (string, error) {
	if !s.enabled {
		return fmt.Sprintf("Данные получены: %s", jsonData), nil
	}

	// Build special prompt for data formatting
	prompt := fmt.Sprintf(`Пользователь запросил: "%s"

Функция %s вернула следующие данные в JSON:
%s

Твоя задача:
1. Проанализировать данные
2. Создать красивый, информативный ответ для пользователя
3. ОБЯЗАТЕЛЬНО используй send_message_with_buttons если это уместно:
   - Если нет данных (пустой список) - добавь полезные кнопки для создания/навигации
   - Если есть данные - добавь кнопки для дальнейших действий
   - Кнопки должны содержать конкретные действия, которые пользователь может выполнить

Отформатируй ответ с эмодзи, сделай его удобным для чтения.
Если список пуст, обязательно предложи альтернативные действия через кнопки.`, userQuery, functionType, jsonData)

	// Get available functions
	openAIFunctions := GetGPTFunctions()

	resp, err := s.provider.(*OpenAIProvider).client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: s.provider.(*OpenAIProvider).model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: GetSystemPrompt(),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Functions:   openAIFunctions,
			MaxTokens:   500,
			Temperature: 0.7,
		},
	)

	if err != nil {
		return "", fmt.Errorf("ChatGPT API error: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from ChatGPT")
	}

	choice := resp.Choices[0]

	// Check if GPT wants to call a function
	if choice.Message.FunctionCall != nil {
		return fmt.Sprintf("function_call:%s:%s", choice.Message.FunctionCall.Name, choice.Message.FunctionCall.Arguments), nil
	}

	response := choice.Message.Content
	log.Printf("AI Data formatting response generated: %d characters", len(response))
	return response, nil
}

// GenerateResponse generates a response using Anthropic Claude
func (p *ClaudeProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	resp, _, err := p.client.Messages.Create(ctx, &anthropic.CreateMessageInput{
		Model:     anthropic.LanguageModel(p.model),
		MaxTokens: 500,
		System:    GetSystemPrompt(),
		Messages: []anthropic.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: &[]float64{0.7}[0],
	})

	if err != nil {
		return "", fmt.Errorf("Claude API error: %v", err)
	}

	if len(resp.Content) == 0 {
		return "", fmt.Errorf("no response from Claude")
	}

	response := resp.Content[0].Text
	log.Printf("Claude Response generated: %d characters", len(response))
	return response, nil
}

// GenerateWelcomeMessage generates a personalized welcome message
func (p *ClaudeProvider) GenerateWelcomeMessage(ctx context.Context, userName, status, timestamp string) (string, error) {
	prompt := fmt.Sprintf(WelcomePromptTemplate, userName, status, timestamp)
	return p.GenerateResponse(ctx, prompt)
}

// GenerateErrorMessage generates a user-friendly error message
func (p *ClaudeProvider) GenerateErrorMessage(ctx context.Context, errorContext string) (string, error) {
	prompt := fmt.Sprintf(ErrorPromptTemplate, errorContext)
	return p.GenerateResponse(ctx, prompt)
}

// TranscribeAudio - Claude doesn't support audio transcription, fallback to OpenAI
func (p *ClaudeProvider) TranscribeAudio(ctx context.Context, audioData io.Reader, filename string) (string, error) {
	return "", fmt.Errorf("audio transcription not supported by Claude provider - use OpenAI Whisper")
}

// GenerateResponseWithContext generates a response using Anthropic Claude with conversation history
func (p *ClaudeProvider) GenerateResponseWithContext(ctx context.Context, prompt string, history []*Message) (string, error) {
	// Build message history
	messages := []anthropic.Message{}

	// Add conversation history
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		}

		messages = append(messages, anthropic.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, anthropic.Message{
		Role:    "user",
		Content: prompt,
	})

	resp, _, err := p.client.Messages.Create(ctx, &anthropic.CreateMessageInput{
		Model:       anthropic.LanguageModel(p.model),
		MaxTokens:   500,
		System:      GetSystemPrompt(),
		Messages:    messages,
		Temperature: &[]float64{0.7}[0],
	})

	if err != nil {
		return "", fmt.Errorf("Claude API error: %v", err)
	}

	if len(resp.Content) == 0 {
		return "", fmt.Errorf("no response from Claude")
	}

	response := resp.Content[0].Text
	log.Printf("Claude Response with context generated: %d characters, history: %d messages", len(response), len(history))
	return response, nil
}

// GenerateResponseWithContextAndProject generates a response using Anthropic Claude with conversation history and current project context
func (p *ClaudeProvider) GenerateResponseWithContextAndProject(ctx context.Context, prompt string, history []*Message, currentProject *Project) (string, error) {
	// Build enhanced system prompt with current project info
	systemPrompt := GetSystemPrompt()
	if currentProject != nil {
		projectInfo := fmt.Sprintf("\n\nТЕКУЩИЙ ПРОЕКТ ПОЛЬЗОВАТЕЛЯ:\n- ID: %d\n- Название: %s\n- Описание: %s\n- Статус: %s\n- Роль пользователя: %s\n\nПри создании задач используй этот проект по умолчанию, если пользователь не указал другой проект явно.",
			currentProject.ID, currentProject.Title, currentProject.Description, currentProject.Status, currentProject.UserRole)
		systemPrompt += projectInfo
	}

	// Build message history
	messages := []anthropic.Message{}

	// Add conversation history
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		}

		messages = append(messages, anthropic.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, anthropic.Message{
		Role:    "user",
		Content: prompt,
	})

	resp, _, err := p.client.Messages.Create(ctx, &anthropic.CreateMessageInput{
		Model:       anthropic.LanguageModel(p.model),
		MaxTokens:   500,
		System:      systemPrompt,
		Messages:    messages,
		Temperature: &[]float64{0.7}[0],
	})

	if err != nil {
		return "", fmt.Errorf("Claude API error: %v", err)
	}

	if len(resp.Content) == 0 {
		return "", fmt.Errorf("no response from Claude")
	}

	response := resp.Content[0].Text
	log.Printf("Claude Response with context and project generated: %d characters, history: %d messages", len(response), len(history))
	return response, nil
}
