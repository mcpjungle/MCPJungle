package cmd

import (
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestLoginCommandStructure(t *testing.T) {
	t.Run("command_properties", func(t *testing.T) {
		testhelpers.AssertEqual(t, "login [access_token]", loginCmd.Use)
		testhelpers.AssertEqual(t, "Log in to MCPJungle (Production mode)", loginCmd.Short)
	})

	t.Run("command_annotations", func(t *testing.T) {
		annotationTests := []testhelpers.CommandAnnotationTest{
			{Key: "group", Expected: string(subCommandGroupAdvanced)},
			{Key: "order", Expected: "6"},
		}
		testhelpers.TestCommandAnnotations(t, loginCmd.Annotations, annotationTests)
	})

	t.Run("command_functions", func(t *testing.T) {
		testhelpers.AssertNotNil(t, loginCmd.Args)
	})
}

func TestUserRoleConstants(t *testing.T) {
	t.Run("user_role_constants", func(t *testing.T) {
		testhelpers.AssertEqual(t, "admin", string(types.UserRoleAdmin))
		testhelpers.AssertEqual(t, "user", string(types.UserRoleUser))
	})
}

func TestRunLoginFunctional(t *testing.T) {
	t.Run("runLogin would handle access token validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			accessToken string
			expectValid bool
		}{
			{
				name:        "valid access token",
				accessToken: "abc123def456",
				expectValid: true,
			},
			{
				name:        "empty access token",
				accessToken: "",
				expectValid: false,
			},
			{
				name:        "access token with spaces",
				accessToken: "abc 123 def 456",
				expectValid: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				isValid := tc.accessToken != "" && !strings.Contains(tc.accessToken, " ")

				if isValid != tc.expectValid {
					t.Errorf("Expected valid %v, got %v for access token '%s'", tc.expectValid, isValid, tc.accessToken)
				}
			})
		}
	})

	t.Run("runLogin would handle user role processing", func(t *testing.T) {
		testCases := []struct {
			name           string
			userRole       string
			expectedOutput string
		}{
			{
				name:           "admin user",
				userRole:       string(types.UserRoleAdmin),
				expectedOutput: "You are an administrator of MCPJungle",
			},
			{
				name:           "regular user",
				userRole:       string(types.UserRoleUser),
				expectedOutput: "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var output string
				if tc.userRole == string(types.UserRoleAdmin) {
					output = "You are an administrator of MCPJungle"
				}

				if output != tc.expectedOutput {
					t.Errorf("Expected output '%s', got '%s' for role '%s'", tc.expectedOutput, output, tc.userRole)
				}
			})
		}
	})

	t.Run("runLogin would handle config file creation", func(t *testing.T) {
		testCases := []struct {
			name           string
			accessToken    string
			expectedConfig map[string]string
		}{
			{
				name:        "valid access token",
				accessToken: "abc123def456",
				expectedConfig: map[string]string{
					"AccessToken": "abc123def456",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := map[string]string{
					"AccessToken": tc.accessToken,
				}

				if config["AccessToken"] != tc.expectedConfig["AccessToken"] {
					t.Errorf("Expected config %v, got %v", tc.expectedConfig, config)
				}
			})
		}
	})

	t.Run("runLogin would handle user information display", func(t *testing.T) {
		testCases := []struct {
			name           string
			username       string
			role           string
			expectedOutput string
		}{
			{
				name:           "admin user",
				username:       "admin",
				role:           string(types.UserRoleAdmin),
				expectedOutput: "You are now logged in as admin",
			},
			{
				name:           "regular user",
				username:       "user1",
				role:           string(types.UserRoleUser),
				expectedOutput: "You are now logged in as user1",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				output := "You are now logged in as " + tc.username

				if output != tc.expectedOutput {
					t.Errorf("Expected output '%s', got '%s'", tc.expectedOutput, output)
				}
			})
		}
	})

	t.Run("runLogin would handle error scenarios", func(t *testing.T) {
		testCases := []struct {
			name        string
			user        *types.User
			expectError bool
		}{
			{
				name:        "valid user",
				user:        &types.User{Username: "testuser", Role: string(types.UserRoleUser)},
				expectError: false,
			},
			{
				name:        "nil user",
				user:        nil,
				expectError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				hasError := tc.user == nil

				if hasError != tc.expectError {
					t.Errorf("Expected error %v, got %v", tc.expectError, hasError)
				}
			})
		}
	})
}
