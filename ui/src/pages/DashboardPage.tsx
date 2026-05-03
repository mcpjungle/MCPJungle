import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { useAppContext } from "../App";
import { StatCard } from "../components/cards/StatCard";
import { api } from "../lib/api";

export function DashboardPage() {
  const { token, settings, user, isAdminEquivalent } = useAppContext();
  const serversQuery = useQuery({ queryKey: ["servers"], queryFn: () => api.servers(token || undefined) });
  const toolsQuery = useQuery({ queryKey: ["tools"], queryFn: () => api.tools(token || undefined) });
  const groupsQuery = useQuery({
    queryKey: ["tool-groups"],
    queryFn: () => api.toolGroups(token || undefined),
    enabled: isAdminEquivalent,
  });
  const clientsQuery = useQuery({
    queryKey: ["clients"],
    queryFn: () => api.clients(token || undefined),
    enabled: isAdminEquivalent && settings.mode !== "development",
  });

  const servers = serversQuery.data ?? [];
  const tools = toolsQuery.data ?? [];

  return (
    <div className="space-y-6">
      {/* Status bar */}
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Health" value="OK" tone="up" detail="Gateway reachable" />
        <StatCard
          label="Mode"
          value={settings.mode}
          tone="accent"
          detail={user ? `Logged in as ${user.username}` : "No auth required"}
        />
        <StatCard
          label="Servers"
          value={serversQuery.isLoading ? "…" : servers.length}
          detail={servers.length > 0 ? `${servers.length} upstream endpoint${servers.length !== 1 ? "s" : ""}` : "None registered"}
          linkTo="/servers"
        />
        <StatCard
          label="Tools"
          value={toolsQuery.isLoading ? "…" : tools.length}
          detail={tools.length > 0 ? `Across ${servers.length} server${servers.length !== 1 ? "s" : ""}` : "Register a server to populate"}
          linkTo="/tools"
        />
      </section>

      {isAdminEquivalent ? (
        <section className="grid gap-4 md:grid-cols-2">
          <StatCard
            label="Tool groups"
            value={groupsQuery.isLoading ? "…" : (groupsQuery.data?.length ?? 0)}
            detail="Scoped subsets exposed to clients"
            linkTo="/tool-groups"
          />
          <StatCard
            label="Clients"
            value={settings.mode === "development" ? "—" : (clientsQuery.isLoading ? "…" : (clientsQuery.data?.length ?? 0))}
            detail={settings.mode === "development" ? "Available in enterprise mode" : "Registered MCP consumers"}
            linkTo={settings.mode !== "development" ? "/clients" : undefined}
          />
        </section>
      ) : null}

      {/* Quick actions */}
      {servers.length === 0 && !serversQuery.isLoading ? (
        <section className="rounded-panel border border-accent/30 bg-accent/5 p-6">
          <p className="text-sm font-medium text-accent">No servers registered yet</p>
          <p className="mt-1 text-sm text-muted">
            Register an upstream MCP server to start routing tool calls through the gateway.
          </p>
          <Link
            to="/servers/new"
            className="mt-4 inline-block rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink hover:bg-accent/90"
          >
            Register server
          </Link>
        </section>
      ) : null}
    </div>
  );
}
