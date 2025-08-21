## Contributing 💻

This document contains notes for Developers and Contributors of MCPJungle.

If you're simply a user of MCPJungle, you can skip this doc.

We're thrilled to have you here 🚀

### Basic Guidelines
MCPJungle always follows **minimalism**.

Whenever we want to implement some functionality or fix an issue, we figure out the least amount of changes required to ship something that works well and doesn't break anything else.

Any other extras or improvements on top are always guided by **user feedback** and never by an engineer's desire to work on something.

This keeps mcpjungle lean, protects it from over-engineering and makes us the TOP choice for our users 🚀

### Getting started

If you'd like to contribute to the MCPJungle Codebase/Documentation, the following approach is suggested:

1. Learn the basic usage of mcpjungle. Go through the README & play around with the tool. Launch and use MCP servers.
2. Skim through the codebase to get a general idea - [CLI](https://github.com/mcpjungle/MCPJungle/tree/main/cmd), [HTTP API](https://github.com/mcpjungle/MCPJungle/blob/main/internal/api/server.go), [MCP Proxy server](https://github.com/mcpjungle/MCPJungle/blob/main/internal/service/mcp/proxy.go) (or ask Copilot to explain the architecture to you ;) )
3. Pick up an [issue](https://github.com/mcpjungle/MCPJungle/issues) that you like. Some of them are marked as `good first issue` but in general, pick whatever you like. Feel free to discuss it with the maintainers either on the issue thread or in [Discord](https://discord.gg/TSrUCTw9).
4. Feel free to open up Discussions if you want to propose something new.
5. When in doubt, just shoot a message in the Discord general chat and the maintainers will be there to help you out!

### Build for local testing
```bash
# Single binary for your current system
$ goreleaser build --single-target --clean --snapshot

# Test the full release assets (binaries, docker image) without publishing
goreleaser release --clean --snapshot --skip publish

# Binaries for all supported platforms
$ goreleaser release --snapshot --clean
```

### Dependency management policy
- We rely on `go.mod` + `go.sum` as the single source of truth for dependencies.
- The `vendor/` directory is not committed to the repository to reduce repo size and PR noise.
- For fully offline or air-gapped builds, regenerate vendors locally with `go mod vendor` if needed, but do not commit the changes.
- CI/Docker builds must use module-aware mode and fetch dependencies via the Go proxy. If necessary, set `GOPROXY="https://proxy.golang.org,direct"`.

### Swagger UI / OpenAPI
- Swagger UI is hosted by the dev server at `http://localhost:8080/docs/index.html`.
- The raw OpenAPI document is served at `http://localhost:8080/openapi.json`.

### Create a new release
1. Create a Git Tag with the new version

```bash
git tag -a 0.1.0 -m "Release version 0.1.0"
git push origin 0.1.0
```

2. Release
```bash
# Make sure GPG is present on your system and you have a default key which is added to Github.

# set your github access token
export GITHUB_TOKEN="<your GH token>"

goreleaser release --clean
```

This will create a new release under Releases and also make it available via Homebrew.


### Use SQLite
When running MCPJungle with SQLite, you can access the database using the `sqlite3` command line tool.

```bash
sqlite3 mcp.db

> .tables
> SELECT * FROM mcp_servers;
> SELECT * FROM tools;

# and so on...
```

### Use PostgreSQL
When running MCPJungle with docker-compose, you can access the PostgreSQL database using the `pgadmin` utility.

Open `http://localhost:5050` in your browser and log in with the Username `admin@admin.com` and Password `admin`.

Add a new DB Server with the following settings:
- Host: `db`
- Port: `5432`
- Username: `mcpjungle`
- Password: `mcpjungle`
- Database: `mcpjungle`

Then you can open up tables and run queries.

### Use MCP Inspector
MCP Inspector is a GUI MCP client to test out all interactions with MCP servers.

Very useful for debugging and testing mcpjungle.

```bash
npx @modelcontextprotocol/inspector
```
