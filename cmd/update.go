package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update resources like Tool Groups",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "8",
	},
}

var updateToolGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "Update a tool group",
	Long: "Update an existing Tool Group\n" +
		"This option allows you to supply the modified configuration file of an existing Tool group.\n" +
		"The new configuration completely overrides the existing one.\n" +
		"Updating a group does not cause any downtime for the MCP clients relying on the group.\n\n" +
		"CAUTION: If you remove any tools from the configuration, calling update will immediately remove them from " +
		"the group. They will no longer be accessible by MCP clients using the group.",
	RunE: runUpdateGroup,
}

var updateToolGroupConfigFilePath string

func init() {
	updateToolGroupCmd.Flags().StringVarP(
		&updateToolGroupConfigFilePath,
		"conf",
		"c",
		"",
		"Path to new JSON configuration file for the Tool Group.\n",
	)
	_ = updateToolGroupCmd.MarkFlagRequired("conf")

	updateCmd.AddCommand(updateToolGroupCmd)
	rootCmd.AddCommand(updateCmd)
}

func runUpdateGroup(cmd *cobra.Command, args []string) error {
	updatedConf, err := readToolGroupConfig(updateToolGroupConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", updateToolGroupConfigFilePath, err)
	}

	resp, err := apiClient.UpdateToolGroup(updatedConf)
	if err != nil {
		return fmt.Errorf("failed to update tool group %s: %w", updatedConf.Name, err)
	}

	cmd.Printf("Tool Group %s updated successfully\n", resp.Name)

	if resp.Old.Description != resp.New.Description {
		cmd.Printf("Description updated from:\n    %s\nto:\n    %s\n\n", resp.Old.Description, resp.New.Description)
	}

	cmd.Println(resp.Old.IncludedTools)
	cmd.Println(resp.New.IncludedTools)

	return nil
}
