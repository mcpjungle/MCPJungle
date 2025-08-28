package cmd

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage <name>",
	Short: "Get usage information for a MCP tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runGetToolUsage,
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "4",
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}

func runGetToolUsage(cmd *cobra.Command, args []string) error {
	t, err := apiClient.GetTool(args[0])
	if err != nil {
		return fmt.Errorf("failed to get tool '%s': %w", args[0], err)
	}

	fmt.Println(t.Name)
	fmt.Println(t.Description)

	if len(t.InputSchema.Properties) == 0 {
		fmt.Println("This tool does not require any input parameters.")
		return nil
	}

	fmt.Println()
	fmt.Println("Input Parameters:")
	for k, v := range t.InputSchema.Properties {
		requiredOrOptional := "optional"
		if slices.Contains(t.InputSchema.Required, k) {
			requiredOrOptional = "required"
		}

		boundary := strings.Repeat("=", len(k)+len(requiredOrOptional)+20)

		fmt.Println(boundary)
		fmt.Printf("%s (%s)\n", k, requiredOrOptional)

		j, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			// Simply print the raw object if we fail to marshal it
			fmt.Println(v)
		} else {
			fmt.Println(string(j))
		}
		fmt.Println(boundary)

		fmt.Println()
	}

	return nil
}
