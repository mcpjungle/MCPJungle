package cmd

import (
	"strings"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

func TestReadMcpServerConfigJson(t *testing.T) {
	input := `{
	"name": "test_server",
	"transport": "stdio",
	"description": "Test JSON server",
	"command": "node",
	"args": ["server.js"],
	"env": {"NODE_ENV": "test"}
}`

	reader := strings.NewReader(input)
	got, err := readMcpServerConfigJson(reader)
	if err != nil {
		t.Fatalf("readMcpServerConfigJson() error = %v", err)
	}

	expected := types.RegisterServerInput{
		Name:        "test_server",
		Transport:   "stdio",
		Description: "Test JSON server",
		Command:     "node",
		Args:        []string{"server.js"},
		Env:         map[string]string{"NODE_ENV": "test"},
	}

	if got.Name != expected.Name {
		t.Errorf("Name = %q, want %q", got.Name, expected.Name)
	}
	if got.Transport != expected.Transport {
		t.Errorf("Transport = %q, want %q", got.Transport, expected.Transport)
	}
	if got.Description != expected.Description {
		t.Errorf("Description = %q, want %q", got.Description, expected.Description)
	}
	if got.Command != expected.Command {
		t.Errorf("Command = %q, want %q", got.Command, expected.Command)
	}
	if len(got.Args) != len(expected.Args) || got.Args[0] != expected.Args[0] {
		t.Errorf("Args = %v, want %v", got.Args, expected.Args)
	}
	if got.Env["NODE_ENV"] != expected.Env["NODE_ENV"] {
		t.Errorf("Env = %v, want %v", got.Env, expected.Env)
	}
}

func TestReadMcpServerConfigYaml(t *testing.T) {
	input := `name: test_server
transport: stdio
description: Test YAML server
command: node
args:
  - server.js
env:
  NODE_ENV: test`

	reader := strings.NewReader(input)
	got, err := readMcpServerConfigYaml(reader)
	if err != nil {
		t.Fatalf("readMcpServerConfigYaml() error = %v", err)
	}

	expected := types.RegisterServerInput{
		Name:        "test_server",
		Transport:   "stdio",
		Description: "Test YAML server",
		Command:     "node",
		Args:        []string{"server.js"},
		Env:         map[string]string{"NODE_ENV": "test"},
	}

	if got.Name != expected.Name {
		t.Errorf("Name = %q, want %q", got.Name, expected.Name)
	}
	if got.Transport != expected.Transport {
		t.Errorf("Transport = %q, want %q", got.Transport, expected.Transport)
	}
	if got.Description != expected.Description {
		t.Errorf("Description = %q, want %q", got.Description, expected.Description)
	}
	if got.Command != expected.Command {
		t.Errorf("Command = %q, want %q", got.Command, expected.Command)
	}
	if len(got.Args) != len(expected.Args) || got.Args[0] != expected.Args[0] {
		t.Errorf("Args = %v, want %v", got.Args, expected.Args)
	}
	if got.Env["NODE_ENV"] != expected.Env["NODE_ENV"] {
		t.Errorf("Env = %v, want %v", got.Env, expected.Env)
	}
}
