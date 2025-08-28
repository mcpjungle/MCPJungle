package cmd

import (
	"errors"
	"fmt"

	"github.com/mcpjungle/mcpjungle/cmd/config"
	"github.com/spf13/cobra"
)

var initServerCmd = &cobra.Command{
	Use:   "init-server",
	Short: "Initialize the MCPJungle Server (for Production Mode only)",
	Long: "If the MCPJungle Server was started in Production Mode, use this command to initialize the server.\n" +
		"Initialization is required before you can use the server.\n",
	RunE: runInitServer,
	Annotations: map[string]string{
		"group": string(subCommandGroupAdvanced),
		"order": "5",
	},
}

func init() {
	rootCmd.AddCommand(initServerCmd)
}

func runInitServer(cmd *cobra.Command, args []string) error {
	fmt.Println("Initializing the MCPJungle Server in Production Mode...")
	resp, err := apiClient.InitServer()
	if err != nil {
		return fmt.Errorf("failed to initialize the server: %w", err)
	}

	if resp.AdminAccessToken == "" {
		return errors.New("server initialization failed: no admin access token received")
	}

	// Create new client configuration
	cfg := &config.ClientConfig{
		AccessToken: resp.AdminAccessToken,
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to create client configuration: %w", err)
	}

	cfgPath, err := config.AbsPath()
	if err != nil {
		return fmt.Errorf("failed to get client configuration path: %w", err)
	}
	fmt.Println("Your Admin access token has been saved to", cfgPath)

	fmt.Println("All done!")
	return nil
}
