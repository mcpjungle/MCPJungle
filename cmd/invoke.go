package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	invokeCmdInput     string
	invokeCmdGroupName string
)

var invokeToolCmd = &cobra.Command{
	Use:   "invoke <name>",
	Short: "Invoke a tool",
	Long:  "Invokes a tool supplied by a registered MCP server",
	Args:  cobra.ExactArgs(1),
	RunE:  runInvokeTool,
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "5",
	},
}

func init() {
	invokeToolCmd.Flags().StringVar(&invokeCmdInput, "input", "{}", "valid JSON payload")
	invokeToolCmd.Flags().StringVar(&invokeCmdGroupName, "group", "", "invoke the tool within a tool group's context")
	rootCmd.AddCommand(invokeToolCmd)
}

func getTextContent(c map[string]any) (string, error) {
	textContent, ok := c["text"].(string)
	if !ok {
		return "", fmt.Errorf("text content item does not have a 'text' field: %v", c)
	}
	return textContent, nil
}

func getImageContent(c map[string]any) ([]byte, string, error) {
	dataStr, ok := c["data"].(string)
	if !ok {
		return nil, "", fmt.Errorf("image content item does not have a valid 'data' field: %v", c)
	}
	mimeType, ok := c["mimeType"].(string)
	if !ok {
		return nil, "", fmt.Errorf("image content item does not have a valid 'mimeType' field: %v", c)
	}

	// Decode base64 image data
	imgData, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64 image data: %w", err)
	}

	// Determine file extension from MIME type
	ext := ".img"
	switch mimeType {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/gif":
		ext = ".gif"
	}

	return imgData, ext, nil
}

func getAudioContent(c map[string]any) ([]byte, string, error) {
	dataStr, ok := c["data"].(string)
	if !ok {
		return nil, "", fmt.Errorf("audio content item does not have a valid 'data' field: %v", c)
	}
	mimeType, ok := c["mimeType"].(string)
	if !ok {
		return nil, "", fmt.Errorf("audio content item does not have a valid 'mimeType' field: %v", c)
	}

	// Decode base64 audio data
	audioData, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64 audio data: %w", err)
	}

	// Determine file extension from MIME type
	ext := ".audio"
	switch mimeType {
	case "audio/mpeg":
		ext = ".mp3"
	case "audio/wav":
		ext = ".wav"
	case "audio/ogg":
		ext = ".ogg"
	}

	return audioData, ext, nil
}

func runInvokeTool(cmd *cobra.Command, args []string) error {
	var input map[string]any
	if err := json.Unmarshal([]byte(invokeCmdInput), &input); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	toolName := args[0]

	// If group is specified, validate that the tool is in the group
	if invokeCmdGroupName != "" {
		group, err := apiClient.GetToolGroup(invokeCmdGroupName)
		if err != nil {
			return fmt.Errorf("failed to get tool group '%s': %w", invokeCmdGroupName, err)
		}

		// Check if the tool is included in the group
		toolInGroup := false
		for _, includedTool := range group.IncludedTools {
			if includedTool == toolName {
				toolInGroup = true
				break
			}
		}

		if !toolInGroup {
			return fmt.Errorf("tool '%s' is not available in group '%s'", toolName, invokeCmdGroupName)
		}

		cmd.Printf("Invoking tool '%s' from group '%s'\n", toolName, invokeCmdGroupName)
		if group.Description != "" {
			cmd.Printf("Group description: %s\n", group.Description)
		}
		cmd.Println()
	}

	result, err := apiClient.InvokeTool(toolName, input)
	if err != nil {
		return fmt.Errorf("failed to invoke tool: %w", err)
	}

	if result.IsError {
		cmd.Println("The tool returned an error:")
		for k, v := range result.Meta {
			cmd.Printf("%s: %v\n", k, v)
		}
	} else {
		cmd.Println("Response from tool:")
	}

	// result Content needs to be printed regardless of whether the tool returned an error or not
	// because it may contain useful information
	cmd.Println()
	for _, c := range result.Content {
		cType, ok := c["type"]
		if !ok {
			return fmt.Errorf("content item does not have a 'type' field: %v", c)
		}
		cmd.Printf("[Content type: %s]\n", cType)

		switch cType {
		case "text":
			textContent, err := getTextContent(c)
			if err != nil {
				return err
			}
			cmd.Println(textContent)

		case "image":
			imgData, ext, err := getImageContent(c)
			if err != nil {
				return err
			}
			filename := fmt.Sprintf("image_%d%s", time.Now().UnixNano(), ext)
			if err := os.WriteFile(filename, imgData, 0o644); err != nil {
				return fmt.Errorf("failed to write image to disk: %w", err)
			}
			cmd.Printf("[Image saved as %s]\n", filename)

		case "audio":
			audioData, ext, err := getAudioContent(c)
			if err != nil {
				return err
			}
			filename := fmt.Sprintf("audio_%d%s", time.Now().UnixNano(), ext)
			if err := os.WriteFile(filename, audioData, 0o644); err != nil {
				return fmt.Errorf("failed to write audio to disk: %w", err)
			}
			cmd.Printf("[Audio saved as %s]\n", filename)
		}
	}

	return nil
}
