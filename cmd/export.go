package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const defaultExportTargetDir = ".mcpjungle"

const (
	exportMcpServersDir = "servers"
	exportToolGroupsDir = "groups"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export configuration files of all entities",
	Long: "This command creates configuration files for all entities (mcp servers, groups) that exist in mcpjungle.\n" +
		"This is useful when you want to track all the entities registered in mcpjungle as code.\n" +
		fmt.Sprintf("By default, the configurations are exported to a directory named %s in the current working directory.\n\n", defaultExportTargetDir) +
		"NOTE: In enterprise mode, you must be an admin to export all configurations successfully.",
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "9",
	},
	RunE: runExport,
}

var exportCmdTargetDir string

func init() {
	exportCmd.Flags().StringVarP(
		&exportCmdTargetDir,
		"dir",
		"d",
		defaultExportTargetDir,
		"Directory to export configuration files to",
	)

	rootCmd.AddCommand(exportCmd)
}

// resolveTargetDirForExport determines the target directory for to export the configurations to.
// The "~" prefix is expanded to home directory, if it exists. The directory is created if it doesn't exist.
func resolveTargetDirForExport() (string, error) {
	// determine target directory (flag overrides default)
	targetDir := exportCmdTargetDir
	if targetDir == "" {
		targetDir = defaultExportTargetDir
	}

	// expand ~ to user home
	if strings.HasPrefix(targetDir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if targetDir == "~" {
			targetDir = home
		} else if strings.HasPrefix(targetDir, "~/") {
			targetDir = filepath.Join(home, targetDir[2:])
		}
	}

	// make absolute and clean
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return "", err
	}
	targetDir = filepath.Clean(absDir)

	// create the directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	// ensure the target directory is empty
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to read contents of target directory %s: %w", targetDir, err)
	}
	if len(entries) > 0 {
		return "", fmt.Errorf("target directory %s is not empty", targetDir)
	}

	return targetDir, nil
}

func writeJSONConfigFile(entityDir, entityName string, entity any) error {
	filename := filepath.Join(entityDir, filepath.Base(entityName)+".json")
	data, err := json.MarshalIndent(entity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize entity %s/%s: %w", entityDir, entityName, err)
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return fmt.Errorf("failed to write entity file %s: %w", filename, err)
	}
	return nil
}

func runExport(cmd *cobra.Command, args []string) error {
	targetDir, err := resolveTargetDirForExport()
	if err != nil {
		return fmt.Errorf("failed to resolve target directory for export: %w", err)
	}

	cmd.Printf("Creating subdirectories inside %s\n\n", targetDir)

	groupsDir := filepath.Join(targetDir, exportToolGroupsDir)
	if err := os.Mkdir(groupsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create groups directory: %w", err)
	}
	serversDir := filepath.Join(targetDir, exportMcpServersDir)
	if err := os.Mkdir(serversDir, 0o755); err != nil {
		return fmt.Errorf("failed to create mcp servers directory: %w", err)
	}

	cmd.Println("Fetching Tool Group configurations...")

	groups, gErr := apiClient.GetToolGroupConfigs()
	if gErr != nil {
		cmd.Printf("warning: failed to fetch tool group configurations: %v\n", gErr)
	} else {
		if len(groups) == 0 {
			cmd.Println("No Tool Groups found.")
		} else {
			cmd.Printf("Writing Tool Groups configurations to %s\n", groupsDir)

			for _, g := range groups {
				if err := writeJSONConfigFile(groupsDir, g.Name, g); err != nil {
					return err
				}
			}
		}
	}

	cmd.Println("Fetching MCP Server configurations...")

	servers, sErr := apiClient.GetServerConfigs()
	if sErr != nil {
		cmd.Printf("warning: failed to fetch mcp server configurations: %v", sErr)
	} else {
		if len(servers) == 0 {
			cmd.Println("No MCP Servers found.")
		} else {
			cmd.Printf("Writing MCP Server configurations to %s\n", serversDir)

			for _, s := range servers {
				if err := writeJSONConfigFile(serversDir, s.Name, s); err != nil {
					return err
				}
			}
		}
	}

	cmd.Println("\nExport complete!")

	return nil
}
