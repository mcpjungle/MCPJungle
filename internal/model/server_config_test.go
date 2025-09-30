package model

import (
	"testing"

	"gorm.io/gorm"
)

func TestIsEnterpriseMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     ServerMode
		expected bool
	}{
		{
			name:     "Enterprise mode",
			mode:     ModeEnterprise,
			expected: true,
		},
		{
			name:     "Prod mode (deprecated alias)",
			mode:     ModeProd,
			expected: true,
		},
		{
			name:     "Development mode",
			mode:     ModeDev,
			expected: false,
		},
		{
			name:     "Unknown mode",
			mode:     ServerMode("unknown"),
			expected: false,
		},
		{
			name:     "Empty string mode",
			mode:     ServerMode(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEnterpriseMode(tt.mode)
			if result != tt.expected {
				t.Errorf("IsEnterpriseMode(%q) = %v; want %v", tt.mode, result, tt.expected)
			}
		})
	}
}

func TestServerConfig_BeforeSave(t *testing.T) {
	tests := []struct {
		name      string
		mode      ServerMode
		wantErr   bool
		wantFinal ServerMode
	}{
		{
			name:      "Valid development mode",
			mode:      ModeDev,
			wantErr:   false,
			wantFinal: ModeDev,
		},
		{
			name:      "Valid enterprise mode",
			mode:      ModeEnterprise,
			wantErr:   false,
			wantFinal: ModeEnterprise,
		},
		{
			name:      "Deprecated prod mode (should convert to enterprise)",
			mode:      ModeProd,
			wantErr:   false,
			wantFinal: ModeEnterprise,
		},
		{
			name:      "Unknown mode",
			mode:      ServerMode("unknown"),
			wantErr:   true,
			wantFinal: ServerMode("unknown"),
		},
		{
			name:      "Empty mode",
			mode:      ServerMode(""),
			wantErr:   true,
			wantFinal: ServerMode(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ServerConfig{Mode: tt.mode}
			err := cfg.BeforeSave(&gorm.DB{})
			if (err != nil) != tt.wantErr {
				t.Errorf("BeforeSave() error = %v, wantErr %v", err, tt.wantErr)
			}
			if cfg.Mode != tt.wantFinal {
				t.Errorf("After BeforeSave(), Mode = %v, want %v", cfg.Mode, tt.wantFinal)
			}
		})
	}
}
