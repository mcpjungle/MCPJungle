import type { Plugin } from "vite";

interface MockRequest {
  method: string;
  url: string;
}

const mockOverview = {
  status: "running",
  mode: "enterprise",
  version: "1.4.2",
  endpoints: [
    { label: "MCP", url: "http://localhost:8080/mcp" },
    { label: "Admin", url: "http://localhost:8080/admin" },
  ],
  server_count: 6,
  tool_count: 23,
  prompt_count: 8,
  resource_count: 12,
};

const mockServers = {
  servers: [
    {
      name: "filesystem",
      transport: "stdio",
      enabled: true,
      status: "connected",
      tool_count: 5,
      prompt_count: 0,
      resource_count: 3,
      last_discovered_at: "2026-09-18T10:30:00Z",
      updated_at: "2026-09-18T10:30:00Z",
      connection_summary: "stdio: npx -y @modelcontextprotocol/server-filesystem /Users/rafaelferreira/Documents",
      config_summary: {
        kind: "stdio",
        command: "npx",
        argument_count: 3,
        env_keys: [],
        description: "Filesystem access with read/write permissions",
        sanitized_summary: "stdio command with 3 arguments",
      },
    },
    {
      name: "github",
      transport: "stdio",
      enabled: true,
      status: "connected",
      tool_count: 12,
      prompt_count: 3,
      resource_count: 2,
      last_discovered_at: "2026-09-18T10:30:00Z",
      updated_at: "2026-09-18T10:30:00Z",
      connection_summary: "stdio: npx -y @modelcontextprotocol/server-github",
      config_summary: {
        kind: "stdio",
        command: "npx",
        argument_count: 2,
        env_keys: ["GITHUB_TOKEN"],
        description: "GitHub API integration for repositories, issues, and PRs",
        sanitized_summary: "stdio command with 2 arguments, 1 env var",
      },
    },
    {
      name: "postgres",
      transport: "stdio",
      enabled: true,
      status: "connected",
      tool_count: 4,
      prompt_count: 1,
      resource_count: 5,
      last_discovered_at: "2026-09-18T10:30:00Z",
      updated_at: "2026-09-18T10:30:00Z",
      connection_summary: "stdio: npx -y @modelcontextprotocol/server-postgres",
      config_summary: {
        kind: "stdio",
        command: "npx",
        argument_count: 2,
        env_keys: ["DATABASE_URL"],
        description: "PostgreSQL database access for queries and schema inspection",
        sanitized_summary: "stdio command with 2 arguments, 1 env var",
      },
    },
    {
      name: "brave-search",
      transport: "stdio",
      enabled: true,
      status: "connected",
      tool_count: 2,
      prompt_count: 0,
      resource_count: 0,
      last_discovered_at: "2026-09-18T10:30:00Z",
      updated_at: "2026-09-18T10:30:00Z",
      connection_summary: "stdio: npx -y @modelcontextprotocol/server-brave-search",
      config_summary: {
        kind: "stdio",
        command: "npx",
        argument_count: 2,
        env_keys: ["BRAVE_API_KEY"],
        description: "Web search via Brave Search API",
        sanitized_summary: "stdio command with 2 arguments, 1 env var",
      },
    },
    {
      name: "puppeteer",
      transport: "stdio",
      enabled: false,
      status: "failed",
      tool_count: 3,
      prompt_count: 1,
      resource_count: 0,
      last_discovered_at: "2026-09-17T14:20:00Z",
      updated_at: "2026-09-18T09:15:00Z",
      connection_summary: "stdio: npx -y @modelcontextprotocol/server-puppeteer",
      config_summary: {
        kind: "stdio",
        command: "npx",
        argument_count: 2,
        env_keys: [],
        description: "Browser automation for web scrapingaping and testing",
        sanitized_summary: "stdio command with 2 arguments",
      },
    },
    {
      name: "slack",
      transport: "streamable_http",
      enabled: true,
      status: "reachable",
      tool_count: 6,
      prompt_count: 2,
      resource_count: 1,
      last_discovered_at: "2026-09-18T10:30:00Z",
      updated_at: "2026-09-18T10:30:00Z",
      connection_summary: "streamable_http: https://slack-mcp.example.com/mcp",
      config_summary: {
        kind: "streamable_http",
        target: "https://slack-mcp.example.com/mcp",
        header_keys: ["Authorization"],
        session_mode: "stateful",
        description: "Slack workspace integration for messages and channels",
        sanitized_summary: "streamable_http to remote endpoint, stateful sessions",
      },
    },
  ],
};

const mockTools = {
  tools: [
    {
      name: "read_file",
      canonical_name: "filesystem__read_file",
      server: "filesystem",
      description: "Read the contents of a file from the filesystem. Supports text files and returns content as text.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          path: { type: "string", description: "The absolute path to the file to read" },
          offset: { type: "number", description: "Line offset to start reading from (0-indexed)" },
          limit: { type: "number", description: "Maximum number of lines to read" },
        },
        required: ["path"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "write_file",
      canonical_name: "filesystem__write_file",
      server: "filesystem",
      description: "Write content to a file. Creates the file if it doesn't exist, overwrites if it does.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          path: { type: "string", description: "The absolute path to the file to write" },
          content: { type: "string", description: "The content to write to the file" },
        },
        required: ["path", "content"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "list_directory",
      canonical_name: "filesystem__list_directory",
      server: "filesystem",
      description: "List the contents of a directory. Returns file names, sizes, and modification dates.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          path: { type: "string", description: "The absolute path to the directory" },
        },
        required: ["path"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "search_files",
      canonical_name: "filesystem__search_files",
      server: "filesystem",
      description: "Search for files matching a glob pattern. Returns matching file paths.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          path: { type: "string", description: "The root directory to search in" },
          pattern: { type: "string", description: "Glob pattern to match files" },
        },
        required: ["path", "pattern"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "create_issue",
      canonical_name: "github__create_issue",
      server: "github",
      description: "Create a new issue in a GitHub repository. Supports title, body, labels, and assignees.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          owner: { type: "string", description: "Repository owner (user or org)" },
          repo: { type: "string", description: "Repository name" },
          title: { type: "string", description: "Issue title" },
          body: { type: "string", description: "Issue body in markdown" },
          labels: { type: "array", items: { type: "string" }, description: "Labels to apply" },
          assignees: { type: "array", items: { type: "string" }, description: "Users to assign" },
        },
        required: ["owner", "repo", "title"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "list_pull_requests",
      canonical_name: "github__list_pull_requests",
      server: "github",
      description: "List pull requests in a repository. Supports filtering by state, author, and labels.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          owner: { type: "string", description: "Repository owner" },
          repo: { type: "string", description: "Repository name" },
          state: { type: "string", enum: ["open", "closed", "all"], description: "Filter by PR state" },
        },
        required: ["owner", "repo"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "get_file_contents",
      canonical_name: "github__get_file_contents",
      server: "github",
      description: "Get the contents of a file or directory from a GitHub repository.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          owner: { type: "string" },
          repo: { type: "string" },
          path: { type: "string", description: "Path within the repository" },
          ref: { type: "string", description: "Branch, tag, or commit SHA" },
        },
        required: ["owner", "repo", "path"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "query",
      canonical_name: "postgres__query",
      server: "postgres",
      description: "Execute a read-only SQL query against the database. SELECT statements only.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          sql: { type: "string", description: "SQL query to execute (SELECT only)" },
          params: { type: "array", items: { type: "string" }, description: "Query parameters" },
        },
        required: ["sql"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "list_tables",
      canonical_name: "postgres__list_tables",
      server: "postgres",
      description: "List all tables in the database with their schemas and row counts.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          schema: { type: "string", description: "Schema name (defaults to 'public')" },
        },
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "describe_table",
      canonical_name: "postgres__describe_table",
      server: "postgres",
      description: "Get detailed schema information for a specific table including columns, types, and constraints.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          table_name: { type: "string", description: "Name of the table to describe" },
          schema: { type: "string", description: "Schema name (defaults to 'public')" },
        },
        required: ["table_name"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "brave_web_search",
      canonical_name: "brave-search__brave_web_search",
      server: "brave-search",
      description: "Search the web using Brave Search API. Returns relevant results with titles, URLs, and snippets.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          query: { type: "string", description: "Search query" },
          count: { type: "number", description: "Number of results (max 20)" },
          offset: { type: "number", description: "Result offset for pagination" },
        },
        required: ["query"],
      },
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "brave_local_search",
      canonical_name: "brave-search__brave_local_search",
      server: "brave-search",
      description: "Search for local businesses and places using Brave Search API.",
      enabled: true,
      server_enabled: true,
      input_schema: {
        type: "object",
        properties: {
          query: { type: "string", description: "Search query for local businesses" },
          location: { type: "string", description: "Geographic location to search near" },
        },
        required: ["query"],
      },
      transport: "stdio",
      server_status: "connected",
    },
  ],
};

const mockToolGroups = {
  tool_groups: [
    {
      name: "code-assistant",
      description: "Tools for code analysis, file manipulation, and repository operations",
      tool_count: 8,
      tools: [
        { name: "read_file", canonical_name: "filesystem__read_file", server: "filesystem" },
        { name: "write_file", canonical_name: "filesystem__write_file", server: "filesystem" },
        { name: "list_directory", canonical_name: "filesystem__list_directory", server: "filesystem" },
        { name: "search_files", canonical_name: "filesystem__search_files", server: "filesystem" },
        { name: "get_file_contents", canonical_name: "github__get_file_contents", server: "github" },
        { name: "list_pull_requests", canonical_name: "github__list_pull_requests", server: "github" },
        { name: "query", canonical_name: "postgres__query", server: "postgres" },
        { name: "describe_table", canonical_name: "postgres__describe_table", server: "postgres" },
      ],
      streamable_http_endpoint: "/mcp/code-assistant",
      sse_endpoint: "/mcp/code-assistant/sse",
      sse_message_endpoint: "/mcp/code-assistant/messages",
    },
    {
      name: "research",
      description: "Web search and information gathering tools",
      tool_count: 3,
      tools: [
        { name: "brave_web_search", canonical_name: "brave-search__brave_web_search", server: "brave-search" },
        { name: "brave_local_search", canonical_name: "brave-search__brave_local_search", server: "brave-search" },
        { name: "get_file_contents", canonical_name: "github__get_file_contents", server: "github" },
      ],
      streamable_http_endpoint: "/mcp/research",
      sse_endpoint: "/mcp/research/sse",
      sse_message_endpoint: "/mcp/research/messages",
    },
    {
      name: "project-management",
      description: "Issue tracking and project management tools",
      tool_count: 4,
      tools: [
        { name: "create_issue", canonical_name: "github__create_issue", server: "github" },
        { name: "list_pull_requests", canonical_name: "github__list_pull_requests", server: "github" },
        { name: "send_message", canonical_name: "slack__send_message", server: "slack" },
        { name: "list_channels", canonical_name: "slack__list_channels", server: "slack" },
      ],
      streamable_http_endpoint: "/mcp/project-management",
      sse_endpoint: "/mcp/project-management/sse",
      sse_message_endpoint: "/mcp/project-management/messages",
    },
  ],
};

const mockPrompts = {
  prompts: [
    {
      name: "code-review",
      canonical_name: "github__code-review",
      server: "github",
      description: "Analyze a pull request and provide code review suggestions with severity levels",
      enabled: true,
      server_enabled: true,
      arguments: [
        { name: "pr_number", type: "string", required: true, description: "Pull request number to review" },
        { name: "focus_areas", type: "string", required: false, description: "Comma-separated areas to focus on (security, performance, style)" },
      ],
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "summarize-changes",
      canonical_name: "github__summarize-changes",
      server: "github",
      description: "Generate a human-readable summary of changes in a commit or pull request",
      enabled: true,
      server_enabled: true,
      arguments: [
        { name: "ref", type: "string", required: true, description: "Commit SHA or PR number" },
      ],
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "database-report",
      canonical_name: "postgres__database-report",
      server: "postgres",
      description: "Generate a comprehensive report of database health, table sizes, and query performance",
      enabled: true,
      server_enabled: true,
      arguments: [
        { name: "include_slow_queries", type: "boolean", required: false, description: "Include slow query analysis" },
      ],
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "research-topic",
      canonical_name: "brave-search__research-topic",
      server: "brave-search",
      description: "Research a topic comprehensively using web search and compile findings into a report",
      enabled: true,
      server_enabled: true,
      arguments: [
        { name: "topic", type: "string", required: true, description: "Topic to research" },
        { name: "depth", type: "string", enum: ["brief", "standard", "deep"], required: false, description: "Research depth" },
      ],
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "explain-code",
      canonical_name: "github__explain-code",
      server: "github",
      description: "Explain a piece of code in plain language with examples and potential gotchas",
      enabled: true,
      server_enabled: true,
      arguments: [
        { name: "file_path", type: "string", required: true, description: "Path to the file to explain" },
        { name: "language", type: "string", required: false, description: "Programming language (auto-detected if omitted)" },
      ],
      transport: "stdio",
      server_status: "connected",
    },
    {
      name: "standup-summary",
      canonical_name: "slack__standup-summary",
      server: "slack",
      description: "Generate a summary of standup messages from a channel over a given time period",
      enabled: true,
      server_enabled: true,
      arguments: [
        { name: "channel", type: "string", required: true, description: "Slack channel name" },
        { name: "hours", type: "number", required: false, description: "Hours to look back (default: 24)" },
      ],
      transport: "streamable_http",
      server_status: "reachable",
    },
    {
      name: "incident-report",
      canonical_name: "slack__incident-report",
      server: "slack",
      description: "Compile an incident report from Slack thread discussions and linked issues",
      enabled: true,
      server_enabled: true,
      arguments: [
        { name: "thread_url", type: "string", required: true, description: "URL of the incident thread" },
      ],
      transport: "streamable_http",
      server_status: "reachable",
    },
    {
      name: "browser-test",
      canonical_name: "puppeteer__browser-test",
      server: "puppeteer",
      description: "Run browser automation tests and generate a report with screenshots",
      enabled: false,
      server_enabled: false,
      arguments: [
        { name: "test_suite", type: "string", required: true, description: "Path to test suite" },
      ],
      transport: "stdio",
      server_status: "failed",
    },
  ],
};

const mockResources = {
  resources: [
    {
      uri: "file:///Users/rafaelferreira/Documents/project-structure.md",
      name: "project-structure.md",
      server: "filesystem",
      description: "Project structure documentation with file tree and descriptions",
      mime_type: "text/markdown",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "file:///Users/rafaelferreira/Documents/api-spec.yaml",
      name: "api-spec.yaml",
      server: "filesystem",
      description: "OpenAPI specification for the project REST API",
      mime_type: "text/yaml",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "file:///Users/rafaelferreira/Documents/CHANGELOG.md",
      name: "CHANGELOG.md",
      server: "filesystem",
      description: "Project changelog with version history and release notes",
      mime_type: "text/markdown",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "github://mcpjungle/MCPJungle/README.md",
      name: "README.md",
      server: "github",
      description: "Repository readme file with project overview and setup instructions",
      mime_type: "text/markdown",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "github://mcpjungle/MCPJungle/CONTRIBUTING.md",
      name: "CONTRIBUTING.md",
      server: "github",
      description: "Contribution guidelines for the project",
      mime_type: "text/markdown",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "postgres://localhost/myapp/users",
      name: "users table",
      server: "postgres",
      description: "User accounts table with authentication and profile data",
      mime_type: "application/sql",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "postgres://localhost/myapp/projects",
      name: "projects table",
      server: "postgres",
      description: "Projects table with metadata and team associations",
      mime_type: "application/sql",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "postgres://localhost/myapp/audit_log",
      name: "audit_log table",
      server: "postgres",
      description: "Audit log table tracking all user actions and system events",
      mime_type: "application/sql",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "postgres://localhost/myapp/sessions",
      name: "sessions table",
      server: "postgres",
      description: "Active user sessions with expiration and device information",
      mime_type: "application/sql",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "postgres://localhost/myapp/settings",
      name: "settings table",
      server: "postgres",
      description: "User preferences and application settings",
      mime_type: "application/sql",
      enabled: true,
      transport: "stdio",
      server_status: "connected",
    },
    {
      uri: "slack://workspace/channels/general",
      name: "#general channel",
      server: "slack",
      description: "General discussion channel with team announcements",
      mime_type: "application/slack",
      enabled: true,
      transport: "streamable_http",
      server_status: "reachable",
    },
    {
      uri: "slack://workspace/channels/engineering",
      name: "#engineering channel",
      server: "slack",
      description: "Engineering team discussions and technical decisions",
      mime_type: "application/slack",
      enabled: true,
      transport: "streamable_http",
      server_status: "reachable",
    },
  ],
};

const mockConnections = {
  static_connections: [
    {
      name: "claude-desktop",
      description: "Claude Desktop app for local development",
      allow_list: ["filesystem", "github", "postgres", "brave-search"],
      last_seen_at: "2026-09-18T10:45:00Z",
    },
    {
      name: "vscode-extension",
      description: "VS Code MCP extension for integrated development",
      allow_list: ["filesystem", "github", "postgres"],
      last_seen_at: "2026-09-18T09:30:00Z",
    },
    {
      name: "cursor-ide",
      description: "Cursor IDE integration",
      allow_list: ["filesystem", "github", "postgres", "brave-search", "slack"],
      last_seen_at: "2026-09-18T10:50:00Z",
    },
  ],
  hosted_connections: [
    {
      client_id: "prod-agent-01",
      tool_group: "code-assistant",
      endpoint: "/mcp/code-assistant",
      last_seen_at: "2026-09-18T10:52:00Z",
    },
    {
      client_id: "research-bot",
      tool_group: "research",
      endpoint: "/mcp/research",
      last_seen_at: "2026-09-18T10:48:00Z",
    },
    {
      client_id: "ci-pipeline",
      tool_group: "project-management",
      endpoint: "/mcp/project-management",
      last_seen_at: "2026-09-18T10:55:00Z",
    },
  ],
};

const mockDiagnostics = {
  version: "1.4.2",
  mode: "enterprise",
  config_source: "config.yaml",
  config_path: "/etc/mcpjungle/config.yaml",
  database: "SQLite: /var/lib/mcpjungle/data.db",
  enabled_transports: ["stdio", "streamable_http", "sse"],
  metrics_endpoint: "http://localhost:8080/metrics",
  primary_endpoint: "http://localhost:8080/mcp",
  troubleshooting_hints: [
    "Check server logs at /var/log/mcpjungle/server.log for detailed errors",
    "Verify environment variables are set for servers that require API keys",
    "Ensure stdio servers have their dependencies installed globally",
    "For streamable_http servers, verify the remote endpoint is accessible",
  ],
  server_count: 6,
  tool_count: 23,
  prompt_count: 8,
  resource_count: 12,
};

const routes: Record<string, unknown> = {
  "/api/dashboard/overview": mockOverview,
  "/api/dashboard/servers": mockServers,
  "/api/dashboard/tools": mockTools,
  "/api/dashboard/tool-groups": mockToolGroups,
  "/api/dashboard/prompts": mockPrompts,
  "/api/dashboard/resources": mockResources,
  "/api/dashboard/connections": mockConnections,
  "/api/dashboard/diagnostics": mockDiagnostics,
};

export function mockServerPlugin(): Plugin {
  return {
    name: "mcpjungle-mock-server",
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        if (!req.url || req.method !== "GET") return next();

        const urlPath = req.url.split("?")[0];
        const data = routes[urlPath];

        if (data) {
          res.setHeader("Content-Type", "application/json");
          res.end(JSON.stringify(data));
          return;
        }

        // Pass through for non-dashboard API routes (auth, etc.)
        if (urlPath.startsWith("/api/dashboard/")) {
          res.statusCode = 404;
          res.setHeader("Content-Type", "application/json");
          res.end(JSON.stringify({ error: "Mock not implemented for " + urlPath }));
          return;
        }

        next();
      });
    },
  };
}
