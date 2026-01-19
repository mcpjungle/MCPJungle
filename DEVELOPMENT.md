# Development Guide

This document contains detailed development information for MCPJungle contributors and maintainers.

## Basic Guidelines

MCPJungle always follows **minimalism**.

Whenever we want to implement some functionality or fix an issue, we figure out the least amount of changes required to ship something that works well and doesn't break anything else.

Any other extras or improvements on top are always guided by **user feedback** and never by an engineer's desire to work on something.

This keeps mcpjungle lean, protects it from over-engineering and makes us the TOP choice for our users 🚀

## Architecture Overview

MCPJungle consists of several key components:

- **CLI**: Command-line interface for managing MCP servers and tools (`cmd/`)
- **HTTP API**: RESTful API for server management (`internal/api/`)
- **MCP Proxy Server**: Handles MCP protocol communication (`internal/service/mcp/`)
- **Database Layer**: SQLite/PostgreSQL support for persistence (`internal/db/`)

## Development Workflow

### 1. Understanding the Codebase

Start by exploring these key areas:
- [CLI Commands](https://github.com/mcpjungle/MCPJungle/tree/main/cmd)
- [HTTP API Server](https://github.com/mcpjungle/MCPJungle/blob/main/internal/api/server.go)
- [MCP Proxy Server](https://github.com/mcpjungle/MCPJungle/blob/main/internal/service/mcp/proxy.go)

### 2. Building and Testing

#### Local Development Build
```bash
# Single binary for your current system
$ goreleaser build --single-target --clean --snapshot

# Test the full release assets (binaries, docker image) without publishing
goreleaser release --clean --snapshot --skip publish

# Binaries for all supported platforms
$ goreleaser release --snapshot --clean
```

#### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test ./internal/service/mcp

# Run Sanity check script
./scripts/test-mcpjungle.sh
```

#### Linting
We use `golangci-lint` for linting. See its [Installation Docs](https://golangci-lint.run/docs/welcome/install/).

```bash
# Run golangci-lint
golangci-lint run

# Fix issues automatically where possible
golangci-lint run --fix
```

### 3. Database Development

#### SQLite Development
When running MCPJungle with SQLite, you can access the database using the `sqlite3` command line tool:

```bash
sqlite3 mcpjungle.db

> .tables
> SELECT * FROM mcp_servers;
> SELECT * FROM tools;
# and so on...
```

**Note:** For backward compatibility, MCPJungle will still use existing `mcp.db` files if they exist, but will create new databases as `mcpjungle.db`.

#### PostgreSQL Development
When running MCPJungle with docker-compose, you can access the PostgreSQL database using the `pgadmin` utility:

1. Open `http://localhost:5050` in your browser
2. Log in with Username `admin@admin.com` and Password `admin`
3. Add a new DB Server with these settings:
   - Host: `db`
   - Port: `5432`
   - Username: `mcpjungle`
   - Password: `mcpjungle`
   - Database: `mcpjungle`

Then you can open up tables and run queries.

### 4. MCP Server Testing

MCP Inspector is a GUI MCP client to test out all interactions with MCP servers. Very useful for debugging and testing mcpjungle:

```bash
npx @modelcontextprotocol/inspector
```

## Dependency Management Policy

- We rely on `go.mod` + `go.sum` as the single source of truth for dependencies
- The `vendor/` directory is not committed to the repository to reduce repo size and PR noise
- For fully offline or air-gapped builds, regenerate vendors locally with `go mod vendor` if needed, but do not commit the changes
- CI/Docker builds must use module-aware mode and fetch dependencies via the Go proxy. If necessary, set `GOPROXY="https://proxy.golang.org,direct"`

## Release Process

### Creating a New Release

1. **Create a Git Tag** with the new version:
   ```bash
   git tag -a 0.1.0 -m "Release version 0.1.0"
   git push origin 0.1.0
   ```

2. **Release** using goreleaser:
   ```bash
   # Make sure GPG is present on your system and you have a default key which is added to Github
   
   # set your github access token
   export GITHUB_TOKEN="<your GH token>"

   # [New] Login to GHCR (required for pushing docker images)
   echo $GITHUB_TOKEN | docker login ghcr.io -u <your-github-username> --password-stdin
   
   goreleaser release --clean
   ```

This will create a new release under Releases and also make it available via Homebrew.

## Development Environment Setup

### Prerequisites
- Go 1.21 or later
- Docker and Docker Compose
- SQLite3 (for local development)
- Node.js (optional, for MCP Inspector)

### Quick Start
```bash
# Clone and setup
git clone https://github.com/mcpjungle/MCPJungle.git
cd MCPJungle

# Start development environment
docker-compose up -d

# Build and test
goreleaser build --single-target --clean --snapshot
```

## Docker Filesystem Access

When running MCPJungle in Docker, the filesystem MCP server needs special configuration to access the host machine's filesystem. The Docker Compose files have been updated to mount the host filesystem at `/host` inside the container.

### Key Points for Docker Filesystem Access

1. **Volume Mount**: The host filesystem is mounted as read-only at `/host` inside the container. It gives access to the current working directory on the host.
2. **Filesystem MCP Configuration**: Use `/host` as the root path when configuring the filesystem MCP server
3. **Security**: The mount is read-only by default for security. Modify the volume mount in `docker-compose.yaml` if you need write access

### Example Filesystem MCP Configuration for Docker

```json
{
  "name": "filesystem",
  "transport": "stdio",
  "description": "filesystem mcp server",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/host"]
}
```

> **Tip**: Copy the JSON configuration above and save it to a file (e.g., `filesystem.json`) to use with the `mcpjungle register -c` command.

### Alternative: Mount Specific Directories for Better Security

Instead of mounting the entire filesystem, you can mount specific directories by modifying the volume mounts in `docker-compose.yaml`:

```yaml
volumes:
  # Mount only your home directory
  - ${HOME}:/host/home:ro
  # Mount a specific project directory
  - /path/to/your/project:/host/project:ro
  # Mount temp directory with write access
  - /tmp:/host/tmp:rw
```

Then configure your filesystem MCP server accordingly:
```json
{
  "name": "filesystem",
  "transport": "stdio", 
  "description": "filesystem mcp server",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/host/home"]
}
```

## Troubleshooting

### Common Issues

1. **Build failures**: Ensure you have the correct Go version and dependencies
2. **Database connection issues**: Check if PostgreSQL/SQLite is running and accessible
3. **MCP server registration failures**: Verify the MCP server is running and accessible
4. **Docker filesystem access issues**: Ensure the host filesystem is properly mounted at `/host`

### Getting Help

- Check existing [issues](https://github.com/mcpjungle/MCPJungle/issues)
- Join our [Discord community](https://discord.gg/CapV4Z3krk)
- Open a Discussion for questions and proposals

## Contributing Guidelines

For detailed contribution guidelines, see [CONTRIBUTION.md](./CONTRIBUTION.md).

---

Happy coding! 🚀
