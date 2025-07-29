<h1 align="center">
  :deciduous_tree: MCPJungle :deciduous_tree:
</h1>
<p align="center">
  Self-hosted MCP Server registry for your private AI agents
</p>
<p align="center">
  <a href="https://discord.gg/TSrUCTw9" style="text-decoration: none;">
    <img src="https://img.shields.io/badge/Discord-MCPJungle-5865F2?style=flat-square&logo=discord&logoColor=white" alt="Discord" style="max-width: 100%;">
  </a>
</p>

MCPJungle is a single source-of-truth registry for all [Model Context Protocol](https://modelcontextprotocol.io/introduction) based Servers running in your Organisation.

🧑‍💻 Developers use it to register & manage MCP servers and the tools they provide from a central place.

🤖 AI Agents use it to discover and consume all these tools from a single MCP Server.

![diagram](./assets/mcpjungle-diagram/mcpjungle-diagram.png)

<p align="center">MCPJungle is the only MCP Server your agents need to connect to!</p>

## Who should use MCPJungle?
1. Individuals using MCP Clients like Claude & Cursor that need to access multiple MCP servers for tools.
2. Devs building AI Agents that need to access MCP servers with built-in **security** and **Access Control**.
3. Organisations wanting to track all their MCP servers from one centralized place. Hosted in their own datacenter 🔒

## Installation

> [!WARNING]
> MCPJungle is **BETA** software.
>
> We're actively working to make it production-ready.
> You can provide your feedback by [starting a discussion](https://github.com/mcpjungle/MCPJungle/discussions) in this repository.

MPCJungle is shipped as a stand-alone binary.

You can either download it from the [Releases](https://github.com/mcpjungle/MCPJungle/releases) Page or use [Homebrew](https://brew.sh/) to install it:

```bash
$ brew install mcpjungle/mcpjungle/mcpjungle
```

Verify your installation by running

```bash
$ mcpjungle version
```

> [!IMPORTANT]
> On MacOS, you will have to use homebrew because the compiled binary is not [Notarized](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution) yet.


## Usage

MCPJungle has a Client-Server architecture and the binary lets you run both a Server and a Client.

### Server
For running the MCPJungle server locally, docker compose is the recommended way:
```shell
curl -O https://raw.githubusercontent.com/mcpjungle/MCPJungle/refs/heads/main/docker-compose.yaml
docker-compose up -d
```

Otherwise, you can run the server directly using the binary:
```bash
$ mcpjungle start
```

This starts the main registry server responsible for managing all MCP servers. It is accessible on port `8080` by default.

The server also exposes its own MCP server at `/mcp` for AI Agents to discover and call Tools provided by the registered MCP Servers.

It relies on a database and by default, creates a SQLite DB in the current working directory.
Alternatively, you can supply a DSN for a Postgresql database to the server:

```bash
$ export DATABASE_URL=postgres://admin:root@localhost:5432/mcpjungle_db
$ mcpjungle start
```

If you use docker-compose, the DB is automatically created and managed for you.

### Client
Once the server is up, you can use the CLI to interact with it.

Let's say you're already running a MCP server locally at `http://127.0.0.1:8000/mcp` which provides basic math tools like `add`, `subtract`, etc.

You can register this MCP server with MCPJungle:
```bash
$ mcpjungle register --name calculator --description "Provides some basic math tools" --url http://127.0.0.1:8000/mcp
```

If you used docker-compose to run the server, and you're not on Linux, you will have to use `host.docker.internal` instead of your local loopback address.
```bash
$ mcpjungle register --name calculator --description "Provides some basic math tools" --url http://host.docker.internal:8000/mcp
```

The registry will now start tracking this MCP server and load its tools.

![register a MCP server in MCPJungle](./assets/register-mcp-server.png)

**Note**: MCPJungle currently only supports MCP Servers using the [Streamable HTTP Transport](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http).

All tools provided by this server are now accessible via MCPJungle:

```bash
$ mcpjungle list tools

# Check tool usage
$ mcpjungle usage calculator__multiply

# Call a tool
$ mcpjungle invoke calculator__multiply --input '{"a": 100, "b": 50}'

```

![Call a tool via MCPJungle Proxy MCP server](./assets/tool-call.png)

> [!NOTE]
> A tool in MCPJungle must be referred to by its canonical name which follows the pattern `<mcp-server-name>__<tool-name>`.
> Server name and tool name are separated by a double underscore `__`.
>
> eg- If you register a MCP server `github` which provides a tool called `git_commit`, you can invoke it in MCPJungle using the name `github__git_commit`.
> 
> Your AI Agent must also use this canonical name to call the tool via MCPJungle.


Finally, you can remove a MCP server from the registry:
```bash
$ mcpjungle deregister calculator
```

After running this, the registry will stop tracking this server and its tools will no longer be available to use.

### Integration with other MCP Clients
Assuming that MCPJungle is running on `http://localhost:8080`, use the following configurations to connect to it:

**Cursor**
```json
{
  "mcpServers": {
    "mcpjungle": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

**Claude**
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

### Enabling/Disabling Tools
You can enable or disable a specific tool or all the tools provided by an MCP Server.

If a tool is disabled, it is not available via the MCPJungle Proxy, so no MCP clients can view or call it.

```bash
# disable the `get-library-docs` tool provided by the `context7` MCP server
mcpjungle disable context7__get-library-docs

# re-enable the tool
mcpjungle enable context7__get-library-docs

# disable all tools provided by the `context7` MCP server
mcpjungle disable context7

# re-enable all tools of `context7`
mcpjungle enable context7
```

A disabled tool is still accessible via mcpjungle's HTTP API, so you can still manage it from the CLI (or any other HTTP client).

> [!NOTE]
> When a new server is registered in MCPJungle, all its tools are **enabled** by default.

### Authentication
MCPJungle currently supports authentication if your MCP Server accepts static tokens for auth.

This is useful when using SaaS-provided MCP Servers like HuggingFace, Stripe, etc. which require your API token for authentication.

You can supply your token while registering the MCP server:
```bash
# If you specify the `--bearer-token` flag, MCPJungle will add the `Authorization: Bearer <token>` header to all requests made to this MCP server.
$ mcpjungle register --name huggingface --description "HuggingFace MCP Server" --url https://huggingface.co/mcp --bearer-token <your-hf-api-token>
```

Support for Oauth flow is coming soon!

### Enterprise Features 🔒

If you're running MCPJungle in your organisation, we recommend running the Server in the `production` mode:
```bash
# enable enterprise features by running in production mode
$ mcpjungle start --prod

# you can also specify the server mode as environment variable (valid values are `development` and `production`)
$ export SERVER_MODE=production
$ mcpjungle start

# this also works when running the server via docker-compose
$ SERVER_MODE=production docker-compose up
```

By default, mcpjungle server runs in `development` mode which is ideal for individuals running it locally.

In Production mode, the server enforces stricter security policies and will provide additional features like Authentication, ACLs, observability and more.

After starting the server in production mode, you must initialize it by running the following command on your client machine:
```bash
$ mcpjungle init-server
```

This will create an admin user in the server and store its API access token in your home directory (`~/.mcpjungle.conf`).

You can then use the mcpjungle cli to make authenticated requests to the server.

#### Access Control

In `development` mode, all MCP clients have full access to all the MCP servers registered in MCPJungle Proxy.

`production` mode lets you control which MCP clients can access which MCP servers.

Suppose you have registered 2 MCP servers `calculator` and `github` in MCPJungle in production mode.

By default, no MCP client can access these servers. **You must create an MCP Client in mcpjungle and explicitly allow it to access the MCP servers.**

```bash
# Create a new MCP client for your Cursor IDE to use. It can access the calculator and github MCP servers
$ mcpjungle create mcp-client cursor-local --allow "calculator, github"

MCP client 'cursor-local' created successfully!
Servers accessible: calculator,github

Access token: 1YHf2LwE1LXtp5lW_vM-gmdYHlPHdqwnILitBhXE4Aw
Send this token in the `Authorization: Bearer {token}` HTTP header.
```

Mcpjungle creates an access token for your client.
Configure your client or agent to send this token in the `Authorization` header when making requests to the mcpjungle proxy.

For example, you can add the following configuration in Cursor to connect to MCPJungle:

```json
{
  "mcpServers": {
    "mcpjungle": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer 1YHf2LwE1LXtp5lW_vM-gmdYHlPHdqwnILitBhXE4Aw"
      }
    }
  }
}
```

A client that has access to a particular server this way can view and call all the tools provided by that server.

> [!NOTE]
> If you don't specify the `--allow` flag, the MCP client will not be able to access any MCP servers.

## Contributing 💻

If you're interested in contributing to MCPJungle, see [Developer Docs](./docs/developer.md).
