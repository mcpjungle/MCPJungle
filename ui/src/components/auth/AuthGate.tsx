import { FormEvent, useState } from "react";

import type { MetadataResponse } from "../../lib/types";

function formatVersion(v: string): string {
  if (/^v?\d+\.\d+\.\d+$/.test(v)) return v;
  if (v === "dev") return "dev";
  const m = v.match(/^v0\.0\.0-\d+-([0-9a-f]+)/);
  if (m) return `dev (${m[1].slice(0, 7)}${v.includes("+dirty") ? "*" : ""})`;
  return v.length > 20 ? v.slice(0, 20) + "…" : v;
}

type AuthGateProps = {
  metadata: MetadataResponse;
  message: string;
  currentToken: string;
  onSubmit: (token: string) => void;
  onClear: () => void;
};

export function AuthGate({ metadata, message, currentToken, onSubmit, onClear }: AuthGateProps) {
  const [token, setToken] = useState(currentToken);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSubmit(token.trim());
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
            <p className="text-xs text-muted">Enterprise</p>
          </div>
        </div>

        {/* Card */}
        <div className="rounded-panel border border-line bg-panel p-8 shadow-panel">
          <h1 className="text-xl font-semibold text-body">Sign in</h1>
          <p className="mt-1 text-sm text-muted">{message || "Enter your access token to continue."}</p>

          <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
            <label className="block">
              <span className="mb-1.5 block text-xs font-medium text-muted">Access token</span>
              <textarea
                className="min-h-[88px] w-full rounded-md border border-line bg-shell px-3 py-2.5 font-mono text-sm text-body outline-none transition focus:border-accent"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="Paste your bearer token here"
                autoFocus
              />
            </label>
            <div className="flex gap-2">
              <button
                className="flex-1 rounded-md bg-accent py-2.5 text-sm font-semibold text-ink transition hover:bg-accent/90 disabled:opacity-40"
                type="submit"
                disabled={!token.trim()}
              >
                Sign in
              </button>
              {token ? (
                <button
                  className="rounded-md border border-line px-4 py-2.5 text-sm text-muted transition hover:text-body"
                  type="button"
                  onClick={() => { setToken(""); onClear(); }}
                >
                  Clear
                </button>
              ) : null}
            </div>
          </form>
        </div>

        {/* Footer */}
        <p className="mt-5 text-center font-mono text-xs text-muted/50" title={metadata.version}>
          {formatVersion(metadata.version)}
        </p>
      </div>
    </div>
  );
}
