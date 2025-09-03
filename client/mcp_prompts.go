package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// ListPrompts retrieves all prompts or prompts filtered by server name
func (c *Client) ListPrompts(serverName string) ([]model.Prompt, error) {
	u, err := c.constructAPIEndpoint("/prompts")
	if err != nil {
		return nil, fmt.Errorf("failed to construct API endpoint: %w", err)
	}

	// Add server filter if specified
	if serverName != "" {
		parsed, _ := url.Parse(u)
		q := parsed.Query()
		q.Set("server", serverName)
		parsed.RawQuery = q.Encode()
		u = parsed.String()
	}

	req, err := c.newRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var prompts []model.Prompt
	if err := json.NewDecoder(resp.Body).Decode(&prompts); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return prompts, nil
}

// GetPrompt retrieves a specific prompt by name
func (c *Client) GetPrompt(name string) (*model.Prompt, error) {
	u, err := c.constructAPIEndpoint("/prompt")
	if err != nil {
		return nil, fmt.Errorf("failed to construct API endpoint: %w", err)
	}

	// Add name as query parameter
	parsed, _ := url.Parse(u)
	q := parsed.Query()
	q.Set("name", name)
	parsed.RawQuery = q.Encode()
	u = parsed.String()

	req, err := c.newRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var prompt model.Prompt
	if err := json.NewDecoder(resp.Body).Decode(&prompt); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &prompt, nil
}

// GetPromptWithArgs retrieves a prompt with arguments and returns the rendered template
func (c *Client) GetPromptWithArgs(name string, arguments map[string]string) (*types.PromptResult, error) {
	u, err := c.constructAPIEndpoint("/prompts/get")
	if err != nil {
		return nil, fmt.Errorf("failed to construct API endpoint: %w", err)
	}

	request := types.PromptGetRequest{
		Name:      name,
		Arguments: arguments,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := c.newRequest("POST", u, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %d, message: %s", resp.StatusCode, body)
	}

	var result types.PromptResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// EnablePrompts enables one or more prompts
func (c *Client) EnablePrompts(entity string) ([]string, error) {
	return c.setPromptsEnabled(entity, true)
}

// DisablePrompts disables one or more prompts
func (c *Client) DisablePrompts(entity string) ([]string, error) {
	return c.setPromptsEnabled(entity, false)
}

// setPromptsEnabled is a helper function to enable or disable prompts
func (c *Client) setPromptsEnabled(entity string, enabled bool) ([]string, error) {
	var endpoint string
	if enabled {
		endpoint = "/prompts/enable"
	} else {
		endpoint = "/prompts/disable"
	}

	u, err := c.constructAPIEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to construct API endpoint: %w", err)
	}

	// Add entity as query parameter
	parsed, _ := url.Parse(u)
	q := parsed.Query()
	q.Set("entity", entity)
	parsed.RawQuery = q.Encode()
	u = parsed.String()

	req, err := c.newRequest("POST", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		action := "enable"
		if !enabled {
			action = "disable"
		}
		return nil, fmt.Errorf("failed to %s prompts: status %d, message: %s", action, resp.StatusCode, body)
	}

	var promptNames []string
	if err := json.NewDecoder(resp.Body).Decode(&promptNames); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return promptNames, nil
}
