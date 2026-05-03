PORT     ?= 8080
MODE     ?= enterprise
AIR      := $(HOME)/go/bin/air
BINARY   := /tmp/mcpjungle

.PHONY: build build-ui start dev dev-ui restart restart-core test lint service-install service-stop service-logs

## start: clone-and-run — install npm deps, build UI + Go binary, start server
start: build-ui build
	$(BINARY) start --port $(PORT) --$(MODE) --ui &
	@sleep 1 && curl -sf http://localhost:$(PORT)/health && echo " server up at http://localhost:$(PORT)"

## build-ui: install npm deps and compile the dashboard UI
build-ui:
	cd ui && npm install && npm run build

## build: compile the Go binary to /tmp/mcpjungle
build:
	go build -o $(BINARY) .

## restart: rebuild Go binary and hot-swap the running server (UI already built)
restart: build
	@PID=$$(pgrep -x mcpjungle 2>/dev/null); \
	if [ -n "$$PID" ]; then \
		echo "killing PID $$PID"; kill $$PID; sleep 1; \
	fi
	$(BINARY) start --port $(PORT) --$(MODE) --ui &
	@sleep 1 && curl -sf http://localhost:$(PORT)/health && echo " server up"

## restart-core: rebuild and hot-swap — gateway only, no dashboard UI
restart-core: build
	@PID=$$(pgrep -x mcpjungle 2>/dev/null); \
	if [ -n "$$PID" ]; then \
		echo "killing PID $$PID"; kill $$PID; sleep 1; \
	fi
	$(BINARY) start --port $(PORT) --$(MODE) &
	@sleep 1 && curl -sf http://localhost:$(PORT)/health && echo " server up (core only)"

## dev: watch Go files and auto-restart on changes — gateway + dashboard UI (uses air)
dev:
	@echo "Starting MCPJungle in watch mode ($(MODE), :$(PORT)) with UI"
	@echo "Edit any .go file to trigger a rebuild."
	@echo ""
	$(AIR) -c .air.toml

## dev-core: watch Go files and auto-restart — gateway only, no UI
dev-core:
	@echo "Starting MCPJungle core in watch mode ($(MODE), :$(PORT)) — no UI"
	@echo "Edit any .go file to trigger a rebuild."
	@echo ""
	AIR_FULL_BIN="$(BINARY) start --port $(PORT) --$(MODE)" $(AIR) -c .air.toml

## dev-ui: run Vite dev server (proxies /api to :8080)
dev-ui:
	cd ui && npm run dev

## test: run full Go test suite
test:
	go test ./...

## lint: run linters
lint:
	golangci-lint run

## service-install: install + load launchd auto-start service
service-install:
	cp ~/Library/LaunchAgents/com.mcpjungle.plist ~/Library/LaunchAgents/com.mcpjungle.plist.bak 2>/dev/null || true
	launchctl unload ~/Library/LaunchAgents/com.mcpjungle.plist 2>/dev/null || true
	launchctl load ~/Library/LaunchAgents/com.mcpjungle.plist
	@echo "MCPJungle service installed and running"

## service-stop: stop the launchd service
service-stop:
	launchctl unload ~/Library/LaunchAgents/com.mcpjungle.plist

## service-logs: tail the gateway log
service-logs:
	tail -f /tmp/mcpjungle.log

## help: show available commands
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
