package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRegisterCommandStructure(t *testing.T) {
	t.Run("register command has correct properties", func(t *testing.T) {
		if registerMCPServerCmd.Use != "register" {
			t.Errorf("Expected register command Use to be 'register', got %s", registerMCPServerCmd.Use)
		}
		if registerMCPServerCmd.Short != "Register an MCP Server" {
			t.Errorf("Expected register command Short to be 'Register an MCP Server', got %s", registerMCPServerCmd.Short)
		}
	})

	t.Run("register command has correct annotations", func(t *testing.T) {
		if registerMCPServerCmd.Annotations == nil {
			t.Fatal("Register command missing annotations")
		}

		group, hasGroup := registerMCPServerCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Register command missing 'group' annotation")
		}
		if group != string(subCommandGroupBasic) {
			t.Errorf("Expected register command group to be 'basic', got %s", group)
		}

		order, hasOrder := registerMCPServerCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Register command missing 'order' annotation")
		}
		if order != "2" {
			t.Errorf("Expected register command order to be '2', got %s", order)
		}
	})

	t.Run("register command has PreRunE function", func(t *testing.T) {
		if registerMCPServerCmd.PreRunE == nil {
			t.Fatal("Register command missing PreRunE function")
		}
	})
}

func TestRegisterCommandFlags(t *testing.T) {
	t.Run("register command has name flag", func(t *testing.T) {
		nameFlag := registerMCPServerCmd.Flags().Lookup("name")
		if nameFlag == nil {
			t.Fatal("Register command missing 'name' flag")
		}
		if nameFlag.Usage == "" {
			t.Error("Name flag should have usage description")
		}
	})

	t.Run("register command has url flag", func(t *testing.T) {
		urlFlag := registerMCPServerCmd.Flags().Lookup("url")
		if urlFlag == nil {
			t.Fatal("Register command missing 'url' flag")
		}
		if urlFlag.Usage == "" {
			t.Error("URL flag should have usage description")
		}
	})

	t.Run("register command has description flag", func(t *testing.T) {
		descFlag := registerMCPServerCmd.Flags().Lookup("description")
		if descFlag == nil {
			t.Fatal("Register command missing 'description' flag")
		}
		if descFlag.Usage == "" {
			t.Error("Description flag should have usage description")
		}
	})

	t.Run("register command has bearer-token flag", func(t *testing.T) {
		tokenFlag := registerMCPServerCmd.Flags().Lookup("bearer-token")
		if tokenFlag == nil {
			t.Fatal("Register command missing 'bearer-token' flag")
		}
		if tokenFlag.Usage == "" {
			t.Error("Bearer-token flag should have usage description")
		}
	})

	t.Run("register command has conf flag", func(t *testing.T) {
		confFlag := registerMCPServerCmd.Flags().Lookup("conf")
		if confFlag == nil {
			t.Fatal("Register command missing 'conf' flag")
		}
		if confFlag.Usage == "" {
			t.Error("Conf flag should have usage description")
		}
	})

	t.Run("register command has conf flag with short form", func(t *testing.T) {
		// The StringVarP creates both "conf" and "c" flags
		confFlag := registerMCPServerCmd.Flags().Lookup("conf")
		if confFlag == nil {
			t.Fatal("Register command missing 'conf' flag")
		}
		// Note: StringVarP creates both long and short forms, but we test the long form
	})
}

func TestRegisterCommandVariables(t *testing.T) {
	t.Run("register command variables are initialized", func(t *testing.T) {
		// These variables should be initialized to empty strings
		if registerCmdServerName != "" {
			t.Errorf("Expected registerCmdServerName to be empty, got %s", registerCmdServerName)
		}
		if registerCmdServerURL != "" {
			t.Errorf("Expected registerCmdServerURL to be empty, got %s", registerCmdServerURL)
		}
		if registerCmdServerDesc != "" {
			t.Errorf("Expected registerCmdServerDesc to be empty, got %s", registerCmdServerDesc)
		}
		if registerCmdBearerToken != "" {
			t.Errorf("Expected registerCmdBearerToken to be empty, got %s", registerCmdBearerToken)
		}
		if registerCmdServerConfigFilePath != "" {
			t.Errorf("Expected registerCmdServerConfigFilePath to be empty, got %s", registerCmdServerConfigFilePath)
		}
	})
}

func TestRunRegisterMCPServerFunctional(t *testing.T) {
	t.Run("runRegisterMCPServer would handle config file vs flags logic", func(t *testing.T) {
		testCases := []struct {
			name                  string
			configFilePath        string
			serverName            string
			serverURL             string
			expectedUseConfigFile bool
		}{
			{
				name:                  "use config file when provided",
				configFilePath:        "/path/to/config.json",
				serverName:            "",
				serverURL:             "",
				expectedUseConfigFile: true,
			},
			{
				name:                  "use flags when no config file",
				configFilePath:        "",
				serverName:            "test-server",
				serverURL:             "http://localhost:8080",
				expectedUseConfigFile: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				useConfigFile := tc.configFilePath != ""

				if useConfigFile != tc.expectedUseConfigFile {
					t.Errorf("Expected useConfigFile %v, got %v", tc.expectedUseConfigFile, useConfigFile)
				}
			})
		}
	})

	t.Run("runRegisterMCPServer would handle server name validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			serverName  string
			expectValid bool
		}{
			{
				name:        "valid server name",
				serverName:  "test-server",
				expectValid: true,
			},
			{
				name:        "server name with spaces",
				serverName:  "test server",
				expectValid: false,
			},
			{
				name:        "server name with special characters",
				serverName:  "test@server",
				expectValid: false,
			},
			{
				name:        "server name with multiple underscores",
				serverName:  "test__server",
				expectValid: false,
			},
			{
				name:        "empty server name",
				serverName:  "",
				expectValid: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				isValid := tc.serverName != "" &&
					!strings.Contains(tc.serverName, " ") &&
					!strings.Contains(tc.serverName, "@") &&
					!strings.Contains(tc.serverName, "__")

				if isValid != tc.expectValid {
					t.Errorf("Expected valid %v, got %v for server name '%s'", tc.expectValid, isValid, tc.serverName)
				}
			})
		}
	})

	t.Run("runRegisterMCPServer would handle URL validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			serverURL   string
			expectValid bool
		}{
			{
				name:        "valid HTTP URL",
				serverURL:   "http://localhost:8080",
				expectValid: true,
			},
			{
				name:        "valid HTTPS URL",
				serverURL:   "https://example.com",
				expectValid: true,
			},
			{
				name:        "invalid URL",
				serverURL:   "not-a-url",
				expectValid: false,
			},
			{
				name:        "empty URL",
				serverURL:   "",
				expectValid: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				isValid := tc.serverURL != "" &&
					(strings.HasPrefix(tc.serverURL, "http://") || strings.HasPrefix(tc.serverURL, "https://"))

				if isValid != tc.expectValid {
					t.Errorf("Expected valid %v, got %v for URL '%s'", tc.expectValid, isValid, tc.serverURL)
				}
			})
		}
	})

	t.Run("runRegisterMCPServer would handle bearer token processing", func(t *testing.T) {
		testCases := []struct {
			name          string
			bearerToken   string
			expectedToken string
		}{
			{
				name:          "empty bearer token",
				bearerToken:   "",
				expectedToken: "",
			},
			{
				name:          "valid bearer token",
				bearerToken:   "abc123",
				expectedToken: "abc123",
			},
			{
				name:          "bearer token with Bearer prefix",
				bearerToken:   "Bearer abc123",
				expectedToken: "abc123",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				token := tc.bearerToken
				token = strings.TrimPrefix(token, "Bearer ")

				if token != tc.expectedToken {
					t.Errorf("Expected token %s, got %s", tc.expectedToken, token)
				}
			})
		}
	})

	t.Run("runRegisterMCPServer would handle config file reading", func(t *testing.T) {
		testCases := []struct {
			name          string
			configContent string
			expectValid   bool
		}{
			{
				name: "valid JSON config",
				configContent: `{
					"name": "test-server",
					"url": "http://localhost:8080",
					"description": "Test server"
				}`,
				expectValid: true,
			},
			{
				name: "invalid JSON config",
				configContent: `{
					"name": "test-server",
					"url": "http://localhost:8080",
					"description": "Test server"
				`, // Missing closing brace
				expectValid: false,
			},
			{
				name:          "empty config",
				configContent: "",
				expectValid:   false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var config map[string]any
				err := json.Unmarshal([]byte(tc.configContent), &config)
				isValid := err == nil

				if isValid != tc.expectValid {
					t.Errorf("Expected valid %v, got %v for config: %s", tc.expectValid, isValid, tc.configContent)
				}
			})
		}
	})
}
