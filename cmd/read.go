package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var readResourceCmdServerName string

var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Read live content from MCP resources",
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "6",
	},
}

var readResourceCmd = &cobra.Command{
	Use:   "resource [uri]",
	Args:  cobra.ExactArgs(1),
	Short: "Read live resource content",
	Long: "Read live content from a resource.\n" +
		"Use the --server option to specify the MCP server to read the resource from.",
	RunE: runReadResource,
}

func init() {
	readResourceCmd.Flags().StringVar(
		&readResourceCmdServerName,
		"server",
		"",
		"Scope the resource read to a specific server",
	)

	readCmd.AddCommand(readResourceCmd)
	rootCmd.AddCommand(readCmd)
}

func runReadResource(cmd *cobra.Command, args []string) error {
	result, err := apiClient.ReadResource(args[0], readResourceCmdServerName)
	if err != nil {
		return fmt.Errorf("failed to read resource: %w", err)
	}

	cmd.Printf("Resource URI: %s\n\n", args[0])
	for i, content := range result.Contents {
		cmd.Printf("Content %d:\n", i+1)

		if uri, ok := content["uri"].(string); ok && uri != "" {
			cmd.Printf("URI: %s\n", uri)
		}
		if mimeType, ok := content["mimeType"].(string); ok && mimeType != "" {
			cmd.Printf("MIME Type: %s\n", mimeType)
		}

		if text, ok := content["text"].(string); ok {
			if json.Valid([]byte(text)) {
				var pretty any
				if err := json.Unmarshal([]byte(text), &pretty); err == nil {
					prettyBytes, _ := json.MarshalIndent(pretty, "", "  ")
					cmd.Printf("%s\n", string(prettyBytes))
				} else {
					cmd.Println(text)
				}
			} else {
				cmd.Println(text)
			}
		} else if blob, ok := content["blob"].(string); ok {
			data, err := base64.StdEncoding.DecodeString(blob)
			if err != nil {
				return fmt.Errorf("failed to decode blob resource content: %w", err)
			}
			filename := fmt.Sprintf("resource_%d.bin", time.Now().UnixNano())
			if err := os.WriteFile(filename, data, 0o644); err != nil {
				return fmt.Errorf("failed to write blob resource to disk: %w", err)
			}
			cmd.Printf("[Blob content saved as %s]\n", filename)
		}

		if i < len(result.Contents)-1 {
			cmd.Println()
		}
	}

	return nil
}
