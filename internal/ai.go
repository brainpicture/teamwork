package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/sashabaranov/go-openai"
)

// AIProvider interface for different AI providers
type AIProvider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	GenerateWelcomeMessage(ctx context.Context, userName, status, timestamp string) (string, error)
	GenerateErrorMessage(ctx context.Context, errorContext string) (string, error)
}

// OpenAIProvider implementation for OpenAI ChatGPT
type OpenAIProvider struct {
	client *openai.Client
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

// GenerateResponse generates a response using OpenAI ChatGPT
func (p *OpenAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: p.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: SystemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
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

	response := resp.Choices[0].Message.Content
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
