package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/accesstoken"
	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

func TestCreateCommandStructure(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "create", createCmd.Use)
	testhelpers.AssertEqual(t, "Create entities in mcpjungle", createCmd.Short)

	// Test command annotations
	annotationTests := []testhelpers.CommandAnnotationTest{
		{Key: "group", Expected: string(subCommandGroupAdvanced)},
		{Key: "order", Expected: "4"},
	}
	testhelpers.TestCommandAnnotations(t, createCmd.Annotations, annotationTests)

	// Test subcommands count
	subcommands := createCmd.Commands()
	testhelpers.AssertEqual(t, 3, len(subcommands))
}

func TestCreateMcpClientSubcommand(t *testing.T) {
	t.Parallel()

	// Test command properties
	testhelpers.AssertEqual(t, "mcp-client [name] | --conf <file>", createMcpClientCmd.Use)
	testhelpers.AssertEqual(t, "Create an authenticated MCP client (Enterprise mode)", createMcpClientCmd.Short)
	testhelpers.AssertNotNil(t, createMcpClientCmd.Long)
	testhelpers.AssertTrue(t, len(createMcpClientCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, createMcpClientCmd.RunE)
	testhelpers.AssertNotNil(t, createMcpClientCmd.Args)

	// Test command flags
	allowFlag := createMcpClientCmd.Flags().Lookup("allow")
	testhelpers.AssertNotNil(t, allowFlag)
	testhelpers.AssertTrue(t, len(allowFlag.Usage) > 0, "Allow flag should have usage description")

	descriptionFlag := createMcpClientCmd.Flags().Lookup("description")
	testhelpers.AssertNotNil(t, descriptionFlag)
	testhelpers.AssertTrue(t, len(descriptionFlag.Usage) > 0, "Description flag should have usage description")

	accessTokenFlag := createMcpClientCmd.Flags().Lookup("access-token")
	testhelpers.AssertNotNil(t, accessTokenFlag)
	testhelpers.AssertTrue(t, len(accessTokenFlag.Usage) > 0, "Access token flag should have usage description")

	configFlag := createMcpClientCmd.Flags().Lookup("conf")
	testhelpers.AssertNotNil(t, configFlag)
	testhelpers.AssertTrue(t, len(configFlag.Usage) > 0, "Config flag should have usage description")
}

func TestCreateUserSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "user [username] | --conf <file>", createUserCmd.Use)
	testhelpers.AssertEqual(t, "Create a new user (Enterprise mode)", createUserCmd.Short)
	testhelpers.AssertNotNil(t, createUserCmd.Long)
	testhelpers.AssertTrue(t, len(createUserCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, createUserCmd.RunE)
	testhelpers.AssertNotNil(t, createUserCmd.Args)

	// Test command flags
	configFlag := createUserCmd.Flags().Lookup("conf")
	testhelpers.AssertNotNil(t, configFlag)
	testhelpers.AssertTrue(t, len(configFlag.Usage) > 0, "Config flag should have usage description")
}

func TestCreateToolGroupSubcommand(t *testing.T) {
	// Test command properties
	testhelpers.AssertEqual(t, "group --conf <file>", createToolGroupCmd.Use)
	testhelpers.AssertEqual(t, "Create a Group of MCP Tools", createToolGroupCmd.Short)
	testhelpers.AssertNotNil(t, createToolGroupCmd.Long)
	testhelpers.AssertTrue(t, len(createToolGroupCmd.Long) > 0, "Long description should not be empty")

	// Test command functions
	testhelpers.AssertNotNil(t, createToolGroupCmd.RunE)

	// Test command flags
	confFlag := createToolGroupCmd.Flags().Lookup("conf")
	testhelpers.AssertNotNil(t, confFlag)
	testhelpers.AssertTrue(t, len(confFlag.Usage) > 0, "Conf flag should have usage description")
}

func TestCreateCommandVariables(t *testing.T) {
	// Test that command variables are properly initialized to empty values
	testhelpers.AssertEqual(t, "", createMcpClientCmdAllowedServers)
	testhelpers.AssertEqual(t, "", createMcpClientCmdDescription)
	testhelpers.AssertEqual(t, "", createMcpClientCmdConfigFilePath)
	testhelpers.AssertEqual(t, "", createUserCmdConfigFilePath)
	testhelpers.AssertEqual(t, "", createToolGroupConfigFilePath)
}

// Test allow list parsing logic
func TestParseAllowList(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", []string{}},
		{"single server", "server1", []string{"server1"}},
		{"multiple servers", "server1,server2,server3", []string{"server1", "server2", "server3"}},
		{"servers with spaces", "server1, server2 , server3", []string{"server1", "server2", "server3"}},
		{"servers with empty elements", "server1,,server2", []string{"server1", "server2"}},
		{"servers with only spaces", "server1,  ,server2", []string{"server1", "server2"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse allow list (simulating the logic from create.go)
			allowList := make([]string, 0)
			for _, s := range strings.Split(tc.input, ",") {
				trimmed := strings.TrimSpace(s)
				if trimmed != "" {
					allowList = append(allowList, trimmed)
				}
			}

			// Compare results
			if len(tc.expected) != len(allowList) {
				t.Errorf("Expected length %d, got %d", len(tc.expected), len(allowList))
				return
			}
			for i, expected := range tc.expected {
				if expected != allowList[i] {
					t.Errorf("Expected[%d] = %s, got %s", i, expected, allowList[i])
				}
			}
		})
	}
}

// Integration tests for create commands
func TestCreateCommandIntegration(t *testing.T) {
	// Verify that createCmd is properly added to rootCmd
	testhelpers.AssertNotNil(t, createCmd)

	// Test all create subcommands are properly configured
	subcommands := createCmd.Commands()
	expectedSubcommands := []string{"mcp-client", "user", "group"}

	testhelpers.AssertEqual(t, len(expectedSubcommands), len(subcommands))

	for _, expected := range expectedSubcommands {
		found := false
		for _, subcmd := range subcommands {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		testhelpers.AssertTrue(t, found, "Expected subcommand '"+expected+"' not found")
	}
}

// Test argument validation
func TestCreateCommandArgumentValidation(t *testing.T) {
	// Test that commands properly validate arguments
	testhelpers.AssertNotNil(t, createMcpClientCmd.Args)
	testhelpers.AssertNotNil(t, createUserCmd.Args)
	// createToolGroupCmd doesn't have Args validation, which is correct

	// Test various invalid input scenarios
	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{"empty args", []string{}, true},
		{"too many args", []string{"arg1", "arg2", "arg3"}, true},
		{"valid single arg", []string{"valid-arg"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test mcp-client command args validation
			if createMcpClientCmd.Args != nil {
				err := createMcpClientCmd.Args(createMcpClientCmd, tc.args)
				if tc.expectError {
					testhelpers.AssertError(t, err)
				} else {
					testhelpers.AssertNoError(t, err)
				}
			}

			// Test user command args validation
			if createUserCmd.Args != nil {
				err := createUserCmd.Args(createUserCmd, tc.args)
				if tc.expectError {
					testhelpers.AssertError(t, err)
				} else {
					testhelpers.AssertNoError(t, err)
				}
			}
		})
	}
}

func TestResolveAccessTokenFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("direct token wins", func(t *testing.T) {
		t.Parallel()
		r := types.AccessTokenRef{}
		token, err := accesstoken.Resolve(accesstoken.Input{Inline: "direct-token", Ref: r})
		testhelpers.AssertNoError(t, err)
		testhelpers.AssertEqual(t, "direct-token", token)
	})

	t.Run("env var used when set", func(t *testing.T) {
		t.Parallel()
		env := "MCPJ_TEST_TOKEN"
		_ = os.Setenv(env, "  env-token  ")
		defer os.Unsetenv(env)

		r := types.AccessTokenRef{Env: env}
		token, err := accesstoken.Resolve(accesstoken.Input{Ref: r})
		testhelpers.AssertNoError(t, err)
		testhelpers.AssertEqual(t, "env-token", token)
	})

	t.Run("env var empty and no file -> error", func(t *testing.T) {
		t.Parallel()
		env := "MCPJ_TEST_TOKEN_EMPTY"
		_ = os.Setenv(env, "   ")
		defer os.Unsetenv(env)

		r := types.AccessTokenRef{Env: env}
		_, err := accesstoken.Resolve(accesstoken.Input{Ref: r})
		testhelpers.AssertError(t, err)
	})

	t.Run("file is used when provided", func(t *testing.T) {
		t.Parallel()
		f, err := os.CreateTemp("", "mcpj-token-*")
		testhelpers.AssertNoError(t, err)
		_ = os.WriteFile(f.Name(), []byte("  file-token\n"), 0o600)
		defer os.Remove(f.Name())

		r := types.AccessTokenRef{File: f.Name()}
		token, err := accesstoken.Resolve(accesstoken.Input{Ref: r})
		testhelpers.AssertNoError(t, err)
		testhelpers.AssertEqual(t, "file-token", token)
	})

	t.Run("env empty but file present -> file used", func(t *testing.T) {
		t.Parallel()
		env := "MCPJ_TEST_TOKEN_EMPTY2"
		_ = os.Setenv(env, " ")
		defer os.Unsetenv(env)

		f, err := os.CreateTemp("", "mcpj-token-*")
		testhelpers.AssertNoError(t, err)
		_ = os.WriteFile(f.Name(), []byte("from-file"), 0o600)
		defer os.Remove(f.Name())

		r := types.AccessTokenRef{Env: env, File: f.Name()}
		token, err := accesstoken.Resolve(accesstoken.Input{Ref: r})
		testhelpers.AssertNoError(t, err)
		testhelpers.AssertEqual(t, "from-file", token)
	})

	t.Run("missing file -> error", func(t *testing.T) {
		t.Parallel()
		r := types.AccessTokenRef{File: "/no/such/file/xxxx"}
		_, err := accesstoken.Resolve(accesstoken.Input{Ref: r})
		testhelpers.AssertError(t, err)
	})

	t.Run("empty file -> error", func(t *testing.T) {
		t.Parallel()
		f, err := os.CreateTemp("", "mcpj-token-empty-*")
		testhelpers.AssertNoError(t, err)
		// leave file empty
		defer os.Remove(f.Name())

		r := types.AccessTokenRef{File: f.Name()}
		_, err = accesstoken.Resolve(accesstoken.Input{Ref: r})
		testhelpers.AssertError(t, err)
	})

	t.Run("nothing provided -> empty and no error", func(t *testing.T) {
		t.Parallel()
		r := types.AccessTokenRef{}
		token, err := accesstoken.Resolve(accesstoken.Input{Ref: r})
		testhelpers.AssertNoError(t, err)
		testhelpers.AssertEqual(t, "", token)
	})
}

func TestParseAllowListBasicCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", []string{}},
		{"single server", "server1", []string{"server1"}},
		{"multiple servers", "server1,server2,server3", []string{"server1", "server2", "server3"}},
		{"servers with spaces", "server1, server2 , server3", []string{"server1", "server2", "server3"}},
		{"servers with empty elements", "server1,,server2", []string{"server1", "server2"}},
		{"servers with only spaces", "server1,  ,server2", []string{"server1", "server2"}},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseAllowList(tc.input, &cobra.Command{})
			for i, srv := range got {
				testhelpers.AssertEqual(t, tc.expected[i], srv)
			}
		})
	}
}

func TestParseAllowListWildcardWarns(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	input := "serverA, " + types.AllowAllMcpServers + ", serverB"
	expected := []string{"serverA", types.AllowAllMcpServers, "serverB"}

	got := parseAllowList(input, cmd)
	for i, srv := range got {
		testhelpers.AssertEqual(t, expected[i], srv)
	}

	out := buf.String()
	testhelpers.AssertTrue(t, strings.Contains(out, "NOTE:"), "expected warning to contain NOTE:")
	testhelpers.AssertTrue(t, strings.Contains(out, "access to all MCP Servers"), "expected warning body")
}
