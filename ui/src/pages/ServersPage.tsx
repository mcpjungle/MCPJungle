import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { useAppContext } from "../App";
import { ServerCard } from "../components/cards/ServerCard";
import { ConfirmDialog } from "../components/ui/ConfirmDialog";
import { EmptyState } from "../components/ui/EmptyState";
import { api } from "../lib/api";
import type { RegisterServerInput } from "../lib/types";

// ─── types for mcp-proxy / Claude Desktop servers.json ───────────────────────
type McpServerEntry = {
  enabled?: boolean;
  url?: string;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  headers?: Record<string, string>;
};
type ServersJson = { mcpServers: Record<string, McpServerEntry> };

function parseServersJson(raw: string): RegisterServerInput[] {
  const parsed = JSON.parse(raw) as ServersJson;
  const entries = parsed.mcpServers ?? {};
  return Object.entries(entries).map(([name, cfg]) => {
    const transport = cfg.command ? "stdio" : "streamable_http";
    return {
      name,
      transport,
      description: "",
      url: cfg.url,
      headers: cfg.headers,
      command: cfg.command,
      args: cfg.args,
      env: cfg.env,
      session_mode: "stateless",
    } satisfies RegisterServerInput;
  });
}

// ─── Import dialog ────────────────────────────────────────────────────────────
type ImportResult = { name: string; status: "ok" | "error"; message?: string };

function ImportDialog({
  token,
  onClose,
  onDone,
}: {
  token: string | null;
  onClose: () => void;
  onDone: () => void;
}) {
  const fileRef = useRef<HTMLInputElement>(null);
  const [parsed, setParsed] = useState<RegisterServerInput[] | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [parseError, setParseError] = useState("");
  const [results, setResults] = useState<ImportResult[] | null>(null);
  const [importing, setImporting] = useState(false);

  function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      try {
        const servers = parseServersJson(ev.target?.result as string);
        setParsed(servers);
        setSelected(new Set(servers.map((s) => s.name)));
        setParseError("");
        setResults(null);
      } catch {
        setParseError("Could not parse file — expected { mcpServers: { ... } } format.");
        setParsed(null);
      }
    };
    reader.readAsText(file);
  }

  function handlePaste(e: React.ChangeEvent<HTMLTextAreaElement>) {
    try {
      const servers = parseServersJson(e.target.value);
      setParsed(servers);
      setSelected(new Set(servers.map((s) => s.name)));
      setParseError("");
      setResults(null);
    } catch {
      setParsed(null);
      if (e.target.value.trim()) setParseError("Invalid JSON — expected { mcpServers: { ... } } format.");
      else setParseError("");
    }
  }

  function toggle(name: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }

  async function handleImport() {
    if (!parsed) return;
    const toImport = parsed.filter((s) => selected.has(s.name));
    setImporting(true);
    const res: ImportResult[] = [];
    for (const server of toImport) {
      try {
        await api.createServer(server, token || undefined, true);
        res.push({ name: server.name, status: "ok" });
      } catch (err) {
        res.push({ name: server.name, status: "error", message: (err as Error).message });
      }
    }
    setResults(res);
    setImporting(false);
    if (res.every((r) => r.status === "ok")) {
      setTimeout(() => { onDone(); onClose(); }, 800);
    }
  }

  const allOk = results?.every((r) => r.status === "ok");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div className="w-full max-w-xl rounded-panel border border-line bg-panel shadow-panel">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-line px-6 py-4">
          <h3 className="text-base font-semibold text-body">Import servers</h3>
          <button className="text-muted hover:text-body" onClick={onClose}>
            <svg className="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="space-y-4 p-6">
          {/* File picker */}
          <div>
            <p className="mb-2 text-xs font-medium text-muted">
              Load a <code className="text-body">servers.json</code> (mcp-proxy / Claude Desktop format)
            </p>
            <div className="flex gap-2">
              <button
                className="rounded-md border border-line bg-shell px-3 py-2 text-sm text-body transition hover:border-accent/50 hover:text-accent"
                onClick={() => fileRef.current?.click()}
              >
                Choose file…
              </button>
              <input ref={fileRef} type="file" accept=".json,application/json" className="hidden" onChange={handleFile} />
              <span className="flex items-center text-xs text-muted">or paste JSON below</span>
            </div>
          </div>

          {/* Paste area */}
          <textarea
            className="w-full rounded-md border border-line bg-shell px-3 py-2 font-mono text-xs text-body outline-none transition focus:border-accent"
            rows={5}
            placeholder={'{\n  "mcpServers": {\n    "my-server": { "command": "npx", "args": ["..."] }\n  }\n}'}
            onChange={handlePaste}
          />

          {parseError ? (
            <p className="text-xs text-down">{parseError}</p>
          ) : null}

          {/* Server list */}
          {parsed && parsed.length > 0 ? (
            <div className="space-y-2">
              <p className="text-xs font-medium text-muted">
                {parsed.length} server{parsed.length !== 1 ? "s" : ""} found — select which to import:
              </p>
              <div className="max-h-48 overflow-y-auto space-y-1 rounded-md border border-line bg-shell p-2">
                {parsed.map((s) => (
                  <label key={s.name} className="flex cursor-pointer items-center gap-3 rounded px-2 py-1.5 hover:bg-panel">
                    <input
                      type="checkbox"
                      checked={selected.has(s.name)}
                      onChange={() => toggle(s.name)}
                      className="accent-accent"
                    />
                    <div className="min-w-0 flex-1">
                      <span className="text-sm font-medium text-body">{s.name}</span>
                      <span className="ml-2 rounded bg-elevated px-1.5 py-0.5 font-mono text-[10px] text-muted">
                        {s.transport}
                      </span>
                      {s.command ? (
                        <p className="truncate font-mono text-[11px] text-muted/70">{s.command} {s.args?.join(" ")}</p>
                      ) : s.url ? (
                        <p className="truncate font-mono text-[11px] text-muted/70">{s.url}</p>
                      ) : null}
                    </div>
                  </label>
                ))}
              </div>
            </div>
          ) : null}

          {/* Results */}
          {results ? (
            <div className="space-y-1">
              {results.map((r) => (
                <div key={r.name} className={`flex items-center gap-2 text-sm ${r.status === "ok" ? "text-up" : "text-down"}`}>
                  <span>{r.status === "ok" ? "✓" : "✗"}</span>
                  <span className="font-medium">{r.name}</span>
                  {r.message ? <span className="text-xs text-muted">{r.message}</span> : null}
                </div>
              ))}
              {allOk ? <p className="text-xs text-up">All done — closing…</p> : null}
            </div>
          ) : null}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 border-t border-line px-6 py-4">
          <button className="rounded-md border border-line px-4 py-2 text-sm text-body hover:text-accent" onClick={onClose}>
            Cancel
          </button>
          <button
            className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink disabled:opacity-40"
            disabled={!parsed || selected.size === 0 || importing || !!results}
            onClick={() => void handleImport()}
          >
            {importing ? `Importing…` : `Import ${selected.size > 0 ? selected.size : ""} server${selected.size !== 1 ? "s" : ""}`}
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────
export function ServersPage() {
  const { token, isAdminEquivalent } = useAppContext();
  const queryClient = useQueryClient();
  const [pendingDelete, setPendingDelete] = useState<string>("");
  const [showImport, setShowImport] = useState(false);

  const serversQuery = useQuery({ queryKey: ["servers"], queryFn: () => api.servers(token || undefined) });
  const enableMutation = useMutation({
    mutationFn: (name: string) => api.enableServer(name, token || undefined),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["servers"] }),
  });
  const disableMutation = useMutation({
    mutationFn: (name: string) => api.disableServer(name, token || undefined),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["servers"] }),
  });
  const deleteMutation = useMutation({
    mutationFn: (name: string) => api.deleteServer(name, token || undefined),
    onSuccess: () => {
      setPendingDelete("");
      void queryClient.invalidateQueries({ queryKey: ["servers"] });
    },
  });

  const servers = serversQuery.data ?? [];

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.22em] text-muted">Registry</p>
          <h2 className="mt-2 text-3xl font-semibold text-body">Servers</h2>
        </div>
        {isAdminEquivalent ? (
          <div className="flex gap-2">
            <button
              className="rounded-md border border-line bg-shell px-4 py-2.5 text-sm font-medium text-body transition hover:border-accent/50 hover:text-accent"
              onClick={() => setShowImport(true)}
            >
              Import from file
            </button>
            <Link className="rounded-md bg-accent px-4 py-2.5 text-sm font-semibold text-ink" to="/servers/new">
              Register server
            </Link>
          </div>
        ) : null}
      </div>

      {serversQuery.isLoading ? (
        <p className="text-sm text-muted">Loading servers…</p>
      ) : serversQuery.isError ? (
        <div className="rounded-panel border border-down/30 bg-panel p-4 text-sm text-down">
          Failed to load servers: {(serversQuery.error as Error).message}
        </div>
      ) : null}

      {(enableMutation.isError || disableMutation.isError || deleteMutation.isError) ? (
        <div className="rounded-panel border border-down/30 bg-panel p-3 text-sm text-down">
          {((enableMutation.error ?? disableMutation.error ?? deleteMutation.error) as Error | null)?.message}
        </div>
      ) : null}

      {servers.length ? (
        <div className="space-y-4">
          {servers.map((server) => (
            <div className="space-y-2" key={server.name}>
              <ServerCard
                server={server}
                canManage={isAdminEquivalent}
                onEnable={(name) => enableMutation.mutate(name)}
                onDisable={(name) => disableMutation.mutate(name)}
                onDelete={(name) => setPendingDelete(name)}
              />
              {isAdminEquivalent ? (
                <div className="flex justify-end">
                  <Link className="text-sm text-accent" to={`/servers/${encodeURIComponent(server.name)}/edit`}>
                    Edit config
                  </Link>
                </div>
              ) : null}
            </div>
          ))}
        </div>
      ) : !serversQuery.isLoading ? (
        <div className="rounded-panel border border-dashed border-line bg-shell/60 p-10 text-center">
          <p className="text-base font-semibold text-body">No servers registered</p>
          <p className="mt-2 text-sm text-muted">Import a <code>servers.json</code> or register one manually.</p>
          {isAdminEquivalent ? (
            <div className="mt-5 flex justify-center gap-3">
              <button
                className="rounded-md border border-line bg-shell px-4 py-2 text-sm text-body transition hover:border-accent/50 hover:text-accent"
                onClick={() => setShowImport(true)}
              >
                Import from file
              </button>
              <Link className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink" to="/servers/new">
                Register server
              </Link>
            </div>
          ) : null}
        </div>
      ) : null}

      {pendingDelete ? (
        <ConfirmDialog
          title="Delete server"
          message={`Delete "${pendingDelete}" from MCPJungle registry? Connected clients will lose access immediately.`}
          confirmLabel="Delete"
          onConfirm={() => deleteMutation.mutate(pendingDelete)}
          onCancel={() => setPendingDelete("")}
        />
      ) : null}

      {showImport ? (
        <ImportDialog
          token={token}
          onClose={() => setShowImport(false)}
          onDone={() => void queryClient.invalidateQueries({ queryKey: ["servers"] })}
        />
      ) : null}
    </div>
  );
}
