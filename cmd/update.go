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
		"CAUTION: If you remove any tools from the configuration (by removing them from include or adding them to exclude), " +
		"calling update will immediately remove them from the group. " +
		"They will no longer be accessible by MCP clients using the group's MCP server.",
	RunE: runUpdateGroup,
}

var updateToolGroupConfigFilePath string

func init() {
	updateToolGroupCmd.Flags().StringVarP(
		&updateToolGroupConfigFilePath,
		"conf",
		"c",
		"",
		"Path to new JSON configuration file for the Tool Group",
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

	// Check if anything was actually changed
	toolsAdded, toolsRemoved := util.DiffTools(resp.Old.IncludedTools, resp.New.IncludedTools)
	serversAdded, serversRemoved := util.DiffTools(resp.Old.IncludedServers, resp.New.IncludedServers)
	excludedAdded, excludedRemoved := util.DiffTools(resp.Old.ExcludedTools, resp.New.ExcludedTools)

	noChangeInTools := len(toolsAdded) == 0 && len(toolsRemoved) == 0
	noChangeInServers := len(serversAdded) == 0 && len(serversRemoved) == 0
	noChangeInExcluded := len(excludedAdded) == 0 && len(excludedRemoved) == 0

	if resp.Old.Description == resp.New.Description && noChangeInTools && noChangeInServers && noChangeInExcluded {
		cmd.Printf("No changes detected for Tool Group %s. Nothing was updated.\n", resp.Name)
		return nil
	}

	cmd.Printf("Tool Group %s updated successfully\n\n", resp.Name)

	if resp.Old.Description != resp.New.Description {
		cmd.Printf("* Description updated from:\n    %s\nto:\n    %s\n\n", resp.Old.Description, resp.New.Description)
	}

	// Report changes in included_tools
	if noChangeInTools {
		cmd.Println("* No changes in included_tools")
	} else {
		if len(toolsRemoved) > 0 {
			cmd.Println("* Tools removed from included_tools:")
			for _, t := range toolsRemoved {
				cmd.Printf("    - %s\n", t)
			}
		}
		if len(toolsAdded) > 0 {
			cmd.Println("* Tools added to included_tools:")
			for _, t := range toolsAdded {
				cmd.Printf("    - %s\n", t)
			}
		}
	}
	cmd.Println()

	// Report changes in included_servers
	if !noChangeInServers {
		if len(serversRemoved) > 0 {
			cmd.Println("* Servers removed from included_servers:")
			for _, s := range serversRemoved {
				cmd.Printf("    - %s\n", s)
			}
		}
		if len(serversAdded) > 0 {
			cmd.Println("* Servers added to included_servers:")
			for _, s := range serversAdded {
				cmd.Printf("    - %s\n", s)
			}
		}
		cmd.Println()
	}

	// Report changes in excluded_tools
	if !noChangeInExcluded {
		if len(excludedRemoved) > 0 {
			cmd.Println("* Tools removed from excluded_tools:")
			for _, e := range excludedRemoved {
				cmd.Printf("    - %s\n", e)
			}
		}
		if len(excludedAdded) > 0 {
			cmd.Println("* Tools added to excluded_tools:")
			for _, e := range excludedAdded {
				cmd.Printf("    - %s\n", e)
			}
		}
		cmd.Println()
	}

	return nil
}
