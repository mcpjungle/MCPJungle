// Package configvalidation validates MCPJungle JSON configuration files without
// requiring a running MCPJungle server.
package configvalidation

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/mcpjungle/mcpjungle/internal"
	"github.com/mcpjungle/mcpjungle/internal/configresolver"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// ConfigType identifies a supported MCPJungle JSON configuration shape.
type ConfigType string

const (
	// ConfigTypeMCPServer is used with `mcpjungle register --conf`.
	ConfigTypeMCPServer ConfigType = "mcp-server"
	// ConfigTypeToolGroup is used with `mcpjungle create group --conf`.
	ConfigTypeToolGroup ConfigType = "tool-group"
	// ConfigTypeMCPClient is used with `mcpjungle create mcp-client --conf`.
	ConfigTypeMCPClient ConfigType = "mcp-client"
	// ConfigTypeUser is used with `mcpjungle create user --conf`.
	ConfigTypeUser ConfigType = "user"
)

// Result describes a successful validation.
type Result struct {
	Type ConfigType
}

type validator struct {
	configType ConfigType
	matches    func(map[string]json.RawMessage) bool
	validate   func([]byte) error
}

var validators = []validator{
	{configType: ConfigTypeMCPServer, matches: looksLikeMCPServerConfig, validate: validateMCPServerConfig},
	{configType: ConfigTypeToolGroup, matches: looksLikeToolGroupConfig, validate: validateToolGroupConfig},
	{configType: ConfigTypeMCPClient, matches: looksLikeMCPClientConfig, validate: validateMCPClientConfig},
	{configType: ConfigTypeUser, matches: looksLikeUserConfig, validate: validateUserConfig},
}

var (
	validServerName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	validGroupName  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
)

// ValidateFile reads and validates a JSON config file, inferring the config type.
func ValidateFile(filePath string) (*Result, error) {
	return ValidateFileAs(filePath, "")
}

// ValidateFileAs reads and validates a JSON config file as the requested type.
func ValidateFileAs(filePath string, configType ConfigType) (*Result, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}
	result, err := ValidateBytesAs(data, configType)
	if err != nil {
		return nil, fmt.Errorf("invalid config file %s: %w", filePath, err)
	}
	return result, nil
}

// ValidateBytes validates a JSON config document, inferring the config type.
func ValidateBytes(data []byte) (*Result, error) {
	return ValidateBytesAs(data, "")
}

// ValidateBytesAs validates a JSON config document as the requested type.
func ValidateBytesAs(data []byte, configType ConfigType) (*Result, error) {
	raw, err := parseConfigObject(data)
	if err != nil {
		return nil, err
	}

	if configType != "" {
		for _, v := range validators {
			if v.configType != configType {
				continue
			}
			if err := v.validate(data); err != nil {
				return nil, err
			}
			return &Result{Type: configType}, nil
		}
		return nil, fmt.Errorf("unsupported config type %q", configType)
	}

	for _, v := range validators {
		if !v.matches(raw) {
			continue
		}
		if err := v.validate(data); err != nil {
			return nil, err
		}
		return &Result{Type: v.configType}, nil
	}

	return nil, fmt.Errorf(
		"config does not match any supported MCPJungle config type: %s",
		supportedConfigTypes(),
	)
}

func parseConfigObject(data []byte) (map[string]json.RawMessage, error) {
	if strings.TrimSpace(string(data)) == "" {
		return nil, fmt.Errorf("config file is empty")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}
	if raw == nil {
		return nil, fmt.Errorf("config must be a JSON object")
	}
	return raw, nil
}

func looksLikeMCPServerConfig(raw map[string]json.RawMessage) bool {
	return hasAnyKey(raw,
		"transport",
		"url",
		"command",
		"args",
		"env",
		"bearer_token",
		"headers",
		"session_mode",
		"oauth_redirect_uri",
		"oauth_client_id",
		"oauth_client_secret",
		"oauth_scopes",
	)
}

func looksLikeToolGroupConfig(raw map[string]json.RawMessage) bool {
	return hasAnyKey(raw, "included_tools", "included_servers", "excluded_tools")
}

func looksLikeMCPClientConfig(raw map[string]json.RawMessage) bool {
	return hasAnyKey(raw, "allowed_servers")
}

func looksLikeUserConfig(raw map[string]json.RawMessage) bool {
	return hasAnyKey(raw, "access_token", "access_token_ref")
}

func validateMCPServerConfig(data []byte) error {
	var input types.RegisterServerInput
	if err := unmarshalAndResolve(data, &input); err != nil {
		return err
	}
	if err := validateServerName(input.Name); err != nil {
		return err
	}
	transport, err := types.ValidateTransport(input.Transport)
	if err != nil {
		return err
	}
	if _, err := types.ValidateSessionMode(input.SessionMode); err != nil {
		return err
	}

	switch transport {
	case types.TransportStreamableHTTP, types.TransportSSE:
		if err := validateHTTPURL(input.URL); err != nil {
			return err
		}
	case types.TransportStdio:
		if strings.TrimSpace(input.Command) == "" {
			return fmt.Errorf("command is required for stdio transport")
		}
	}

	return nil
}

func validateToolGroupConfig(data []byte) error {
	var group types.ToolGroup
	if err := unmarshalAndResolve(data, &group); err != nil {
		return err
	}
	if strings.TrimSpace(group.Name) == "" {
		return fmt.Errorf("tool group name cannot be empty")
	}
	if !validGroupName.MatchString(group.Name) {
		return fmt.Errorf("invalid group name: must start with an alphanumeric character and can only contain alphanumeric characters, underscores, and hyphens")
	}
	if len(group.IncludedTools) == 0 && len(group.IncludedServers) == 0 {
		return fmt.Errorf("tool group config must define included_tools or included_servers")
	}
	if err := validateNonEmptyStrings("included_tools", group.IncludedTools); err != nil {
		return err
	}
	if err := validateNonEmptyStrings("included_servers", group.IncludedServers); err != nil {
		return err
	}
	return validateNonEmptyStrings("excluded_tools", group.ExcludedTools)
}

func validateMCPClientConfig(data []byte) error {
	var client types.McpClientConfig
	if err := unmarshalAndResolve(data, &client); err != nil {
		return err
	}
	if strings.TrimSpace(client.Name) == "" {
		return fmt.Errorf("MCP client config must define a client name")
	}
	if err := validateNonEmptyStrings("allowed_servers", client.AllowMcpServers); err != nil {
		return err
	}
	return validateAccessTokenConfig("MCP client", client.AccessToken, client.AccessTokenRef)
}

func validateUserConfig(data []byte) error {
	var user types.UserConfig
	if err := unmarshalAndResolve(data, &user); err != nil {
		return err
	}
	if strings.TrimSpace(user.Username) == "" {
		return fmt.Errorf("user config must define a username")
	}
	return validateAccessTokenConfig("user", user.AccessToken, user.AccessTokenRef)
}

func unmarshalAndResolve(data []byte, target any) error {
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	if err := configresolver.ResolveEnvVars(target); err != nil {
		return fmt.Errorf("failed to resolve config file environment variables: %w", err)
	}
	return nil
}

func validateAccessTokenConfig(label, token string, ref types.AccessTokenRef) error {
	resolved, err := resolveAccessToken(token, ref)
	if err != nil {
		return err
	}
	if resolved == "" {
		return fmt.Errorf("%s config must supply a custom access token", label)
	}
	if err := internal.ValidateAccessToken(resolved); err != nil {
		return fmt.Errorf("invalid access token: %v", err)
	}
	return nil
}

func resolveAccessToken(token string, ref types.AccessTokenRef) (string, error) {
	if token != "" {
		return token, nil
	}
	if ref.Env != "" {
		value, ok := os.LookupEnv(ref.Env)
		if ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed, nil
			}
		}
		if ref.File == "" {
			return "", fmt.Errorf("environment variable %s is not set or empty", ref.Env)
		}
	}
	if ref.File != "" {
		data, err := os.ReadFile(ref.File)
		if err != nil {
			return "", fmt.Errorf("failed to read access token file %s: %w", ref.File, err)
		}
		trimmed := strings.TrimSpace(string(data))
		if trimmed == "" {
			return "", fmt.Errorf("access token file %s is empty", ref.File)
		}
		return trimmed, nil
	}
	return "", nil
}

func validateServerName(name string) error {
	if name == "" {
		return fmt.Errorf("invalid server name: must not be empty")
	}
	if !validServerName.MatchString(name) {
		return fmt.Errorf("invalid server name: must only contain alphanumeric characters, underscores, and hyphens")
	}
	if strings.Contains(name, "__") {
		return fmt.Errorf("invalid server name: must not contain multiple consecutive underscores")
	}
	if strings.HasSuffix(name, "_") {
		return fmt.Errorf("invalid server name: must not end with an underscore")
	}
	return nil
}

func validateHTTPURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid url: %q must be a valid http or https url", rawURL)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("invalid url: %q must be a valid http or https url", rawURL)
	}
}

func validateNonEmptyStrings(field string, values []string) error {
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s[%d] cannot be empty", field, i)
		}
	}
	return nil
}

func hasAnyKey(raw map[string]json.RawMessage, keys ...string) bool {
	for _, key := range keys {
		if _, ok := raw[key]; ok {
			return true
		}
	}
	return false
}

func supportedConfigTypes() string {
	types := make([]string, 0, len(validators))
	for _, v := range validators {
		types = append(types, string(v.configType))
	}
	return strings.Join(types, ", ")
}
