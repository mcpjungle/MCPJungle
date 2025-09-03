package types

// PromptArgument represents an argument that can be passed to a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// PromptResult represents the result of getting a prompt with arguments
type PromptResult struct {
	Description string          `json:"description"`
	Messages    []PromptMessage `json:"messages"`
	Meta        map[string]any  `json:"meta,omitempty"`
}

// PromptMessage represents a message in a prompt template
type PromptMessage struct {
	Role    string         `json:"role"` // "user" or "assistant"
	Content map[string]any `json:"content"`
}

// PromptGetRequest represents a request to get a prompt with arguments
type PromptGetRequest struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments"`
}
