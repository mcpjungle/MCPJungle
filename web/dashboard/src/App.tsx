import { useEffect, useMemo, useState } from "react";
import logoUrl from "@repo-assets/logo.png";
import diagramUrl from "@repo-assets/mcpjungle-diagram/april-2026/mcpjungle-diagram.png";
import { api } from "@/lib/api";
import type {
  AppSection,
  DashboardDiagnosticsResponse,
  DashboardOverviewResponse,
  DashboardPromptsResponse,
  DashboardResourcesResponse,
  DashboardServersResponse,
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

export default function App() {
  const [section, setSection] = useState<AppSection>("overview");
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [errorMessage, setErrorMessage] = useState("");
  const [data, setData] = useState<DashboardData>({});
  const [serverFilter, setServerFilter] = useState("");
  const [toolFilter, setToolFilter] = useState("");
  const [expandedServer, setExpandedServer] = useState<string | null>(null);

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
    const tools = data.tools?.tools ?? [];
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
  }, [data.tools?.tools, toolFilter]);

  const overview = data.overview;
  const diagnostics = data.diagnostics;

  return (
    <div className="app-shell">
      <NavSidebar active={section} logoUrl={logoUrl} onSelect={setSection} />
      <main className="main-shell">
        <header className="topbar">
          <div>
            <p className="eyeline">MCPJungle</p>
            <h1>Overview</h1>
            <p className="topbar-subtitle">High-level view of your MCPJungle gateway.</p>
          </div>
          <div className="topbar-meta">
            {overview ? (
              <StatusBadge text={overview.status} tone={toneForStatus(overview.status)} />
            ) : null}
            {overview?.version ? <span className="version-chip">{overview.version}</span> : null}
            {overview?.endpoints[0] ? (
              <div className="topbar-endpoint">
                <code>{overview.endpoints[0].url}</code>
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
            {section === "overview" && overview && diagnostics ? (
              <>
                <section className="hero-grid">
                  <div className="hero-card">
                    <div className="hero-header">
                      <p className="panel-label">Gateway status</p>
                    </div>
                    <div className="hero-summary-grid">
                      <div className="hero-status-block">
                        <h2>Running</h2>
                        <p>MCPJungle is serving your local MCP gateway in <strong>{overview.mode}</strong> mode.</p>
                        <div className="mode-chip">Mode: {overview.mode}</div>
                      </div>
                      <div className="hero-endpoint-block">
                        <span>Primary MCP endpoint</span>
                        <code>{overview.endpoints[0]?.url}</code>
                        <div className="hero-endpoint-actions">
                          <CopyButton value={overview.endpoints[0]?.url ?? ""} />
                        </div>
                      </div>
                    </div>
                    <p className="hero-copy">
                      Use this single local endpoint in your MCP clients to access the registered servers, tools, prompts, and resources behind MCPJungle.
                    </p>
                  </div>

                  <aside className="hero-side">
                    <div className="metric-card">
                      <span>Registered servers</span>
                      <strong>{overview.server_count}</strong>
                    </div>
                    <div className="metric-card">
                      <span>Discovered tools</span>
                      <strong>{overview.tool_count}</strong>
                    </div>
                    <div className="metric-card">
                      <span>Prompts</span>
                      <strong>{overview.prompt_count}</strong>
                    </div>
                    <div className="metric-card">
                      <span>Resources</span>
                      <strong>{overview.resource_count}</strong>
                    </div>
                  </aside>
                </section>

                <section className="overview-lower-grid">
                  <SectionCard
                    title="Server inventory"
                    subtitle="Registered MCP servers"
                    action={<input className="table-filter" onChange={(event) => setServerFilter(event.target.value)} placeholder="Filter servers" value={serverFilter} />}
                  >
                    {filteredServers.length === 0 && data.servers?.empty_state ? (
                      <EmptyStateCard emptyState={data.servers.empty_state} />
                    ) : (
                      <table className="data-table">
                        <thead>
                          <tr>
                            <th>Server</th>
                            <th>Transport</th>
                            <th>Status</th>
                            <th>Tools</th>
                            <th>Prompts</th>
                            <th>Resources</th>
                            <th>Last discovered</th>
                          </tr>
                        </thead>
                        <tbody>
                          {filteredServers.slice(0, 5).map((server) => (
                            <tr key={server.name}>
                              <td>{server.name}</td>
                              <td>{server.transport}</td>
                              <td>
                                <StatusBadge text={server.status} tone={toneForStatus(server.status)} />
                              </td>
                              <td>{server.tool_count}</td>
                              <td>{server.prompt_count}</td>
                              <td>{server.resource_count}</td>
                              <td>{formatDate(server.last_discovered_at)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )}
                  </SectionCard>

                  <div className="side-column">
                    <SectionCard title="System" subtitle="Diagnostics at a glance">
                      <dl className="diagnostic-list">
                        <div>
                          <dt>Database</dt>
                          <dd>{diagnostics.database}</dd>
                        </div>
                        <div>
                          <dt>Runtime mode</dt>
                          <dd>{diagnostics.mode}</dd>
                        </div>
                        <div>
                          <dt>Metrics</dt>
                          <dd>{diagnostics.metrics_endpoint ?? "Disabled"}</dd>
                        </div>
                        <div>
                          <dt>Enabled transports</dt>
                          <dd>{diagnostics.enabled_transports.join(", ")}</dd>
                        </div>
                      </dl>
                    </SectionCard>

                    <SectionCard title="Brand map" subtitle="Gateway mental model">
                      <img alt="MCPJungle architecture diagram" className="diagram-card" src={diagramUrl} />
                    </SectionCard>
                  </div>
                </section>

                <SectionCard title="Troubleshooting" subtitle="What to check first">
                  <div className="hint-grid">
                    {(overview.troubleshooting ?? []).map((hint) => (
                      <div className="hint-card" key={hint}>
                        {hint}
                      </div>
                    ))}
                  </div>
                </SectionCard>
              </>
            ) : null}

            {section === "servers" && data.servers ? (
              <SectionCard
                title="Servers"
                subtitle="Registered MCP servers"
                action={<input className="table-filter" onChange={(event) => setServerFilter(event.target.value)} placeholder="Search name, transport, or summary" value={serverFilter} />}
              >
                {data.servers.empty_state && filteredServers.length === 0 ? (
                  <EmptyStateCard emptyState={data.servers.empty_state} />
                ) : (
                  <div className="server-list">
                    {filteredServers.map((server) => {
                      const expanded = expandedServer === server.name;
                      return (
                        <article className="server-row" key={server.name}>
                          <button
                            className="server-row-head"
                            onClick={() => setExpandedServer(expanded ? null : server.name)}
                            type="button"
                          >
                            <div>
                              <h3>{server.name}</h3>
                              <p>{server.connection_summary}</p>
                            </div>
                            <div className="server-row-meta">
                              <StatusBadge text={server.status} tone={toneForStatus(server.status)} />
                              <span>{server.transport}</span>
                              <strong>{server.tool_count} tools</strong>
                            </div>
                          </button>
                          {expanded ? (
                            <div className="server-detail">
                              <dl>
                                <div>
                                  <dt>Session mode</dt>
                                  <dd>{server.config_summary.session_mode ?? "Unknown"}</dd>
                                </div>
                                <div>
                                  <dt>Target</dt>
                                  <dd>{server.config_summary.target ?? server.config_summary.command ?? "Unknown"}</dd>
                                </div>
                                <div>
                                  <dt>Header keys</dt>
                                  <dd>{server.config_summary.header_keys?.join(", ") || "None"}</dd>
                                </div>
                                <div>
                                  <dt>Env keys</dt>
                                  <dd>{server.config_summary.env_keys?.join(", ") || "None"}</dd>
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
            ) : null}

            {section === "tools" && data.tools ? (
              <SectionCard
                title="Tools"
                subtitle="All discovered tools"
                action={<input className="table-filter" onChange={(event) => setToolFilter(event.target.value)} placeholder="Search tool or server" value={toolFilter} />}
              >
                {data.tools.empty_state && filteredTools.length === 0 ? (
                  <EmptyStateCard emptyState={data.tools.empty_state} />
                ) : (
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>Tool</th>
                        <th>Canonical name</th>
                        <th>Server</th>
                        <th>Description</th>
                        <th>Input schema</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredTools.map((tool) => (
                        <tr key={tool.canonical_name}>
                          <td>{tool.name}</td>
                          <td><code>{tool.canonical_name}</code></td>
                          <td>{tool.server}</td>
                          <td>{tool.description || "No description"}</td>
                          <td><code>{tool.input_preview || "No schema"}</code></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </SectionCard>
            ) : null}

            {section === "prompts" && data.prompts ? (
              <SectionCard title="Prompts" subtitle="Discovered prompt templates">
                {data.prompts.empty_state && data.prompts.prompts.length === 0 ? (
                  <EmptyStateCard emptyState={data.prompts.empty_state} />
                ) : (
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>Prompt</th>
                        <th>Server</th>
                        <th>Description</th>
                        <th>Arguments</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.prompts.prompts.map((prompt) => (
                        <tr key={prompt.canonical_name}>
                          <td>{prompt.name}</td>
                          <td>{prompt.server}</td>
                          <td>{prompt.description || "No description"}</td>
                          <td><code>{prompt.arguments_preview || "No arguments"}</code></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </SectionCard>
            ) : null}

            {section === "resources" && data.resources ? (
              <SectionCard title="Resources" subtitle="Discovered MCP resources">
                {data.resources.empty_state && data.resources.resources.length === 0 ? (
                  <EmptyStateCard emptyState={data.resources.empty_state} />
                ) : (
                  <table className="data-table">
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
                          <td><code>{resource.uri}</code></td>
                          <td>{resource.server}</td>
                          <td>{resource.mime_type || "Unknown"}</td>
                          <td>{resource.description || "No description"}</td>
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
                  <div className="diagnostics-grid">
                    <div className="diag-card">
                      <span>Version</span>
                      <strong>{diagnostics.version}</strong>
                    </div>
                    <div className="diag-card">
                      <span>Runtime mode</span>
                      <strong>{diagnostics.mode}</strong>
                    </div>
                    <div className="diag-card">
                      <span>Database</span>
                      <strong>{diagnostics.database}</strong>
                    </div>
                    <div className="diag-card">
                      <span>Primary endpoint</span>
                      <div className="inline-copy">
                        <code>{diagnostics.primary_endpoint}</code>
                        <CopyButton value={diagnostics.primary_endpoint} />
                      </div>
                    </div>
                  </div>
                </SectionCard>

                <SectionCard title="Runtime details" subtitle="Safe system information">
                  <dl className="diagnostic-list">
                    <div>
                      <dt>Config source</dt>
                      <dd>{diagnostics.config_source ?? "Unknown"}</dd>
                    </div>
                    <div>
                      <dt>Metrics endpoint</dt>
                      <dd>{diagnostics.metrics_endpoint ?? "Disabled"}</dd>
                    </div>
                    <div>
                      <dt>Enabled transports</dt>
                      <dd>{diagnostics.enabled_transports.join(", ")}</dd>
                    </div>
                  </dl>
                </SectionCard>

                <SectionCard title="Troubleshooting" subtitle="Common local issues">
                  <div className="hint-grid">
                    {diagnostics.troubleshooting_hints.map((hint) => (
                      <div className="hint-card" key={hint}>
                        {hint}
                      </div>
                    ))}
                  </div>
                  {diagnostics.empty_state ? <EmptyStateCard emptyState={diagnostics.empty_state} /> : null}
                </SectionCard>
              </>
            ) : null}
          </div>
        ) : null}
      </main>
    </div>
  );
}
