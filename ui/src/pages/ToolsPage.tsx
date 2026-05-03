import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { useAppContext } from "../App";
import { ToolCard } from "../components/cards/ToolCard";
import { JSONSchemaForm } from "../components/forms/JSONSchemaForm";
import { EmptyState } from "../components/ui/EmptyState";
import { api } from "../lib/api";
import type { Tool, ToolInvokeResult } from "../lib/types";

function renderContent(item: Record<string, unknown>): string {
  if (typeof item.text === "string") return item.text;
  return JSON.stringify(item, null, 2);
}

function InvokeResult({ result, error }: { result: ToolInvokeResult | null; error: string }) {
  if (error) {
    return (
      <div className="mt-5 rounded-ui border border-down/40 bg-down/5 p-4 text-sm text-down">
        <p className="font-medium">Invocation error</p>
        <p className="mt-1 text-xs">{error}</p>
      </div>
    );
  }
  if (!result) return null;

  const isErr = result.isError;
  const contents = result.content ?? [];

  return (
    <div className={`mt-5 rounded-ui border p-4 ${isErr ? "border-down/40 bg-down/5" : "border-up/30 bg-up/5"}`}>
      <div className="mb-2 flex items-center gap-2">
        <span className={`text-xs font-medium uppercase tracking-[0.14em] ${isErr ? "text-down" : "text-up"}`}>
          {isErr ? "Error result" : "Success"}
        </span>
        {result.structuredContent ? (
          <span className="rounded bg-elevated px-1.5 py-0.5 font-mono text-[10px] text-muted">
            structured
          </span>
        ) : null}
      </div>
      {contents.length > 0 ? (
        <div className="space-y-3">
          {contents.map((item, idx) => {
            const rendered = renderContent(item);
            const isJson = !item.text;
            return (
              <pre
                key={idx}
                className={`overflow-x-auto rounded border border-line bg-shell p-3 text-xs leading-5 ${
                  isErr ? "text-down" : "text-body"
                } ${isJson ? "font-mono" : ""}`}
              >
                {rendered}
              </pre>
            );
          })}
        </div>
      ) : result.structuredContent ? (
        <pre className="overflow-x-auto rounded border border-line bg-shell p-3 font-mono text-xs text-body">
          {JSON.stringify(result.structuredContent, null, 2)}
        </pre>
      ) : (
        <p className="text-xs text-muted">(empty result)</p>
      )}
    </div>
  );
}

export function ToolsPage() {
  const { token, isAdminEquivalent } = useAppContext();
  const [selectedServer, setSelectedServer] = useState("");
  const [activeTool, setActiveTool] = useState<Tool | null>(null);
  const [invokeResult, setInvokeResult] = useState<ToolInvokeResult | null>(null);
  const [invokeError, setInvokeError] = useState("");

  const serversQuery = useQuery({ queryKey: ["servers"], queryFn: () => api.servers(token || undefined) });
  const toolsQuery = useQuery({
    queryKey: ["tools", selectedServer],
    queryFn: () => api.tools(token || undefined, selectedServer || undefined),
  });

  const invokeMutation = useMutation({
    mutationFn: (payload: Record<string, unknown>) => api.invokeTool(token || undefined, payload),
    onSuccess: (result) => {
      setInvokeError("");
      setInvokeResult(result);
    },
    onError: (err: Error) => {
      setInvokeResult(null);
      setInvokeError(err.message);
    },
  });
  const toggleMutation = useMutation({
    mutationFn: (tool: Tool) =>
      tool.enabled ? api.disableTool(tool.name, token || undefined) : api.enableTool(tool.name, token || undefined),
    onSuccess: () => void toolsQuery.refetch(),
  });

  const serverOptions = useMemo(() => serversQuery.data ?? [], [serversQuery.data]);
  const tools = toolsQuery.data ?? [];

  function handleToolSelect(tool: Tool) {
    if (activeTool?.name !== tool.name) {
      setInvokeResult(null);
      setInvokeError("");
    }
    setActiveTool(tool);
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.22em] text-muted">Invocation surface</p>
          <h2 className="mt-2 text-3xl font-semibold text-body">Tools</h2>
        </div>
        <select
          className="w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body xl:max-w-sm"
          value={selectedServer}
          onChange={(event) => {
            setSelectedServer(event.target.value);
            setActiveTool(null);
            setInvokeResult(null);
            setInvokeError("");
          }}
        >
          <option value="">All servers</option>
          {serverOptions.map((server) => (
            <option key={server.name} value={server.name}>
              {server.name}
            </option>
          ))}
        </select>
      </div>

      {toolsQuery.isError ? (
        <div className="rounded-panel border border-down/30 bg-panel p-3 text-sm text-down">
          Failed to load tools: {(toolsQuery.error as Error).message}
        </div>
      ) : null}

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1.2fr)_420px]">
        <section className="space-y-4">
          {toolsQuery.isLoading ? (
            <p className="text-sm text-muted">Loading tools…</p>
          ) : tools.length ? (
            tools.map((tool) => (
              <ToolCard
                key={tool.name}
                tool={tool}
                canManage={isAdminEquivalent}
                active={activeTool?.name === tool.name}
                onInvoke={handleToolSelect}
                onToggle={(next) => toggleMutation.mutate(next)}
              />
            ))
          ) : (
            <EmptyState title="No tools found" message="Adjust server filter or register upstream server first." />
          )}
        </section>

        <section className="rounded-panel border border-line bg-panel p-6">
          {activeTool ? (
            <>
              <p className="text-xs uppercase tracking-[0.22em] text-accent">Invoke</p>
              <h3 className="mt-3 text-xl font-semibold text-body">{activeTool.name}</h3>
              <p className="mt-2 text-sm leading-6 text-muted">{activeTool.description}</p>
              <div
                className={`mt-1.5 inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${
                  activeTool.enabled
                    ? "bg-up/10 text-up"
                    : "bg-down/10 text-down"
                }`}
              >
                {activeTool.enabled ? "enabled" : "disabled"}
              </div>
              <div className="mt-5">
                <JSONSchemaForm
                  schema={activeTool.input_schema}
                  onSubmit={(values) => {
                    setInvokeResult(null);
                    setInvokeError("");
                    invokeMutation.mutate({ name: activeTool.name, ...values });
                  }}
                />
                {invokeMutation.isPending ? (
                  <p className="mt-3 text-xs text-muted">Running…</p>
                ) : null}
              </div>
              <InvokeResult result={invokeResult} error={invokeError} />
            </>
          ) : (
            <EmptyState title="Select a tool" message="Click a tool card to open the invoke panel." />
          )}
        </section>
      </div>
    </div>
  );
}
