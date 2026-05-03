import { FormEvent, useState } from "react";

import type { MetadataResponse, ServerMode } from "../../lib/types";

type InitGateProps = {
  metadata: MetadataResponse;
  onReady: (token: string) => void;
  onRefresh: () => Promise<void>;
};

type InitResponse = {
  admin_access_token?: string;
};

export function InitGate({ metadata, onReady, onRefresh }: InitGateProps) {
  const [mode, setMode] = useState<ServerMode>("development");
  const [error, setError] = useState("");
  const [working, setWorking] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setWorking(true);
    setError("");

    try {
      const response = await fetch("/init", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode }),
      });
      const payload = (await response.json()) as InitResponse & { error?: string };

      if (!response.ok) {
        setError(payload.error ?? "Initialization failed");
        return;
      }

      if (payload.admin_access_token) {
        onReady(payload.admin_access_token);
        return;
      }

      await onRefresh();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Initialization failed");
    } finally {
      setWorking(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-shell px-6 py-12 text-body">
      <div className="w-full max-w-md">
        {/* Logo mark */}
        <div className="mb-8 flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent text-ink">
            <svg className="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
              <path d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <div>
            <p className="text-sm font-semibold text-body">MCPJungle</p>
            <p className="text-xs text-muted">First-time setup</p>
          </div>
        </div>

        {/* Card */}
        <div className="rounded-panel border border-line bg-panel p-8 shadow-panel">
          <h1 className="text-xl font-semibold text-body">Choose a mode</h1>
          <p className="mt-1 text-sm text-muted">
            This only needs to be done once. The mode cannot be changed without resetting the gateway.
          </p>

          <form className="mt-6 space-y-3" onSubmit={handleSubmit}>
            <button
              type="button"
              onClick={() => setMode("development")}
              className={`w-full rounded-md border px-4 py-4 text-left transition ${
                mode === "development"
                  ? "border-accent bg-accent/10"
                  : "border-line bg-shell hover:border-line/80"
              }`}
            >
              <div className="flex items-center justify-between">
                <p className="text-sm font-semibold text-body">Development</p>
                {mode === "development" && (
                  <span className="rounded bg-accent px-2 py-0.5 text-xs font-medium text-ink">Selected</span>
                )}
              </div>
              <p className="mt-1 text-xs text-muted">No auth required. All requests pass through. Best for local use.</p>
            </button>

            <button
              type="button"
              onClick={() => setMode("enterprise")}
              className={`w-full rounded-md border px-4 py-4 text-left transition ${
                mode === "enterprise"
                  ? "border-accent bg-accent/10"
                  : "border-line bg-shell hover:border-line/80"
              }`}
            >
              <div className="flex items-center justify-between">
                <p className="text-sm font-semibold text-body">Enterprise</p>
                {mode === "enterprise" && (
                  <span className="rounded bg-accent px-2 py-0.5 text-xs font-medium text-ink">Selected</span>
                )}
              </div>
              <p className="mt-1 text-xs text-muted">
                Token auth on every request. Generates an admin token. Enables user and client management.
              </p>
            </button>

            {error ? (
              <p className="rounded-md border border-down/30 bg-down/5 px-3 py-2 text-sm text-down">{error}</p>
            ) : null}

            <button
              className="w-full rounded-md bg-accent py-2.5 text-sm font-semibold text-ink transition hover:bg-accent/90 disabled:opacity-40"
              disabled={working}
              type="submit"
            >
              {working ? "Initializing…" : "Initialize gateway"}
            </button>
          </form>
        </div>

        <p className="mt-5 text-center font-mono text-xs text-muted/50" title={metadata.version}>
          {metadata.version.length > 20 ? "dev" : metadata.version}
        </p>
      </div>
    </div>
  );
}
