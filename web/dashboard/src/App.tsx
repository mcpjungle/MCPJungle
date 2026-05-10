import { useEffect, useMemo, useState } from "react";
import logoUrl from "@repo-assets/logo.png";
import { api } from "@/lib/api";
import type {
  AppSection,
  DashboardDiagnosticsResponse,
  DashboardOverviewResponse,
  DashboardPrompt,
  DashboardPromptsResponse,
  DashboardResource,
  DashboardResourcesResponse,
  DashboardServer,
  DashboardServersResponse,
  DashboardTool,
  DashboardToolsResponse,
} from "@/lib/types";
import { CopyButton } from "@/components/CopyButton";
import { EmptyStateCard } from "@/components/EmptyStateCard";
import { NavSidebar } from "@/components/NavSidebar";
import { SectionCard } from "@/components/SectionCard";
import { StatusBadge } from "@/components/StatusBadge";

type LoadState = "idle" | "loading" | "ready" | "error";

interface DashboardData {
  overview?: DashboardOverviewResponse;
  servers?: DashboardServersResponse;
  tools?: DashboardToolsResponse;
  prompts?: DashboardPromptsResponse;
  resources?: DashboardResourcesResponse;
  diagnostics?: DashboardDiagnosticsResponse;
}

const sectionMeta: Record<AppSection, { title: string; subtitle: string }> = {
  servers: {
    title: "Servers",
    subtitle: "Registered MCP backends and discovery details.",
  },
  tools: {
    title: "Tools",
    subtitle: "All discovered tools across registered servers.",
  },
  prompts: {
    title: "Prompts",
    subtitle: "Prompt templates currently exposed through MCPJungle.",
  },
  resources: {
    title: "Resources",
    subtitle: "Resources registered and proxied through the gateway.",
  },
  diagnostics: {
    title: "Diagnostics",
    subtitle: "Runtime health, build info, and troubleshooting signals.",
  },
};

function formatDate(value?: string) {
  if (!value) {
    return "Unknown";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Unknown";
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function toneForStatus(status: string) {
  switch (status) {
    case "running":
    case "connected":
    case "reachable":
      return "good";
    case "degraded":
      return "warn";
    case "failed":
      return "bad";
    default:
      return "muted";
  }
}

function shortVersion(version?: string) {
  if (!version) {
    return "";
  }
  const match = version.match(/v?\d+\.\d+\.\d+/);
  if (match) {
    return match[0];
  }
  return version.length > 16 ? version.slice(0, 16) : version;
}

function transportLabel(value?: string) {
  return value ? value.split("_").join(" ") : "unknown";
}

function toolDescription(tool: DashboardTool) {
  return tool.description || "No description";
}

function promptDescription(prompt: DashboardPrompt) {
  return prompt.description || "No description";
}

function resourceDescription(resource: DashboardResource) {
  return resource.description || "No description";
}

function prettyJSON(value?: Record<string, unknown>) {
  if (!value) {
    return "No schema available.";
  }
  return JSON.stringify(value, null, 2);
}

function prettyPromptArguments(value?: Array<Record<string, unknown>>) {
  if (!value || value.length === 0) {
    return "No arguments";
  }
  return JSON.stringify(value, null, 2);
}

function discoveryState(server: DashboardServer) {
  if (server.tool_count + server.prompt_count + server.resource_count === 0) {
    return { label: "No discovery", tone: "warn" };
  }
  return { label: "Discovered", tone: "good" };
}

export default function App() {
  const [section, setSection] = useState<AppSection>("servers");
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [errorMessage, setErrorMessage] = useState("");
  const [data, setData] = useState<DashboardData>({});
  const [serverFilter, setServerFilter] = useState("");
  const [toolFilter, setToolFilter] = useState("");
  const [toolServerFilter, setToolServerFilter] = useState("all");
  const [expandedServer, setExpandedServer] = useState<string | null>(null);
  const [selectedTool, setSelectedTool] = useState<DashboardTool | null>(null);
  const [selectedPrompt, setSelectedPrompt] = useState<DashboardPrompt | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoadState("loading");
    Promise.all([
      api.overview(),
      api.servers(),
      api.tools(),
      api.prompts(),
      api.resources(),
      api.diagnostics(),
    ])
      .then(([overview, servers, tools, prompts, resources, diagnostics]) => {
        if (cancelled) {
          return;
        }
        setData({ overview, servers, tools, prompts, resources, diagnostics });
        setSelectedTool(tools.tools[0] ?? null);
        setSelectedPrompt(prompts.prompts[0] ?? null);
        setLoadState("ready");
      })
      .catch((error: Error) => {
        if (cancelled) {
          return;
        }
        setErrorMessage(error.message);
        setLoadState("error");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const filteredServers = useMemo(() => {
    const servers = data.servers?.servers ?? [];
    if (!serverFilter.trim()) {
      return servers;
    }
    const term = serverFilter.toLowerCase();
    return servers.filter(
      (server) =>
        server.name.toLowerCase().includes(term) ||
        server.transport.toLowerCase().includes(term) ||
        server.connection_summary.toLowerCase().includes(term),
    );
  }, [data.servers?.servers, serverFilter]);

  const filteredTools = useMemo(() => {
    let tools = data.tools?.tools ?? [];
    if (toolServerFilter !== "all") {
      tools = tools.filter((tool) => tool.server === toolServerFilter);
    }
    if (!toolFilter.trim()) {
      return tools;
    }
    const term = toolFilter.toLowerCase();
    return tools.filter(
      (tool) =>
        tool.name.toLowerCase().includes(term) ||
        tool.server.toLowerCase().includes(term) ||
        tool.canonical_name.toLowerCase().includes(term),
    );
  }, [data.tools?.tools, toolFilter, toolServerFilter]);

  const uniqueToolServers = useMemo(() => {
    const servers = new Set((data.tools?.tools ?? []).map((tool) => tool.server));
    return Array.from(servers).sort();
  }, [data.tools?.tools]);

  const overview = data.overview;
  const diagnostics = data.diagnostics;
  const currentSectionMeta = sectionMeta[section];

  return (
    <div className="app-shell">
      <NavSidebar active={section} logoUrl={logoUrl} onSelect={setSection} />
      <main className="main-shell">
        <header className="topbar">
          <div>
            <h1>{currentSectionMeta.title}</h1>
            {currentSectionMeta.subtitle ? (
              <p className="topbar-subtitle">{currentSectionMeta.subtitle}</p>
            ) : null}
          </div>
          <div className="topbar-meta">
            {overview?.version ? (
              <span className="version-chip">{`Server version ${shortVersion(overview.version)}`}</span>
            ) : null}
            {overview?.endpoints[0] ? (
              <div className="topbar-endpoint">
                <span className="topbar-endpoint-label">Endpoint</span>
                <code title={overview.endpoints[0].url}>{overview.endpoints[0].url}</code>
                <CopyButton value={overview.endpoints[0].url} />
              </div>
            ) : null}
          </div>
        </header>

        {loadState === "loading" ? (
          <section className="loading-screen panel">
            <h2>Loading dashboard</h2>
            <p>Querying local MCPJungle state, servers, tools, prompts, and resources.</p>
          </section>
        ) : null}

        {loadState === "error" ? (
          <section className="loading-screen panel error-screen">
            <h2>Dashboard API unavailable</h2>
            <p>Failed to load dashboard data from the local server.</p>
            <code>{errorMessage}</code>
          </section>
        ) : null}

        {loadState === "ready" ? (
          <div className="content-grid">
            {section === "servers" && data.servers ? (
              <>
                {overview ? (
                  <section className="dense-metrics-grid">
                    <div className="metric-card compact-metric">
                      <span>Servers</span>
                      <strong>{overview.server_count}</strong>
                    </div>
                    <div className="metric-card compact-metric">
                      <span>Tools</span>
                      <strong>{overview.tool_count}</strong>
                    </div>
                    <div className="metric-card compact-metric">
                      <span>Prompts</span>
                      <strong>{overview.prompt_count}</strong>
                    </div>
                    <div className="metric-card compact-metric">
                      <span>Resources</span>
                      <strong>{overview.resource_count}</strong>
                    </div>
                  </section>
                ) : null}

                <SectionCard
                  title="Servers"
                  subtitle="Registered MCP servers"
                  action={
                    <input
                      className="table-filter compact-filter"
                      onChange={(event) => setServerFilter(event.target.value)}
                      placeholder="Search name, transport, or summary"
                      value={serverFilter}
                    />
                  }
                >
                  {data.servers.empty_state && filteredServers.length === 0 ? (
                    <EmptyStateCard emptyState={data.servers.empty_state} />
                  ) : (
                    <div className="server-list compact-server-list">
                      {filteredServers.map((server) => {
                        const expanded = expandedServer === server.name;
                        const discovery = discoveryState(server);
                        return (
                          <article className="server-row compact-server-row" key={server.name}>
                            <button
                              className="server-row-head compact-server-head"
                              onClick={() => setExpandedServer(expanded ? null : server.name)}
                              type="button"
                            >
                              <div className="server-head-main">
                                <h3>{server.name}</h3>
                                <p>{server.connection_summary}</p>
                              </div>
                              <div className="server-row-meta compact-server-meta">
                                <code>{transportLabel(server.transport)}</code>
                                <StatusBadge
                                  text={server.status}
                                  tone={toneForStatus(server.status)}
                                />
                                <StatusBadge text={discovery.label} tone={discovery.tone} />
                                <strong>{server.tool_count} tools</strong>
                              </div>
                            </button>
                            {expanded ? (
                              <div className="server-detail">
                                <dl>
                                  <div>
                                    <dt>Target</dt>
                                    <dd>
                                      <code>
                                        {server.config_summary.target ??
                                          server.config_summary.command ??
                                          "Unknown"}
                                      </code>
                                    </dd>
                                  </div>
                                  <div>
                                    <dt>Session mode</dt>
                                    <dd>
                                      <code>
                                        {server.config_summary.session_mode ?? "Unknown"}
                                      </code>
                                    </dd>
                                  </div>
                                  <div>
                                    <dt>Header keys</dt>
                                    <dd>
                                      <code>
                                        {server.config_summary.header_keys?.join(", ") || "None"}
                                      </code>
                                    </dd>
                                  </div>
                                  <div>
                                    <dt>Env keys</dt>
                                    <dd>
                                      <code>
                                        {server.config_summary.env_keys?.join(", ") || "None"}
                                      </code>
                                    </dd>
                                  </div>
                                  <div>
                                    <dt>Last discovered</dt>
                                    <dd>{formatDate(server.last_discovered_at)}</dd>
                                  </div>
                                  <div>
                                    <dt>Updated</dt>
                                    <dd>{formatDate(server.updated_at)}</dd>
                                  </div>
                                </dl>
                              </div>
                            ) : null}
                          </article>
                        );
                      })}
                    </div>
                  )}
                </SectionCard>
              </>
            ) : null}

            {section === "tools" && data.tools ? (
              <SectionCard
                title="Tools"
                subtitle="Discovered tools with canonical names and schema inspector"
                action={
                  <div className="toolbar-cluster">
                    <input
                      className="table-filter compact-filter"
                      onChange={(event) => setToolFilter(event.target.value)}
                      placeholder="Search tool or server"
                      value={toolFilter}
                    />
                    <select
                      className="table-filter compact-filter compact-select"
                      onChange={(event) => setToolServerFilter(event.target.value)}
                      value={toolServerFilter}
                    >
                      <option value="all">All servers</option>
                      {uniqueToolServers.map((server) => (
                        <option key={server} value={server}>
                          {server}
                        </option>
                      ))}
                    </select>
                  </div>
                }
              >
                {data.tools.empty_state && filteredTools.length === 0 ? (
                  <EmptyStateCard emptyState={data.tools.empty_state} />
                ) : (
                  <div className="tools-layout">
                    <div className="tools-table-wrap">
                      <table className="data-table compact-table tools-table">
                        <thead>
                          <tr>
                            <th>Tool</th>
                            <th>Canonical name</th>
                            <th>Server</th>
                            <th>Description</th>
                            <th>Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          {filteredTools.map((tool) => (
                            <tr
                              className={selectedTool?.canonical_name === tool.canonical_name ? "is-selected" : ""}
                              key={tool.canonical_name}
                            >
                              <td>
                                <div className="table-primary">{tool.name}</div>
                                <div className="table-secondary">
                                  <code>{transportLabel(tool.transport)}</code>
                                </div>
                              </td>
                              <td>
                                <code className="identifier-code" title={tool.canonical_name}>
                                  {tool.canonical_name}
                                </code>
                              </td>
                              <td>{tool.server}</td>
                              <td>{toolDescription(tool)}</td>
                              <td>
                                <div className="row-actions">
                                  <CopyButton value={tool.canonical_name} />
                                  <button
                                    className="secondary-action"
                                    onClick={() => setSelectedTool(tool)}
                                    type="button"
                                  >
                                    View schema
                                  </button>
                                </div>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>

                    <aside className="schema-panel">
                      <div className="schema-panel-header">
                        <div>
                          <p className="panel-label">Tool schema</p>
                          <h3>{selectedTool?.name ?? "Select a tool"}</h3>
                          {selectedTool ? (
                            <code className="identifier-code">{selectedTool.canonical_name}</code>
                          ) : null}
                        </div>
                        {selectedTool ? <CopyButton value={selectedTool.canonical_name} /> : null}
                      </div>
                      <div className="schema-panel-body">
                        {selectedTool ? (
                          <>
                            <dl className="schema-meta">
                              <div>
                                <dt>Server</dt>
                                <dd>{selectedTool.server}</dd>
                              </div>
                              <div>
                                <dt>Description</dt>
                                <dd>{toolDescription(selectedTool)}</dd>
                              </div>
                            </dl>
                            <pre className="schema-code">
                              <code>{prettyJSON(selectedTool.input_schema)}</code>
                            </pre>
                          </>
                        ) : (
                          <p className="empty-inline">
                            Select a tool row to inspect and pretty-print its input schema.
                          </p>
                        )}
                      </div>
                    </aside>
                  </div>
                )}
              </SectionCard>
            ) : null}

            {section === "prompts" && data.prompts ? (
              <SectionCard title="Prompts" subtitle="Discovered prompt templates">
                {data.prompts.empty_state && data.prompts.prompts.length === 0 ? (
                  <EmptyStateCard emptyState={data.prompts.empty_state} />
                ) : (
                  <div className="tools-layout">
                    <div className="tools-table-wrap">
                      <table className="data-table compact-table prompts-table">
                        <thead>
                          <tr>
                            <th>Prompt</th>
                            <th>Canonical name</th>
                            <th>Server</th>
                            <th>Description</th>
                            <th>Action</th>
                          </tr>
                        </thead>
                        <tbody>
                          {data.prompts.prompts.map((prompt) => (
                            <tr
                              className={
                                selectedPrompt?.canonical_name === prompt.canonical_name
                                  ? "is-selected"
                                  : ""
                              }
                              key={prompt.canonical_name}
                            >
                              <td>
                                <div className="table-primary">{prompt.name}</div>
                              </td>
                              <td>
                                <code
                                  className="identifier-code"
                                  title={prompt.canonical_name}
                                >
                                  {prompt.canonical_name}
                                </code>
                              </td>
                              <td>{prompt.server}</td>
                              <td>
                                <div
                                  className="clamped-description"
                                  title={promptDescription(prompt)}
                                >
                                  {promptDescription(prompt)}
                                </div>
                              </td>
                              <td>
                                <button
                                  className="secondary-action"
                                  onClick={() => setSelectedPrompt(prompt)}
                                  type="button"
                                >
                                  View arguments
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>

                    <aside className="schema-panel">
                      <div className="schema-panel-header">
                        <div>
                          <p className="panel-label">Prompt arguments</p>
                          <h3>{selectedPrompt?.name ?? "Select a prompt"}</h3>
                          {selectedPrompt ? (
                            <code className="identifier-code">
                              {selectedPrompt.canonical_name}
                            </code>
                          ) : null}
                        </div>
                      </div>
                      <div className="schema-panel-body">
                        {selectedPrompt ? (
                          <>
                            <dl className="schema-meta">
                              <div>
                                <dt>Server</dt>
                                <dd>{selectedPrompt.server}</dd>
                              </div>
                              <div>
                                <dt>Description</dt>
                                <dd>{promptDescription(selectedPrompt)}</dd>
                              </div>
                            </dl>
                            <pre className="schema-code">
                              <code>
                                {prettyPromptArguments(selectedPrompt.arguments)}
                              </code>
                            </pre>
                          </>
                        ) : (
                          <p className="empty-inline">
                            Select a prompt row to inspect its arguments.
                          </p>
                        )}
                      </div>
                    </aside>
                  </div>
                )}
              </SectionCard>
            ) : null}

            {section === "resources" && data.resources ? (
              <SectionCard title="Resources" subtitle="Discovered MCP resources">
                {data.resources.empty_state && data.resources.resources.length === 0 ? (
                  <EmptyStateCard emptyState={data.resources.empty_state} />
                ) : (
                  <table className="data-table compact-table">
                    <thead>
                      <tr>
                        <th>Name</th>
                        <th>URI</th>
                        <th>Server</th>
                        <th>MIME</th>
                        <th>Description</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.resources.resources.map((resource) => (
                        <tr key={resource.uri}>
                          <td>{resource.name}</td>
                          <td>
                            <code className="identifier-code">{resource.uri}</code>
                          </td>
                          <td>{resource.server}</td>
                          <td>
                            <code>{resource.mime_type || "Unknown"}</code>
                          </td>
                          <td>{resourceDescription(resource)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </SectionCard>
            ) : null}

            {section === "diagnostics" && diagnostics ? (
              <>
                <SectionCard title="Diagnostics" subtitle="Runtime and troubleshooting">
                  <div className="diagnostics-grid compact-diagnostics-grid">
                    <div className="diag-card compact-metric">
                      <span>Version</span>
                      <strong>{shortVersion(diagnostics.version)}</strong>
                    </div>
                    <div className="diag-card compact-metric">
                      <span>Mode</span>
                      <strong>{diagnostics.mode}</strong>
                    </div>
                    <div className="diag-card compact-metric">
                      <span>Database</span>
                      <strong>{diagnostics.database}</strong>
                    </div>
                    <div className="diag-card compact-metric">
                      <span>Warnings</span>
                      <strong>{diagnostics.troubleshooting_hints.length}</strong>
                    </div>
                  </div>
                </SectionCard>

                <SectionCard title="Runtime details" subtitle="Safe system information">
                  <dl className="diagnostic-list compact-diagnostic-list">
                    <div>
                      <dt>Full build</dt>
                      <dd>
                        <code>{diagnostics.version}</code>
                      </dd>
                    </div>
                    <div>
                      <dt>Config source</dt>
                      <dd>{diagnostics.config_source ?? "Unknown"}</dd>
                    </div>
                    <div>
                      <dt>Primary endpoint</dt>
                      <dd>
                        <code>{diagnostics.primary_endpoint}</code>
                      </dd>
                    </div>
                    <div>
                      <dt>Metrics endpoint</dt>
                      <dd>{diagnostics.metrics_endpoint ?? "Disabled"}</dd>
                    </div>
                    <div>
                      <dt>Enabled transports</dt>
                      <dd>
                        <code>{diagnostics.enabled_transports.join(", ")}</code>
                      </dd>
                    </div>
                  </dl>
                </SectionCard>

                <SectionCard title="Troubleshooting" subtitle="Common local issues">
                  <div className="hint-grid compact-hint-grid">
                    {diagnostics.troubleshooting_hints.map((hint) => (
                      <div className="hint-card compact-hint-card" key={hint}>
                        {hint}
                      </div>
                    ))}
                  </div>
                  {diagnostics.empty_state ? (
                    <EmptyStateCard emptyState={diagnostics.empty_state} />
                  ) : null}
                </SectionCard>
              </>
            ) : null}
          </div>
        ) : null}
      </main>
    </div>
  );
}
