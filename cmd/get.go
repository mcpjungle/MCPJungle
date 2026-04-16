package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get entities like Prompts and Tool Groups",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "1",
	},
}

var (
	getPromptArgs      map[string]string
	getResourceCmdRead bool
)

var getGroupCmd = &cobra.Command{
	Use:   "group [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Get information about a specific Tool Group",
	Long: "Get information about a specific Tool Group by name.\n" +
		"This returns the configuration of the Tool Group including which tools are included.\n",
	RunE: runGetGroup,
}

var getPromptCmd = &cobra.Command{
	Use:   "prompt [name]",
	Args:  cobra.ExactArgs(1),
	Short: "Get a prompt template",
	Long: "Retrieve a prompt template from an MCP server with optional arguments.\n" +
		"The prompt will be rendered with the provided arguments and returned as structured messages.",
	Example: `  # Get a basic prompt
  mcpjungle get prompt github__code-review

  # Get a prompt with arguments
  mcpjungle get prompt github__code-review --arg code="def hello(): print('world')" --arg language="python"`,
	RunE: runGetPrompt,
}

var getResourceCmd = &cobra.Command{
	Use:   "resource [uri]",
	Args:  cobra.ExactArgs(1),
	Short: "Get resource metadata",
	Long: "Get resource metadata by URI.\n" +
		"Use --read to read the resource content instead.",
	RunE: runGetResource,
}

func init() {
	getPromptCmd.Flags().StringToStringVar(
		&getPromptArgs,
		"arg",
		nil,
		"Arguments to pass to the prompt in the form of 'key=value' (this flag can be specified multiple times)",
	)
	getResourceCmd.Flags().BoolVar(
		&getResourceCmdRead,
		"read",
		false,
		"Read the resource content instead of showing metadata",
	)

	getCmd.AddCommand(getGroupCmd)
	getCmd.AddCommand(getPromptCmd)
	getCmd.AddCommand(getResourceCmd)
	rootCmd.AddCommand(getCmd)
}

func runGetGroup(cmd *cobra.Command, args []string) error {
	name := args[0]
	group, err := apiClient.GetToolGroup(name)
	if err != nil {
		return fmt.Errorf("failed to get tool group: %w", err)
	}

	cmd.Println(group.Name)
	if group.Description != "" {
		cmd.Println()
		cmd.Println("Description: " + group.Description)
	}

	cmd.Println()
	cmd.Println("MCP Server streamable http endpoint:")
	cmd.Println(group.StreamableHTTPEndpoint)
	cmd.Println()
	cmd.Println("MCP server SSE endpoints:")
	cmd.Println(group.SSEEndpoint)
	cmd.Println(group.SSEMessageEndpoint)
	cmd.Println()

	if len(group.IncludedTools) == 0 {
		cmd.Println("Included Tools: None")
	} else {
		cmd.Println("Included Tools:")
		for i, t := range group.IncludedTools {
			cmd.Printf("%d. %s\n", i+1, t)
			// TODO: Also show whether the tool is still active, disabled, or deleted at the moment
			// ie, is it practically available as part of this group?
		}
	}
	cmd.Println()

	if len(group.IncludedServers) == 0 {
		cmd.Println("Included Servers: None")
	} else {
		cmd.Println("Included Servers:")
		for i, s := range group.IncludedServers {
			cmd.Printf("%d. %s\n", i+1, s)
		}
	}
	cmd.Println()

	if len(group.ExcludedTools) == 0 {
		cmd.Println("Excluded Tools: None")
	} else {
		cmd.Println("Excluded Tools:")
		for i, t := range group.ExcludedTools {
			cmd.Printf("%d. %s\n", i+1, t)
		}
	}
	cmd.Println()

	cmd.Println(
		"NOTE: If a tool in this group is disabled globally or has been deleted, " +
			"then it will not be available via the group's MCP endpoint.",
	)

	return nil
}

func runGetPrompt(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Convert CLI args to proper format
	arguments := make(map[string]string)
	for k, v := range getPromptArgs {
		arguments[k] = v
	}

	result, err := apiClient.GetPromptWithArgs(name, arguments)
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	// Pretty print the result
	cmd.Printf("Prompt: %s\n", name)
	if result.Description != "" {
		cmd.Printf("Description: %s\n", result.Description)
	}
	cmd.Println("\nGenerated Messages:")
	cmd.Println("=" + strings.Repeat("=", 50))

	for i, message := range result.Messages {
		cmd.Printf("\nMessage %d (%s):\n", i+1, message.Role)
		cmd.Println("-" + strings.Repeat("-", 30))

		// Format the content nicely
		contentBytes, err := json.MarshalIndent(message.Content, "", "  ")
		if err != nil {
			cmd.Printf("Content: %+v\n", message.Content)
		} else {
			cmd.Printf("Content: %s\n", string(contentBytes))
		}
	}

	return nil
}

func runGetResource(cmd *cobra.Command, args []string) error {
	resource, err := apiClient.GetResource(args[0])
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	if getResourceCmdRead {
		return runGetResourceRead(cmd, resource)
	}

	cmd.Printf("Resource: %s\n", resource.Name)
	cmd.Printf("URI: %s\n", resource.URI)
	if resource.MIMEType != "" {
		cmd.Printf("MIME Type: %s\n", resource.MIMEType)
	}
	if resource.Description != "" {
		cmd.Printf("Description: %s\n", resource.Description)
	}
	if resource.Enabled {
		cmd.Println("Status: ENABLED")
	} else {
		cmd.Println("Status: DISABLED")
	}

	return nil
}

func runGetResourceRead(cmd *cobra.Command, resource *types.Resource) error {
	result, err := apiClient.ReadResource(resource.URI)
	if err != nil {
		return fmt.Errorf("failed to read resource: %w", err)
	}

	cmd.Printf("Resource: %s\n", resource.Name)
	cmd.Printf("URI: %s\n\n", resource.URI)
	for i, content := range result.Contents {
		cmd.Printf("Content %d:\n", i+1)

		if resourceURI, ok := content["uri"].(string); ok && resourceURI != "" {
			cmd.Printf("URI: %s\n", resourceURI)
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
