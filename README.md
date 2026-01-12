<h1 align="center">
  :deciduous_tree: MCPJungle :deciduous_tree:
</h1>
<p align="center">
  Self-hosted MCP Gateway for your private AI agents
</p>
<p align="center">
  <a href="https://discord.gg/CapV4Z3krk" style="text-decoration: none;">
    <img src="https://img.shields.io/badge/Discord-MCPJungle-5865F2?style=flat-square&logo=discord&logoColor=white" alt="Discord" style="max-width: 100%;">
  </a>
</p>

MCPJungle is a single source-of-truth registry for all [Model Context Protocol](https://modelcontextprotocol.io/introduction) Servers running in your Organisation.

🧑‍💻 Developers use it to register & manage MCP servers and the tools they provide from a central place.

🤖 MCP Clients use it to discover and consume all these tools from a single "Gateway" MCP Server.

![diagram](./assets/mcpjungle-diagram/mcpjungle-diagram.png)

<p align="center">MCPJungle is the only MCP Server your AI agents need to connect to!</p>

# Who should use MCPJungle?
1. **Developers** using MCP Clients like Claude & Cursor that need to access MCP servers for tool-calling
2. **Developers** building production-grade AI Agents that need to access MCP servers with built-in security, privacy and Access Control.
3. **Organisations** wanting to view & manage all MCP client-server interactions from a central place. Hosted in their own datacenter 🔒

# 📋 Table of Contents

- [Quick Start guide](#quickstart-guide)
- [Installation](#installation)
- [Usage](#usage)
  - [Server](#server)
    - [Running mcpjungle server inside Docker](#running-inside-docker)
    - [Running mcpjungle server directly on the host machine](#running-directly-on-host)
    - [Shutting down the server](#shutting-down)
  - [Client](#client)
    - [Adding Streamable HTTP-based MCP servers](#registering-streamable-http-based-servers)
    - [Adding STDIO-based MCP servers](#registering-stdio-based-servers)
    - [Removing MCP servers](#deregistering-mcp-servers)
  - [Cold-start problem & Stateful Connections](#cold-start-problem--stateful-connections)
  - [Connect to mcpjungle from Claude](#claude)
  - [Connect to mcpjungle from Cursor](#cursor)
  - [Enabling/Disabling Tools globally](#enablingdisabling-tools)
  - [Prompts](#prompts)
  - [Tool Groups](#tool-groups)
  - [Authentication](#authentication)
  - [Enterprise features](#enterprise-features-)
    - [Access Control](#access-control)
    - [OpenTelemetry](#opentelemetry)
- [Limitations](#current-limitations-)
- [Contributing](#contributing-)

# Quickstart guide
This quickstart guide will show you how to:
1. Start the MCPJungle server locally using `docker compose`
2. Register a simple MCP server in mcpjungle
3. Connect your Claude to mcpjungle to access your MCP tools

## Start the server
```bash
curl -O https://raw.githubusercontent.com/mcpjungle/MCPJungle/refs/heads/main/docker-compose.yaml
docker compose up -d
```

## Register MCP servers
Download the `mcpjungle` CLI on your local machine either using brew or directly from the [Releases Page](https://github.com/mcpjungle/MCPJungle/releases).
```bash
brew install mcpjungle/mcpjungle/mcpjungle
```

The CLI lets you manage everything in mcpjungle.

Next, lets add an MCP server to mcpjungle using the CLI. For this example, we'll use [context7](https://context7.com/).
```bash
mcpjungle register --name context7 --url https://mcp.context7.com/mcp
```

## Connect to mcpjungle

Use the following configuration for your Claude MCP servers config:
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

Once mcpjungle is added as an MCP to your Claude, try asking it the following:
```text
Use context7 to get the documentation for `/lodash/lodash`
```

Claude will then attempt to call the `context7__get-library-docs` tool via MCPJungle, which will return the documentation for the Lodash library.

<p align="center">
  <img src="./assets/quickstart-claude-call-tool.png" alt="claude calls context7 tool via mcpjungle" height="400">
</p>

Congratulations! 🎉

You have successfully registered a remote MCP server in MCPJungle and called one of its tools via Claude

You can now proceed to play around with the mcpjungle and explore the documentation & CLI for more details.

# Installation
MCPJungle is shipped as a stand-alone binary.

You can either download it from the [Releases](https://github.com/mcpjungle/MCPJungle/releases) Page or use [Homebrew](https://brew.sh/) to install it:

```bash
brew install mcpjungle/mcpjungle/mcpjungle
```

Verify your installation by running

```bash
mcpjungle version
```

> [!IMPORTANT]
> On MacOS, you will have to use homebrew because the compiled binary is not [Notarized](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution) yet.

MCPJungle provides a Docker image which is useful for running the registry server (more about it later).

```bash
docker pull mcpjungle/mcpjungle
```

# Usage
MCPJungle has a Client-Server architecture and the binary lets you run both the Server and the Client.

## Server
The MCPJungle server is responsible for managing all the MCP servers registered in it and providing a unified MCP gateway for AI Agents to discover and call tools provided by these registered servers.

The gateway itself runs over streamable http transport and is accessible at the `/mcp` endpoint.

### Running inside Docker
For running the MCPJungle server locally, docker compose is the recommended way:
```shell
# docker-compose.yaml is optimized for individuals running mcpjungle on their local machines for personal use.
# mcpjungle will run in `development` mode by default.
curl -O https://raw.githubusercontent.com/mcpjungle/MCPJungle/refs/heads/main/docker-compose.yaml

docker compose up -d

# docker-compose.prod.yaml is optimized for orgs deploying mcpjungle on a remote server for multiple users.
# mcpjungle will run in `enterprise` mode by default, which enables enterprise features.
curl -O https://raw.githubusercontent.com/mcpjungle/MCPJungle/refs/heads/main/docker-compose.prod.yaml

docker compose -f docker-compose.prod.yaml up -d
```

> [!NOTE]
> The `enterprise` mode used to be called `production` mode.
> The mode has now been renamed for clarity. Everything else remains the same.

This will start the MCPJungle server along with a persistent Postgres database container.

You can quickly verify that the server is running:
```bash
curl http://localhost:8080/health
```

If you plan on registering stdio-based MCP servers that rely on `npx` or `uvx`, use mcpjungle's `stdio` tagged docker image instead.
```bash
MCPJUNGLE_IMAGE_TAG=latest-stdio docker compose up -d
```

> [!NOTE]
> If you're using `docker-compose.yaml`, this is already the default image tag.
> You only need to specify the stdio image tag if you're using `docker-compose.prod.yaml`.

This image is significantly larger. But it is very convenient and recommended for running locally when you rely on stdio-based MCP servers.

For example, if you only want to register remote mcp servers like context7 and deepwiki, you can use the standard (minimal) image.

But if you also want to use stdio-based servers like `filesystem`, `time`, `github`, etc., you should use the `stdio`-tagged image instead.

> [!NOTE]
> If your stdio servers rely on tools other than `npx` or `uvx`, you will have to create a custom docker image that includes those dependencies along with the mcpjungle binary.

**Production Deployment**

The default [MCPJungle Docker image](https://hub.docker.com/r/mcpjungle/mcpjungle) is very lightweight - it only contains a minimal base image and the `mcpjungle` binary.

It is therefore suitable and recommended for production deployments.

For the database, we recommend you deploy a separate Postgres DB cluster and supply its endpoint to mcpjungle (see [Database](#database) section below).

You can see the definitions of the [standard Docker image](./Dockerfile) and the [stdio Docker image](./stdio.Dockerfile).

### Running directly on host
You can also run the server directly on your host machine using the binary:

```bash
mcpjungle start
```

This starts the main registry server and MCP gateway, accessible on port `8080` by default.

### Shutting down
It is important that the mcpjungle server shuts down gracefully to ensure proper cleanup.

The recommended way to stop the server process is to send a `SIGTERM` signal to it.



### Database
The mcpjungle server relies on a database and by default, creates a SQLite DB file `mcpjungle.db` in the current working directory.

This is okay when you're just testing things out locally.

For more serious deployments, mcpjungle also supports Postgresql. You can supply the DSN to connect to it:

```bash
# You can supply the database DSN as an env var
export DATABASE_URL=postgres://admin:root@localhost:5432/mcpjungle_db

#run as container
docker run mcpjungle/mcpjungle:latest

# or run directly
mcpjungle start
```

You can also supply postgres-specific env vars or files if you don't prefer using the DSN:
```bash
# host is mandatory if you're using postgres-specific env vars
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432

export POSTGRES_USER=admin
export POSTGRES_USER_FILE=/path/to/user-file

export POSTGRES_PASSWORD=secret
export POSTGRES_PASSWORD_FILE=/path/to/password-file

export POSTGRES_DB=mcpjungle_db
export POSTGRES_DB_FILE=/path/to/db-file

mcpjungle start
```

## Client
Once the server is up, you can use the mcpjungle CLI to interact with it.

MCPJungle currently supports MCP servers using [stdio](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#stdio) and [Streamable HTTP](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http) Transports.

> [!NOTE]
> Support for SSE (server-sent events) also exists but is currently not mature.

Let's see how to register them in mcpjungle.

### Registering streamable HTTP-based servers
Let's say you're already running a streamable http MCP server locally at `http://127.0.0.1:8000/mcp` which provides basic math tools like `add`, `subtract`, etc.

You can register this MCP server with MCPJungle:
```bash
mcpjungle register --name calculator --description "Provides some basic math tools" --url http://127.0.0.1:8000/mcp
```

If you used docker compose to run the server, and you're not on Linux, you will have to use `host.docker.internal` instead of your local loopback address.
```bash
mcpjungle register --name calculator --description "Provides some basic math tools" --url http://host.docker.internal:8000/mcp
```

The registry will now start tracking this MCP server and load its tools.

![register a MCP server in MCPJungle](./assets/register-mcp-server.png)

You can also provide a configuration file to register the MCP server:
```bash
cat ./calculator.json
{
  "name": "calculator",
  "transport": "streamable_http",
  "description": "Provides some basic math tools",
  "url": "http://127.0.0.1:8000/mcp"
}

mcpjungle register -c ./calculator.json
```

All tools provided by this server are now accessible via MCPJungle:

```bash
mcpjungle list tools

# Check tool usage
mcpjungle usage calculator__multiply

# Call a tool
mcpjungle invoke calculator__multiply --input '{"a": 100, "b": 50}'
```

![Call a tool via MCPJungle Proxy MCP server](./assets/tool-call.png)

> [!NOTE]
> A tool in MCPJungle must be referred to by its canonical name which follows the pattern `<mcp-server-name>__<tool-name>`.
> Server name and tool name are separated by a double underscore `__`.
>
> eg- If you register a MCP server `github` which provides a tool called `git_commit`, you can invoke it in MCPJungle using the name `github__git_commit`.
> 
> Your MCP client must also use this canonical name to call the tool via MCPJungle.

The config file format for registering a Streamable HTTP-based MCP server is:
```json
{
  "name": "<name of your mcp server>",
  "transport": "streamable_http",
  "description": "<description>",
  "url": "<url of the mcp server>",
  "bearer_token": "<optional bearer token for authentication>"
}
```

### Registering STDIO-based servers

Here's an example configuration file (let's call it `filesystem.json`) for a MCP server that uses the STDIO transport:

```json
{
  "name": "filesystem",
  "transport": "stdio",
  "description": "filesystem mcp server",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "."]
}
```

You can register this MCP server in MCPJungle by providing the configuration file:
```bash
# Save the JSON configuration to a file (e.g., filesystem.json)
mcpjungle register -c ./filesystem.json
```

The config file format for registering a STDIO-based MCP server is:

```json
{
  "name": "<name of your mcp server>",
  "transport": "stdio",
  "description": "<description>",
  "command": "<command to run the mcp server, eg- 'npx', 'uvx'>",
  "args": ["arguments", "to", "pass", "to", "the", "command"],
  "env": {
    "KEY": "value"
  }
}
```

You can also watch a quick video on [How to register a STDIO-based MCP server](https://youtu.be/YqHiuexR5fw).

> [!TIP]
> If your STDIO server fails or throws errors for some reason, check the mcpjungle server's logs to view its `stderr` output.


**Caveat** ⚠️

When running mcpjungle inside Docker, you need some extra configuration to run the `filesystem` mcp server.

By default, mcpjungle inside container does not have access to your host filesystem.

So you must:
- mount the host directory you want to access as a volume in the container
- specify the mount path as the directory in the filesystem mcp server command args

The `docker-compose.yaml` provided by mcpjungle mounts the current working directory as `/host` in the container.

So you can use the following configuration for the filesystem mcp server:

```json
{
  "name": "filesystem",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/host"]
}
```

Then, the mcp has access to `/host`, ie, the current working directory on your host machine.

See [DEVELOPMENT.md](./DEVELOPMENT.md#docker-filesystem-access) for more details.


### Deregistering MCP servers
You can remove a MCP server from mcpjungle.

```bash
mcpjungle deregister calculator
mcpjungle deregister filesystem
```

Once removed, this mcp server and its tools are no longer available to you or your MCP clients.

## Cold-start problem & Stateful Connections
By default, MCPJungle always creates a new connection with the upstream MCP server when a tool is called.

When the tool call is complete, the connection is closed.

This keeps the system clean and avoids memory leaks.

But sometimes this can cause a latency overhead. For eg- a new process is spawned every time you call a tool of a STDIO-based mcp server. If the server takes several seconds to start up, this slows down the tool call and the overall interaction.

To solve this, MCPJungle also supports stateful connections.

You can set the `session_mode` to `stateful` (default is `stateless`) in you MCP server configuration:
```json
{
  "name": "filesystem",
  "transport": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "."],
  "session_mode": "stateful"
}
```

mcpjungle will create a new connection with this mcp server **the first time you call one of its tools**.

This connection is not closed when the tool call is complete. Subsequent tool calls to this server reuse the same connection, avoiding the cold-start overhead.

The connection is only closed when:
1. mcpjungle server is stopped
2. the mcp server is deregistered from mcpjungle
3. the connection times out after a period of inactivity. You can set the number of seconds using the `SESSION_IDLE_TIMEOUT_SEC` env var to configure this globally in mcpjungle server (default value is -1, which means no timeout).

When possible, it is recommended that you use stateless connections (default setting).

## Integration with other MCP Clients
Assuming that MCPJungle is running on `http://localhost:8080`, use the following configurations to connect to it:

### Claude
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

### Cursor
```json
{
  "mcpServers": {
    "mcpjungle": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

You can watch a quick video on [How to connect Cursor to MCPJungle](https://youtu.be/SaUqj-eLPnw).

## Enabling/Disabling Tools
You can disable and re-enable a specific tool or all the tools provided by an MCP Server.

If a tool is disabled, it is not available via the MCPJungle Proxy or any of the Tool Groups, so no MCP clients can view or call it.

You can disable and enable Prompts as well.

```bash
# disable the `get-library-docs` tool provided by the `context7` MCP server
mcpjungle disable tool context7__get-library-docs

# re-enable the tool
mcpjungle enable tool context7__get-library-docs

# disable all tools in context7
mcpjungle disable tool context7

# disable the whole `context7` MCP server (disables all tools & prompts)
mcpjungle disable server context7

# re-enable `context7`
mcpjungle enable server context7

# disable a prompt
mcpjungle disable prompt "huggingface_Model Details"

# disable all prompts in context7
mcpjungle disable prompt context7
```

A disabled tool is still accessible via mcpjungle's HTTP API, so humans can still manage it from the CLI (or any other HTTP client).

> [!NOTE]
> When a new server is registered in MCPJungle, all its tools & prompts are **enabled** by default.

## Prompts
Mcpjungle supports [Prompts](https://modelcontextprotocol.io/specification/2025-06-18/server/prompts).

When you register a new MCP server, if it provides prompts, they're registered in mcpjungle too.

Here are some examples of how you can interact with Prompts using the CLI:
```bash
# list all prompts provided by the huggingface mcp
$ mcpjungle list prompts --server huggingface

# Retrieve the "Model Details" prompt, supply custom arguments
$ mcpjungle get prompt "huggingface__Model Details" --arg model_id="openai/gpt-oss-120b"
```

## Tool Groups
As you add more MCP servers to MCPJungle, the number of tools available through the Gateway can grow significantly.

If your MCP client is exposed to hundreds of tools through the gateway MCP, its performance may degrade.

MCPJungle allows you to **expose only a subset of all available tools to your MCP clients using Tool Groups**.

You can create a new group and only include specific tools that you wish to expose.

Once a group is created, mcpjungle returns a unique endpoint for it.

You can then configure your MCP client to use this group-specific endpoint instead of the main gateway endpoint.

### Creating a Tool Group
You can create a new tool group by providing a JSON configuration file to the `create group` command.

You must specify a unique `name` for the group and define which tools to include using one or more of the following fields:
- **`included_tools`**: List specific tool names to include (e.g., `["filesystem__read_file", "time__get_current_time"]`)
- **`included_servers`**: Include ALL tools from specific MCP servers (e.g., `["time", "deepwiki"]`)
- **`excluded_tools`**: Exclude specific tools (useful when including entire servers)

#### Example 1: Cherry-picking specific tools
Here is an example of a tool group configuration file (`claude-tools-group.json`):
```json
{
  "name": "claude-tools",
  "description": "This group only contains tools for Claude Desktop to use",
  "included_tools": [
    "filesystem__read_file",
    "deepwiki__read_wiki_contents",
    "time__get_current_time"
  ]
}
```

This group exposes only 3 handpicked tools instead of all available tools.

#### Example 2: Including entire servers with exclusions
You can also include all tools from specific servers and optionally exclude some:
```json
{
  "name": "claude-tools",
  "description": "All tools from time and deepwiki servers except time__convert_time",
  "included_servers": ["time", "deepwiki"],
  "excluded_tools": ["time__convert_time"]
}
```

This includes ALL tools from the `time` and `deepwiki` servers except `time__convert_time`.

#### Example 3: Mixing approaches
You can combine all three fields for maximum flexibility:
```json
{
  "name": "comprehensive-tools",
  "description": "Mix of manual tools, server inclusion, and exclusions",
  "included_tools": ["filesystem__read_file"],
  "included_servers": ["time"],
  "excluded_tools": ["time__convert_time"]
}
```

This includes `filesystem__read_file` plus all tools from the `time` server except `time__convert_time`.

You can create this group in mcpjungle:
```bash
$ mcpjungle create group -c ./claude-tools-group.json

Tool Group claude-tools created successfully
It is now accessible at the following streamable http endpoint:

    http://127.0.0.1:8080/v0/groups/claude-tools/mcp

```

You can then configure Claude (or any other MCP client) to use this group-specific endpoint to access the MCP server.

The client will then ONLY see and be able to use these 3 tools and will not be aware of any other tools registered in MCPJungle.

> [!TIP]
> You can run `mcpjungle list tools` to view all available tools and pick the ones you want to include in your group.

You can also watch a [Video on using Tool Groups](https://youtu.be/A21rfGgo38A).

> [!NOTE]
> The exclusion is always applied at the end.
> So if you add a tool to `included_tools` and also list it in `excluded_tools`, it will be excluded from the final group.

#### Limitation 🚧
[Prompts](https://modelcontextprotocol.io/specification/2025-06-18/server/prompts) are currently not supported in Tool Groups. We're working to fix this [issue](https://github.com/mcpjungle/MCPJungle/issues/136) 🛠️

### Managing tool groups
You can currently perform operations like listing all groups, viewing details of a specific group and deleting a group.

```bash
# list all tool groups
mcpjungle list groups

# view details of a specific group
mcpjungle get group claude-tools

# delete a group
mcpjungle delete group claude-tools
```

### Working with tools in groups
You can list and invoke tools within specific groups using the `--group` flag:

```bash
# list tools in a specific group
mcpjungle list tools --group claude-tools

# invoke a tool from a specific group context
mcpjungle invoke filesystem__read_file --group claude-tools --input '{"path": "README.md"}'
```

These commands provide group-scoped operations, making it easier to work with tools within specific contexts and validate that tools are available in your groups.

> [!NOTE]
> If a tool is included in a group but is later disabled globally or deleted, then it will not be available via the group's MCP endpoint.
>
> But if the tool is re-enabled or added again later, it will automatically become available in the group again.

**Limitations** 🚧
1. Currently, you cannot update an existing tool group. You must delete the group and create a new one with the modified configuration file.
2. In `enterprise` mode, currently only an admin can create a Tool Group. We're working on allowing standard Users to create their own groups as well.

## Authentication
MCPJungle currently supports authentication if your Streamable HTTP MCP Server accepts static tokens for auth.

This is useful when using SaaS-provided MCP Servers like HuggingFace, Stripe, etc. which require your API token for authentication.

You can supply your token while registering the MCP server:
```bash
# If you specify the `--bearer-token` flag, MCPJungle will add the `Authorization: Bearer <token>` header to all requests made to this MCP server.
mcpjungle register --name huggingface --description "HuggingFace MCP Server" --url https://huggingface.co/mcp --bearer-token <your-hf-api-token>
```

Or from your configuration file
```bash
{
  "name": "huggingface",
  "transport": "streamable_http",
  "url": "https://huggingface.co/mcp",
  "description": "hugging face mcp server",
  "bearer_token": "<your-hf-api-token>"
}
```

Support for Oauth flow is coming soon!

## Enterprise Features 🔒

If you're running MCPJungle in your organisation, we recommend running the Server in the `enterprise` mode:
```bash
# enable enterprise features by running in enterprise mode
mcpjungle start --enterprise

# you can also specify the server mode as environment variable (valid values are `development` and `enterprise`)
export SERVER_MODE=enterprise
mcpjungle start

# Or use the enterprise-mode docker compose file as described above
docker compose -f docker-compose.prod.yaml up -d
```

By default, mcpjungle server runs in `development` mode which is ideal for individuals running it locally.

In Enterprise mode, the server enforces stricter security policies and will provide additional features like Authentication, ACLs, observability and more.

After starting the server in enterprise mode, you must initialize it by running the following command on your client machine:
```bash
mcpjungle init-server
```

This will create an admin user in the server and store its API access token in your home directory (`~/.mcpjungle.conf`).

You can then use the mcpjungle cli to make authenticated requests to the server.

### Access Control

In `development` mode, all MCP clients have full access to all the MCP servers registered in MCPJungle Proxy.

`enterprise` mode lets you control which MCP clients can access which MCP servers.

Suppose you have registered 2 MCP servers `calculator` and `github` in MCPJungle in enterprise mode.

By default, no MCP client can access these servers. **You must create an MCP Client in mcpjungle and explicitly allow it to access the MCP servers.**

```bash
# Create a new MCP client for your Cursor IDE to use. It can access the calculator and github MCP servers
mcpjungle create mcp-client cursor-local --allow "calculator, github"

MCP client 'cursor-local' created successfully!
Servers accessible: calculator,github

Access token: 1YHf2LwE1LXtp5lW_vM-gmdYHlPHdqwnILitBhXE4Aw
Send this token in the `Authorization: Bearer {token}` HTTP header.
```

Mcpjungle creates an access token for your client.
Configure your client or agent to send this token in the `Authorization` header when making requests to the mcpjungle proxy.

> [!TIP]
> You can also supply a custom access token for your mcp clients and user accounts using the `--access-token` flag.
> This is useful when you want to manage tokens yourself, perhaps through a central identity server.

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

### OpenTelemetry
MCPJungle supports Prometheus-compatible OpenTelemetry Metrics for observability.

- In `enterprise` mode, OpenTelemetry is enabled by default.
- In `development` mode, telemetry is disabled by default. You can enable it by setting the `OTEL_ENABLED` environment variable to `true` before starting the server:

```bash
# enable OpenTelemetry metrics
export OTEL_ENABLED=true

# optionally, set additional attributes to be added to all metrics
export OTEL_RESOURCE_ATTRIBUTES=deployment.environment.name=enterprise

# start the server
mcpjungle start
```

Once the mcpjungle server is started, metrics are available at the `/metrics` endpoint.

# Current limitations 🚧
We're not perfect yet, but we're working hard to get there!

### 1. MCPJungle does not support OAuth flow for authentication yet
This is a work in progress.

We're collecting more feedback on how people use OAuth with MCP servers, so feel free to start a Discussion or open an issue to share your use case.

# Contributing 💻

We welcome contributions from the community! 

- **For contribution guidelines and standards**, see [CONTRIBUTION.md](./CONTRIBUTION.md)
- **For development setup and technical details**, see [DEVELOPMENT.md](./DEVELOPMENT.md)

Join our [Discord community](https://discord.gg/CapV4Z3krk) to connect with other contributors and maintainers.
