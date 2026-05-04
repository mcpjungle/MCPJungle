# AGENTS.md

Instructions for coding agents working on MCPJungle.

This file is the agent-facing companion to [CONTRIBUTION.md](./CONTRIBUTION.md) and [DEVELOPMENT.md](./DEVELOPMENT.md). Read both before making non-trivial changes.

## Project overview

MCPJungle is a single-binary Go server that proxies and manages multiple Model Context Protocol (MCP) servers. Components:

- `cmd/`: CLI commands (built with `cobra`)
- `internal/api/`: HTTP API
- `internal/service/mcp/`: MCP proxy server and protocol handling
- `internal/db/`: persistence layer (SQLite by default, PostgreSQL optional)
- `client/`: Go client library
- `pkg/`: public packages

Module path: `github.com/mcpjungle/mcpjungle`. Go version: see `go.mod` (currently 1.24+).

## Setup

```bash
git clone https://github.com/mcpjungle/MCPJungle.git
cd MCPJungle
go mod download
```

The `vendor/` directory is intentionally not committed. Use module-aware mode (`GOPROXY="https://proxy.golang.org,direct"` if needed).

## Build

```bash
# Single binary for the current platform
goreleaser build --single-target --clean --snapshot

# All platforms (release dry-run, no publish)
goreleaser release --clean --snapshot --skip publish
```

If `goreleaser` is not installed, `go build ./...` is enough for type checking.

## Test

```bash
# All unit tests
go test ./...

# With coverage
go test -cover ./...

# A single package
go test ./internal/service/mcp

# End-to-end sanity script (requires a running server)
./scripts/test-mcpjungle.sh
```

Add tests for new functionality in the same package as the code you change.

## Lint

```bash
golangci-lint run

# Auto-fix where possible
golangci-lint run --fix
```

Lint config lives in `.golangci.yml`. Enabled linters include `govet`, `staticcheck`, `unused`, `revive`, and `bodyclose`. Run lint before committing. CI rejects lint failures.

## Format

```bash
gofmt -w .
go vet ./...
```

`gofmt` is not optional. CI checks formatting.

## Run locally

```bash
# Build a snapshot binary
goreleaser build --single-target --clean --snapshot

# Or with docker-compose (PostgreSQL + pgadmin)
docker-compose up -d
```

For SQLite development, the local database file is `mcpjungle.db` (legacy installs may still use `mcp.db`).

## Code style

- Follow [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- Keep changes minimal. From `DEVELOPMENT.md`: "Whenever we want to implement some functionality or fix an issue, we figure out the least amount of changes required to ship something that works well and doesn't break anything else."
- Prefer small, focused PRs over sweeping refactors.
- Add comments only where the why is non-obvious.
- Update relevant docs (`README.md`, `DEVELOPMENT.md`, examples) when changing user-facing behavior.

## Commit and PR conventions

- Conventional commit prefixes are common in the history (`feat:`, `fix:`, `docs:`, `chore:`), with optional scope (`fix(mcp): ...`, `feat(e2e-tests): ...`). Recent merges show both scoped and unscoped forms; either is acceptable.
- Keep the subject line short and imperative. The PR description carries the detail.
- Rebase your branch onto the latest `main` before opening a PR.
- One logical change per PR. Reviewers will ask for splits if a PR mixes unrelated work.

## Before opening a PR

- Run `go test ./...`, `golangci-lint run`, and `gofmt -w .`.
- For significant or architectural changes, open a Discussion first to align with maintainers.
- For AI-generated code: run it, run the tests, and review the diff yourself before pushing. Maintainers explicitly call this out in `CONTRIBUTION.md`.

## Things to avoid

- Committing the `vendor/` directory.
- Adding dependencies without justification. The project values minimalism.
- Disabling lint checks instead of fixing the underlying issue.
- Large speculative refactors. Match the change to the issue.

## Useful pointers

- HTTP API entry point: `internal/api/server.go`
- MCP proxy: `internal/service/mcp/proxy.go`
- CLI command tree: `cmd/`
- DB models: `internal/db/`
- Sample MCP config: see `README.md` for the full registration flow

For protocol-level debugging, the MCP Inspector GUI is helpful:

```bash
npx @modelcontextprotocol/inspector
```

## Where to ask

- Existing issues: <https://github.com/mcpjungle/MCPJungle/issues>
- Discussions for proposals and questions
- Discord for chat: linked from `CONTRIBUTION.md`
