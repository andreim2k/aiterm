package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andreim2k/aiterm/config"
	"github.com/andreim2k/aiterm/logger"
)

const (
	// DefaultHTTPTimeout is the default timeout for HTTP requests to AI APIs
	DefaultHTTPTimeout = 60 * time.Second
)

// AiClient represents an AI client for interacting with OpenAI-compatible APIs including Azure OpenAI
type AiClient struct {
	config    *config.Config
	configMgr *Manager // To access model configuration methods
	client    *http.Client
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents a request to the chat completion API
type ChatCompletionRequest struct {
	Model    string    `json:"model,omitempty"`
	Messages []Message `json:"messages"`
}

// ChatCompletionChoice represents a choice in the chat completion response
type ChatCompletionChoice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

// ChatCompletionResponse represents a response from the chat completion API
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Choices []ChatCompletionChoice `json:"choices"`
}

// Responses API Types

// ResponseInput represents the input for the Responses API
type ResponseInput interface{}

// ResponseContent represents content in the Responses API
type ResponseContent struct {
	Type        string        `json:"type"`
	Text        string        `json:"text,omitempty"`
	Annotations []interface{} `json:"annotations,omitempty"`
}

// ResponseOutputItem represents an output item in the Responses API
type ResponseOutputItem struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`             // "message", "reasoning", "function_call", etc.
	Status  string            `json:"status,omitempty"` // "completed", "in_progress", etc.
	Content []ResponseContent `json:"content,omitempty"`
	Role    string            `json:"role,omitempty"` // "assistant", "user", etc.
	Summary []interface{}     `json:"summary,omitempty"`
}

// ResponseRequest represents a request to the Responses API
type ResponseRequest struct {
	Model              string                 `json:"model"`
	Input              ResponseInput          `json:"input"`
	Instructions       string                 `json:"instructions,omitempty"`
	Tools              []interface{}          `json:"tools,omitempty"`
	PreviousResponseID string                 `json:"previous_response_id,omitempty"`
	Store              bool                   `json:"store,omitempty"`
	Include            []string               `json:"include,omitempty"`
	Text               map[string]interface{} `json:"text,omitempty"` // for structured outputs
}

// Response represents a response from the Responses API
type Response struct {
	ID         string               `json:"id"`
	Object     string               `json:"object"`
	CreatedAt  int64                `json:"created_at"`
	Model      string               `json:"model"`
	Output     []ResponseOutputItem `json:"output"`
	OutputText string               `json:"output_text,omitempty"`
	Error      *ResponseError       `json:"error,omitempty"`
	Usage      *ResponseUsage       `json:"usage,omitempty"`
}

// ResponseError represents an error in the Responses API
type ResponseError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// ResponseUsage represents token usage in the Responses API
type ResponseUsage struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	TotalTokens     int `json:"total_tokens"`
}

func NewAiClient(cfg *config.Config) *AiClient {
	return &AiClient{
		config: cfg,
		client: &http.Client{
			Timeout: DefaultHTTPTimeout,
		},
	}
}

// SetConfigManager sets the configuration manager for accessing model configurations
func (c *AiClient) SetConfigManager(mgr *Manager) {
	c.configMgr = mgr
}

// determineAPIType determines which API to use based on the model and configuration
func (c *AiClient) determineAPIType(model string) string {
	// If we have a config manager, try to get the current model configuration
	if c.configMgr != nil {
		if modelConfig, exists := c.configMgr.GetCurrentModelConfig(); exists {
			switch modelConfig.Provider {
			case "openai":
				return "responses"
			case "azure":
				return "azure"
			case "openrouter":
				return "openrouter"
			case "requesty":
				return "requesty"
			case "zai":
				return "zai"
			case "xai":
				return "xai"
			case "alibaba":
				return "alibaba"
			default:
				return "openrouter"
			}
		}
	}

	// Fallback to legacy configuration
	// If OpenAI API key is configured, use Responses API
	if c.config.OpenAI.APIKey != "" {
		return "responses"
	}

	// If Azure OpenAI is configured, use Azure Chat Completions
	if c.config.AzureOpenAI.APIKey != "" {
		return "azure"
	}

	// Default to OpenRouter Chat Completions
	return "openrouter"
}

// GetResponseFromChatMessages gets a response from the AI based on chat messages
func (c *AiClient) GetResponseFromChatMessages(ctx context.Context, chatMessages []ChatMessage, model string) (string, error) {
	// Convert chat messages to AI client format
	aiMessages := []Message{}

	for i, msg := range chatMessages {
		var role string

		if i == 0 && !msg.FromUser {
			role = "system"
		} else if msg.FromUser {
			role = "user"
		} else {
			role = "assistant"
		}

		aiMessages = append(aiMessages, Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	logger.Info("Sending %d messages to AI using model: %s", len(aiMessages), model)

	// Determine which API to use
	apiType := c.determineAPIType(model)
	logger.Debug("Using API type: %s for model: %s", apiType, model)

	// Route to appropriate API
	var response string
	var err error

	switch apiType {
	case "responses":
		response, err = c.Response(ctx, aiMessages, model)
	case "azure":
		response, err = c.ChatCompletion(ctx, aiMessages, model)
	case "openrouter":
		response, err = c.ChatCompletion(ctx, aiMessages, model)
	case "requesty":
		response, err = c.ChatCompletion(ctx, aiMessages, model)
	case "zai":
		response, err = c.ChatCompletion(ctx, aiMessages, model)
	case "xai":
		response, err = c.ChatCompletion(ctx, aiMessages, model)
	case "alibaba":
		response, err = c.ChatCompletion(ctx, aiMessages, model)
	default:
		return "", fmt.Errorf("unknown API type: %s", apiType)
	}

	if err != nil {
		return "", err
	}

	return response, nil
}

// ChatCompletion sends a chat completion request to the OpenRouter API
func (c *AiClient) ChatCompletion(ctx context.Context, messages []Message, model string) (string, error) {
	reqBody := ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}

	// Get model configuration
	var provider string
	var apiKey string
	var baseURL string
	var apiBase string
	var apiVersion string
	var deploymentName string

	// Try to get model configuration
	if c.configMgr != nil {
		if modelConfig, exists := c.configMgr.GetCurrentModelConfig(); exists {
			provider = modelConfig.Provider
			apiKey = modelConfig.APIKey
			baseURL = modelConfig.BaseURL
			apiBase = modelConfig.APIBase
			apiVersion = modelConfig.APIVersion
			deploymentName = modelConfig.DeploymentName
		}
	}

	// Fallback to legacy configuration if no model config found
	if provider == "" {
		if c.config.AzureOpenAI.APIKey != "" {
			provider = "azure"
			apiKey = c.config.AzureOpenAI.APIKey
			apiBase = c.config.AzureOpenAI.APIBase
			apiVersion = c.config.AzureOpenAI.APIVersion
			deploymentName = c.config.AzureOpenAI.DeploymentName
		} else if c.config.OpenRouter.APIKey != "" {
			provider = "openrouter"
			apiKey = c.config.OpenRouter.APIKey
			baseURL = c.config.OpenRouter.BaseURL
		}
	}

	// determine endpoint and headers based on configuration
	var url string
	var apiKeyHeader string

	if provider == "azure" {
		// Use Azure OpenAI endpoint
		base := strings.TrimSuffix(apiBase, "/")
		url = fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
			base,
			deploymentName,
			apiVersion)
		apiKeyHeader = "api-key"

		// Azure endpoint doesn't expect model in body
		reqBody.Model = ""
	} else {
		// default OpenRouter/OpenAI compatible endpoint
		if baseURL == "" {
			switch provider {
			case "requesty":
				baseURL = "https://router.requesty.ai/v1"
			case "zai":
				baseURL = "https://api.zai.com/v1"
			case "xai":
				baseURL = "https://api.x.ai/v1"
			case "alibaba":
				baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
			default:
				baseURL = c.config.OpenRouter.BaseURL
			}
		}
		base := strings.TrimSuffix(baseURL, "/")
		url = base + "/chat/completions"
		apiKeyHeader = "Authorization"
		apiKey = "Bearer " + apiKey
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		logger.Error("Failed to marshal request: %v", err)
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		logger.Error("Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(apiKeyHeader, apiKey)

	req.Header.Set("HTTP-Referer", "https://github.com/andreim2k/aiterm")
	req.Header.Set("X-Title", "AITerm")

	// Log the request details for debugging before sending
	logger.Debug("Sending API request to: %s with model: %s", url, model)

	// Send the request
	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return "", fmt.Errorf("request canceled: %w", ctx.Err())
		}
		logger.Error("Failed to send request: %v", err)
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response: %v", err)
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log the raw response for debugging
	logger.Debug("API response status: %d, response size: %d bytes", resp.StatusCode, len(body))

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		logger.Error("API returned error: %s", body)
		return "", fmt.Errorf("API returned error: %s", body)
	}

	// Parse the response
	var completionResp ChatCompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		logger.Error("Failed to unmarshal response: %v, body: %s", err, body)
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Return the response content
	if len(completionResp.Choices) > 0 {
		responseContent := completionResp.Choices[0].Message.Content
		logger.Debug("Received AI response (%d characters): %s", len(responseContent), responseContent)
		return responseContent, nil
	}

	// Enhanced error for no completion choices
	logger.Error("No completion choices returned. Raw response: %s", string(body))
	return "", fmt.Errorf("no completion choices returned (model: %s, status: %d)", model, resp.StatusCode)
}

// Response sends a request to the OpenAI Responses API
func (c *AiClient) Response(ctx context.Context, messages []Message, model string) (string, error) {
	// Convert messages to Responses API format
	var input ResponseInput
	var instructions string

	if len(messages) == 0 {
		return "", fmt.Errorf("no messages provided")
	}

	// Check if first message is a system message
	if messages[0].Role == "system" {
		instructions = messages[0].Content
		if len(messages) > 1 {
			input = messages[1:]
		} else {
			// Only system message provided, no user input
			return "", fmt.Errorf("only system message provided, no user message to process")
		}
	} else {
		input = messages
	}

	reqBody := ResponseRequest{
		Model:        model,
		Input:        input,
		Instructions: instructions,
		Store:        false, // Default to stateless for better control over API usage and costs
	}

	// Get model configuration for OpenAI
	var apiKey string
	var baseURL string

	// Try to get model configuration
	if c.configMgr != nil {
		if modelConfig, exists := c.configMgr.GetCurrentModelConfig(); exists && modelConfig.Provider == "openai" {
			apiKey = modelConfig.APIKey
			baseURL = modelConfig.BaseURL
		}
	}

	// Fallback to legacy configuration
	if apiKey == "" {
		apiKey = c.config.OpenAI.APIKey
	}

	if baseURL == "" {
		baseURL = c.config.OpenAI.BaseURL
	}

	baseURL = strings.TrimSuffix(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	url := baseURL + "/responses"

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		logger.Error("Failed to marshal Responses API request: %v", err)
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		logger.Error("Failed to create Responses API request: %v", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	req.Header.Set("HTTP-Referer", "https://github.com/andreim2k/aiterm")
	req.Header.Set("X-Title", "AITerm")

	// Log the request details for debugging before sending
	logger.Debug("Sending Responses API request to: %s with model: %s", url, model)

	// Send the request
	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return "", fmt.Errorf("request canceled: %w", ctx.Err())
		}
		logger.Error("Failed to send Responses API request: %v", err)
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read Responses API response: %v", err)
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log the raw response for debugging
	logger.Debug("Responses API response status: %d, response size: %d bytes", resp.StatusCode, len(body))

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		logger.Error("Responses API returned error: %s", body)
		return "", fmt.Errorf("API returned error: %s", body)
	}

	// Parse the response
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		logger.Error("Failed to unmarshal Responses API response: %v, body: %s", err, body)
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API errors in response body
	if response.Error != nil {
		logger.Error("Responses API returned error: %s", response.Error.Message)
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	// Return the response content
	if response.OutputText != "" {
		logger.Debug("Received Responses API response (%d characters): %s", len(response.OutputText), response.OutputText)
		return response.OutputText, nil
	}

	// If no output_text, extract from message items
	for _, item := range response.Output {
		if item.Type == "message" && item.Status == "completed" {
			for _, content := range item.Content {
				if (content.Type == "output_text" || content.Type == "text") && content.Text != "" {
					logger.Debug("Received Responses API response from output items (%d characters): %s", len(content.Text), content.Text)
					return content.Text, nil
				}
			}
		}
	}

	// Enhanced error for no response content
	logger.Error("No response content returned. Raw response: %s", string(body))
	return "", fmt.Errorf("no response content returned (model: %s, status: %d)", model, resp.StatusCode)
}

func debugChatMessages(chatMessages []ChatMessage, response string) {

	timestamp := time.Now().Format("20060102-150405")
	configDir, _ := config.GetConfigDir()

	debugDir := fmt.Sprintf("%s/debug", configDir)
	if _, err := os.Stat(debugDir); os.IsNotExist(err) {
		_ = os.Mkdir(debugDir, 0755)
	}

	debugFileName := fmt.Sprintf("%s/debug-%s.txt", debugDir, timestamp)

	file, err := os.Create(debugFileName)
	if err != nil {
		logger.Error("Failed to create debug file: %v", err)
		return
	}
	defer func() { _ = file.Close() }()

	_, _ = file.WriteString("==================    SENT CHAT MESSAGES ==================\n\n")

	for i, msg := range chatMessages {
		role := "assistant"
		if msg.FromUser {
			role = "user"
		}
		if i == 0 && !msg.FromUser {
			role = "system"
		}
		timeStr := msg.Timestamp.Format(time.RFC3339)

		_, _ = fmt.Fprintf(file, "Message %d: Role=%s, Time=%s\n", i+1, role, timeStr)
		_, _ = fmt.Fprintf(file, "Content:\n%s\n\n", msg.Content)
	}

	_, _ = file.WriteString("==================    RECEIVED RESPONSE ==================\n\n")
	_, _ = file.WriteString(response)
	_, _ = file.WriteString("\n\n==================    END DEBUG ==================\n")
}

// TranslateNaturalLanguage translates natural language to shell commands without requiring a full manager
func (c *AiClient) TranslateNaturalLanguage(naturalLanguage string, osName string, shellPath string, cwd string, model string) (string, error) {
	// Build AI prompt for command translation
	if shellPath == "" {
		shellPath = "/bin/bash"
	}

	systemPrompt := fmt.Sprintf(`You are a shell command translator. Convert natural language to shell commands.

Operating System: %s
Shell: %s
Current Directory: %s

Rules:
1. Output ONLY a single shell command, nothing else
2. No explanations, no comments, no markdown
3. Command should be safe and follow best practices

Examples:
Input: "list all files"
Output: ls -la

Input: "find python files"
Output: find . -name "*.py"

Respond with ONLY the command.`, osName, shellPath, cwd)

	userPrompt := fmt.Sprintf("Translate: %s", naturalLanguage)

	// Create chat messages
	aiMessages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Call AI
	ctx := context.Background()

	// Determine which API to use
	apiType := c.determineAPIType(model)

	// Get the actual model identifier if we have a config manager
	actualModel := model
	if c.configMgr != nil {
		if modelConfig, exists := c.configMgr.GetModelConfig(model); exists {
			// Use the model as specified in the configuration
			actualModel = modelConfig.Model
		}
	}

	// Route to appropriate API
	var response string
	var err error

	switch apiType {
	case "responses":
		response, err = c.Response(ctx, aiMessages, actualModel)
	case "azure":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "openrouter":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "requesty":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "zai":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "xai":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "alibaba":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	default:
		return "", fmt.Errorf("unknown API type: %s", apiType)
	}

	if err != nil {
		return "", fmt.Errorf("AI API call failed: %v", err)
	}

	// Clean up response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```bash")
	response = strings.TrimPrefix(response, "```sh")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	return response, nil
}

// TranslateNaturalLanguageMultiple generates multiple shell command options from natural language
func (c *AiClient) TranslateNaturalLanguageMultiple(naturalLanguage string, osName string, shellPath string, cwd string, model string, count int) ([]string, error) {
	// Build AI prompt for command translation with multiple options
	if shellPath == "" {
		shellPath = "/bin/bash"
	}

	systemPrompt := `You are a shell command translator. Convert natural language tasks to multiple DIFFERENT shell command options.

Rules:
1. Output as many DIFFERENT and VALID shell commands as you can think of, one per line
2. Number each option (1., 2., 3., etc.)
3. No explanations, no comments, no markdown
4. Each command should be safe and follow best practices
5. Each option should be a valid, DIFFERENT approach to accomplish the EXACT task
6. Focus on quality over quantity - only include commands that are truly different approaches

Examples:
Task: "list all files"
1. ls -la
2. find . -type f
3. tree -a
4. ls -R

Task: "find python files"
1. find . -name "*.py"
2. grep -r "\.py$" .
3. ls *.py
4. locate "*.py"

Respond with as many DIFFERENT and VALID commands that directly accomplish the requested task, one per line.`

	userPrompt := fmt.Sprintf("Task: %s\n\nProvide as many different and valid shell commands as you can to accomplish this task.", naturalLanguage)

	// Create chat messages
	aiMessages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Call AI
	ctx := context.Background()

	// Determine which API to use
	apiType := c.determineAPIType(model)

	// Get the actual model identifier if we have a config manager
	actualModel := model
	if c.configMgr != nil {
		if modelConfig, exists := c.configMgr.GetModelConfig(model); exists {
			// Use the model as specified in the configuration
			actualModel = modelConfig.Model
		}
	}

	// Route to appropriate API
	var response string
	var err error

	switch apiType {
	case "responses":
		response, err = c.Response(ctx, aiMessages, actualModel)
	case "azure":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "openrouter":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "requesty":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "zai":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "xai":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	case "alibaba":
		response, err = c.ChatCompletion(ctx, aiMessages, actualModel)
	default:
		return nil, fmt.Errorf("unknown API type: %s", apiType)
	}

	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %v", err)
	}

	// Parse the response into multiple options
	response = strings.TrimSpace(response)
	lines := strings.Split(response, "\n")

	var options []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines
		if line == "" {
			continue
		}
		// Remove numbering if present (e.g., "1. ", "2. ", etc.)
		if len(line) > 2 && line[1] == '.' && line[0] >= '1' && line[0] <= '9' {
			line = strings.TrimSpace(line[2:])
		} else if len(line) > 3 && line[2] == '.' && line[0] >= '1' && line[0] <= '9' && line[1] >= '0' && line[1] <= '9' {
			line = strings.TrimSpace(line[3:])
		}
		// Remove any code block markers
		line = strings.TrimPrefix(line, "```bash")
		line = strings.TrimPrefix(line, "```sh")
		line = strings.TrimPrefix(line, "```")
		line = strings.TrimSuffix(line, "```")
		line = strings.TrimSpace(line)

		// Only add non-empty lines that look like commands
		if line != "" && !strings.HasPrefix(line, "Input:") && !strings.HasPrefix(line, "Output:") && !strings.HasPrefix(line, "Examples:") {
			options = append(options, line)
		}
	}

	// Ensure we have the requested number of options (or at least one)
	if len(options) == 0 {
		return nil, fmt.Errorf("no valid command options generated")
	}

	return options, nil
}
