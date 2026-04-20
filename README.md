<h1 align="center">
  MCPJungle
</h1>
<p align="center">
  <strong>Self-hosted MCP gateway. One endpoint for all your tools.</strong>
</p>
<p align="center">
  <a href="https://docs.mcpjungle.com" style="text-decoration: none;">
    <img src="https://img.shields.io/badge/Documentation-docs.mcpjungle.com-blue?style=flat-square&logo=book" alt="Documentation" style="max-width: 100%;">
  </a>

  <a href="https://github.com/mcpjungle/mcpjungle/pkgs/container/mcpjungle" style="text-decoration: none;">
    <img src="https://img.shields.io/badge/GHCR-available-green.svg?style=flat-square&logo=github" alt="GHCR" style="max-width: 100%;">
  </a>

  <a href="https://discord.gg/CapV4Z3krk" style="text-decoration: none;">
    <img src="https://img.shields.io/badge/Discord-MCPJungle-5865F2?style=flat-square&logo=discord&logoColor=white" alt="Discord" style="max-width: 100%;">
  </a>

</p>

MCPJungle aggregates multiple MCP servers into a single MCP endpoint.

It acts as a gateway that your AI clients connect to, providing unified access to tools, prompts, and resources across all MCP servers.

## Why MCPJungle?

Without a gateway, MCP usage does not scale:

- 🔗 Clients must connect to each MCP server separately
- 🧩 Tools and resources are scattered across servers
- 🔐 Access control is duplicated and inconsistent
- 👁️ No centralized visibility into available tools

MCPJungle introduces a single control layer:

- 🎯 One MCP endpoint for all your servers
- 🧰 Unified access to tools, prompts, and resources
- 🛡️ Centralized discovery, access control, and observability

## Quickstart

This quickstart guide will show you how to:
1. Start the mcpjungle server locally using `docker compose`
2. Add an MCP server in mcpjungle
3. Connect your Claude Desktop to mcpjungle to access your MCP tools

### Start the server
Fetch the `docker-compose.yaml` and start the mcpjungle server:
```bash
curl -O https://raw.githubusercontent.com/mcpjungle/MCPJungle/refs/heads/main/docker-compose.yaml
docker compose up -d
```

This exposes mcpjungle's streamable http mcp server at `http://localhost:8080/mcp` by default.

### Add an MCP server
1. Download the `mcpjungle` CLI on your local machine either using brew or directly from the [Releases Page](https://github.com/mcpjungle/MCPJungle/releases).
```bash
brew install mcpjungle/mcpjungle/mcpjungle
```

 2. Add the [context7](https://context7.com/) MCP server to mcpjungle using the CLI:
```bash
mcpjungle register --name context7 --url https://mcp.context7.com/mcp
```

You should see output similar to this:

![register-context7](./assets/register-context7.png)

### Connect to mcpjungle

In your Claude Desktop, add the configuration for mcpjungle MCP server:
```json
{
  "mcpServers": {
    "mcpjungle": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "http://localhost:8080/mcp",
        "--allow-http"
      ]
    }
  }
}
```

Once you have added the configuration, try asking claude something simple: 
```text
Use context7 to get the documentation for `/lodash/lodash`
```

Claude will then attempt to call the `context7__get-library-docs` tool via MCPJungle, which will return the documentation for the Lodash library.

<p align="center">
  <img src="./assets/quickstart-claude-call-tool.png" alt="claude calls context7 tool via mcpjungle" height="400">
</p>

You now have a working MCP setup with a single unified endpoint!

Next, explore the complete documentation at [docs.mcpjungle.com](https://docs.mcpjungle.com/).