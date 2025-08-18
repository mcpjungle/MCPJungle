# Makefile for MCPJungle
# Go project with build and linting commands

# Variables
BINARY_NAME=mcpjungle
BUILD_DIR=build
MAIN_PATH=./main.go

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build

# Linting tools
GOLANGCI_LINT=golangci-lint

# Default target
.DEFAULT_GOAL := build

# Build target
.PHONY: build
build: ## Build the application
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Linting target
.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	@$(GOLANGCI_LINT) run ./...