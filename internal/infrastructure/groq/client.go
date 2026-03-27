package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with Groq AI API
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	maxRetries int
}

// Config holds Groq client configuration
type Config struct {
	APIKey         string
	Model          string
	BaseURL        string
	TimeoutSeconds int
	MaxRetries     int
}

// chatRequest represents the Groq API request structure
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

// message represents a chat message
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse represents the Groq API response structure
type chatResponse struct {
	Choices []choice `json:"choices"`
	Error   *apiError `json:"error,omitempty"`
}

// choice represents a response choice
type choice struct {
	Message message `json:"message"`
}

// apiError represents an API error
type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// NewClient creates a new Groq AI client
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.groq.com/openai/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "llama3-70b-8192"
	}
	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = 30
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return &Client{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
		maxRetries: cfg.MaxRetries,
	}
}

// ScoreAnswer scores an essay answer using Groq API
func (c *Client) ScoreAnswer(ctx context.Context, question, expectedAnswer, actualAnswer string) (float64, string, error) {
	prompt := c.buildScoringPrompt(question, expectedAnswer, actualAnswer)

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		result, err := c.callAPI(ctx, prompt)
		if err != nil {
			lastErr = err
			continue
		}

		// Parse the result
		score, feedback, err := c.parseScoreResult(result)
		if err != nil {
			lastErr = err
			continue
		}

		return score, feedback, nil
	}

	return 0, "", fmt.Errorf("failed after %d attempts: %w", c.maxRetries, lastErr)
}

// buildScoringPrompt creates the scoring prompt for the AI
func (c *Client) buildScoringPrompt(question, expectedAnswer, actualAnswer string) string {
	return fmt.Sprintf(`You are an expert grader for essay questions. Score the following answer on a scale of 0-100.

Question: %s

Expected Answer (Key Points): %s

Student's Answer: %s

Please provide:
1. A score from 0-100
2. Brief feedback explaining the score

Format your response as JSON:
{
  "score": <number>,
  "feedback": "<string>"
}

Be fair and constructive in your feedback. Consider:
- Correctness of information
- Completeness of the answer
- Clarity of explanation
- Coverage of key points from the expected answer`, question, expectedAnswer, actualAnswer)
}

// callAPI makes the API call to Groq
func (c *Client) callAPI(ctx context.Context, prompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 500,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr chatResponse
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error != nil {
			return "", fmt.Errorf("API error: %s", apiErr.Error.Message)
		}
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// parseScoreResult parses the AI response into score and feedback
func (c *Client) parseScoreResult(response string) (float64, string, error) {
	var result struct {
		Score    float64 `json:"score"`
		Feedback string  `json:"feedback"`
	}

	// Try to parse as JSON
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If parsing fails, try to extract from text
		// This is a fallback for cases where the AI doesn't return pure JSON
		return c.extractScoreFromText(response)
	}

	// Validate score range
	if result.Score < 0 || result.Score > 100 {
		return 0, "", fmt.Errorf("score out of range: %.2f", result.Score)
	}

	return result.Score, result.Feedback, nil
}

// extractScoreFromText attempts to extract score and feedback from unstructured text
func (c *Client) extractScoreFromText(text string) (float64, string, error) {
	// This is a simple fallback - in production, you might want more sophisticated parsing
	// For now, return an error to force proper JSON responses
	return 0, "", fmt.Errorf("failed to parse response as JSON: %s", text)
}
