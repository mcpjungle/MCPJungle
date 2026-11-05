package cmd

import (
	"fmt"

	"github.com/mcpjungle/mcpjungle/internal/service/configvalidation"
	"github.com/spf13/cobra"
)

var validateCmdConfigType string

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Args:  cobra.ExactArgs(1),
	Short: "Validate an MCPJungle JSON config file",
	Long: "Validate an MCPJungle JSON config file before using it with register or create commands.\n" +
		"By default, the command infers the config type from its fields. Use --type to validate\n" +
		"ambiguous files as one of: mcp-server, tool-group, mcp-client, user.",
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "6",
	},
	RunE: runValidateConfig,
}

func init() {
	validateCmd.Flags().StringVar(
		&validateCmdConfigType,
		"type",
		"",
		"Config type to validate as: mcp-server, tool-group, mcp-client, or user",
	)

	rootCmd.AddCommand(validateCmd)
}

func runValidateConfig(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	var (
		result *configvalidation.Result
		err    error
	)
	if validateCmdConfigType == "" {
		result, err = configvalidation.ValidateFile(filePath)
	} else {
		result, err = configvalidation.ValidateFileAs(filePath, configvalidation.ConfigType(validateCmdConfigType))
	}
	if err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	cmd.Printf("%s is a valid MCPJungle config (%s)\n", filePath, result.Type)
	return nil
}
