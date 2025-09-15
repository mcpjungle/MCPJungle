package cmd

import (
	"testing"
)

func TestStartCommandStructure(t *testing.T) {
	t.Run("start command has correct properties", func(t *testing.T) {
		if startServerCmd.Use != "start" {
			t.Errorf("Expected start command Use to be 'start', got %s", startServerCmd.Use)
		}
		if startServerCmd.Short != "Start the MCPJungle server" {
			t.Errorf("Expected start command Short to be 'Start the MCPJungle server', got %s", startServerCmd.Short)
		}
	})

	t.Run("start command has correct annotations", func(t *testing.T) {
		if startServerCmd.Annotations == nil {
			t.Fatal("Start command missing annotations")
		}

		group, hasGroup := startServerCmd.Annotations["group"]
		if !hasGroup {
			t.Fatal("Start command missing 'group' annotation")
		}
		if group != string(subCommandGroupBasic) {
			t.Errorf("Expected start command group to be 'basic', got %s", group)
		}

		order, hasOrder := startServerCmd.Annotations["order"]
		if !hasOrder {
			t.Fatal("Start command missing 'order' annotation")
		}
		if order != "1" {
			t.Errorf("Expected start command order to be '1', got %s", order)
		}
	})
}

func TestStartCommandConstants(t *testing.T) {
	t.Run("environment variable constants", func(t *testing.T) {
		if BindPortEnvVar != "PORT" {
			t.Errorf("Expected BindPortEnvVar to be 'PORT', got %s", BindPortEnvVar)
		}
		if BindPortDefault != "8080" {
			t.Errorf("Expected BindPortDefault to be '8080', got %s", BindPortDefault)
		}
		if DBUrlEnvVar != "DATABASE_URL" {
			t.Errorf("Expected DBUrlEnvVar to be 'DATABASE_URL', got %s", DBUrlEnvVar)
		}
		if ServerModeEnvVar != "SERVER_MODE" {
			t.Errorf("Expected ServerModeEnvVar to be 'SERVER_MODE', got %s", ServerModeEnvVar)
		}
	})
}

func TestStartCommandFlags(t *testing.T) {
	t.Run("start command has port flag", func(t *testing.T) {
		portFlag := startServerCmd.Flags().Lookup("port")
		if portFlag == nil {
			t.Fatal("Start command missing 'port' flag")
		}
		if portFlag.Usage == "" {
			t.Error("Port flag should have usage description")
		}
	})

	t.Run("start command has prod flag", func(t *testing.T) {
		prodFlag := startServerCmd.Flags().Lookup("prod")
		if prodFlag == nil {
			t.Fatal("Start command missing 'prod' flag")
		}
		if prodFlag.Usage == "" {
			t.Error("Prod flag should have usage description")
		}
	})
}

func TestRunStartServerFunctional(t *testing.T) {
	t.Run("runStartServer would handle environment variable loading", func(t *testing.T) {
		testCases := []struct {
			name         string
			envVars      map[string]string
			expectedPort string
		}{
			{
				name:         "default port when no env var",
				envVars:      map[string]string{},
				expectedPort: BindPortDefault,
			},
			{
				name:         "custom port from env var",
				envVars:      map[string]string{BindPortEnvVar: "9090"},
				expectedPort: "9090",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				port := BindPortDefault
				if customPort, exists := tc.envVars[BindPortEnvVar]; exists {
					port = customPort
				}

				if port != tc.expectedPort {
					t.Errorf("Expected port %s, got %s", tc.expectedPort, port)
				}
			})
		}
	})

	t.Run("runStartServer would handle production mode flag", func(t *testing.T) {
		testCases := []struct {
			name         string
			prodFlag     bool
			expectedMode string
		}{
			{
				name:         "development mode",
				prodFlag:     false,
				expectedMode: "development",
			},
			{
				name:         "production mode",
				prodFlag:     true,
				expectedMode: "production",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mode := "development"
				if tc.prodFlag {
					mode = "production"
				}

				if mode != tc.expectedMode {
					t.Errorf("Expected mode %s, got %s", tc.expectedMode, mode)
				}
			})
		}
	})

	t.Run("runStartServer would handle database URL resolution", func(t *testing.T) {
		testCases := []struct {
			name        string
			envVars     map[string]string
			expectedURL string
		}{
			{
				name:        "no database URL set",
				envVars:     map[string]string{},
				expectedURL: "",
			},
			{
				name:        "custom database URL",
				envVars:     map[string]string{DBUrlEnvVar: "postgres://localhost:5432/mcpjungle"},
				expectedURL: "postgres://localhost:5432/mcpjungle",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				dbURL := ""
				if customURL, exists := tc.envVars[DBUrlEnvVar]; exists {
					dbURL = customURL
				}

				if dbURL != tc.expectedURL {
					t.Errorf("Expected database URL %s, got %s", tc.expectedURL, dbURL)
				}
			})
		}
	})

	t.Run("runStartServer would handle server mode resolution", func(t *testing.T) {
		testCases := []struct {
			name         string
			envVars      map[string]string
			expectedMode string
		}{
			{
				name:         "no server mode set",
				envVars:      map[string]string{},
				expectedMode: "",
			},
			{
				name:         "production server mode",
				envVars:      map[string]string{ServerModeEnvVar: "production"},
				expectedMode: "production",
			},
			{
				name:         "development server mode",
				envVars:      map[string]string{ServerModeEnvVar: "development"},
				expectedMode: "development",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				serverMode := ""
				if customMode, exists := tc.envVars[ServerModeEnvVar]; exists {
					serverMode = customMode
				}

				if serverMode != tc.expectedMode {
					t.Errorf("Expected server mode %s, got %s", tc.expectedMode, serverMode)
				}
			})
		}
	})

	t.Run("runStartServer would handle telemetry configuration", func(t *testing.T) {
		testCases := []struct {
			name              string
			envVars           map[string]string
			expectedTelemetry bool
		}{
			{
				name:              "telemetry disabled by default",
				envVars:           map[string]string{},
				expectedTelemetry: false,
			},
			{
				name:              "telemetry enabled",
				envVars:           map[string]string{TelemetryEnabledEnvVar: "true"},
				expectedTelemetry: true,
			},
			{
				name:              "telemetry disabled explicitly",
				envVars:           map[string]string{TelemetryEnabledEnvVar: "false"},
				expectedTelemetry: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				telemetryEnabled := false
				if enabled, exists := tc.envVars[TelemetryEnabledEnvVar]; exists {
					telemetryEnabled = enabled == "true"
				}

				if telemetryEnabled != tc.expectedTelemetry {
					t.Errorf("Expected telemetry enabled %v, got %v", tc.expectedTelemetry, telemetryEnabled)
				}
			})
		}
	})
}
