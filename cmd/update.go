package cmd

import (
	"fmt"

	"github.com/mcpjungle/mcpjungle/pkg/util"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update resources",
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
		"Note that you cannot update the name of a group once it is created.\n" +
		"Updating a group does not cause any downtime for the MCP clients relying on its endpoint.\n\n" +
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

	// if nothing was actually changed, inform the user and exit

	toolsAdded, toolsRemoved := util.DiffTools(resp.Old.IncludedTools, resp.New.IncludedTools)

	noChangeInTools := len(toolsAdded) == 0 && len(toolsRemoved) == 0
	if resp.Old.Description == resp.New.Description && noChangeInTools {
		cmd.Printf("No changes detected for Tool Group %s. Nothing was updated.\n", resp.Name)
		return nil
	}

	cmd.Printf("Tool Group %s updated successfully\n\n", resp.Name)

	if resp.Old.Description != resp.New.Description {
		cmd.Printf("* Description updated from:\n    %s\nto:\n    %s\n\n", resp.Old.Description, resp.New.Description)
	}

	if noChangeInTools {
		cmd.Println("* No changes in Tool list")
		return nil
	}

	if len(toolsRemoved) > 0 {
		cmd.Println("* Tools removed from the group:")
		for _, t := range toolsRemoved {
			cmd.Printf("    - %s\n", t)
		}
	} else {
		cmd.Println("* No tools were removed from the group")
	}
	cmd.Println()

	if len(toolsAdded) > 0 {
		cmd.Println("* Tools added to the group:")
		for _, t := range toolsAdded {
			cmd.Printf("    - %s\n", t)
		}
	} else {
		cmd.Println("* No tools were added to the group")
	}
	cmd.Println()

	return nil
}
