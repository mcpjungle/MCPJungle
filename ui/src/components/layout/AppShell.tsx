import { Outlet } from "react-router-dom";

import { useAppContext } from "../../App";
import { NavSidebar } from "./NavSidebar";

function formatVersion(v: string): string {
  // Real semver tag: v1.2.3 → show as-is
  if (/^v?\d+\.\d+\.\d+$/.test(v)) return v;
  // dev fallback
  if (v === "dev") return "dev";
  // Go pseudo-version: v0.0.0-20260429133657-2a8c08eceb82+dirty → dev (abc1234)
  const pseudoMatch = v.match(/^v0\.0\.0-\d+-([0-9a-f]+)/);
  if (pseudoMatch) {
    const dirty = v.includes("+dirty") ? "*" : "";
    return `dev (${pseudoMatch[1].slice(0, 7)}${dirty})`;
  }
  // Anything else: truncate to 20 chars
  return v.length > 20 ? v.slice(0, 20) + "…" : v;
}

export function AppShell() {
  const { clearToken, refresh, settings, user, metadata } = useAppContext();

  return (
    <div className="min-h-screen bg-shell px-4 py-4 text-body md:px-6">
      <div className="mx-auto grid max-w-[1480px] gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
        <NavSidebar />
        <div className="rounded-panel border border-line bg-panel/70 shadow-panel">
          <header className="flex flex-col gap-3 border-b border-line px-6 py-4 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-3">
              <span
                className={`rounded px-2 py-0.5 text-xs font-medium uppercase tracking-wider ${
                  settings.mode === "development"
                    ? "bg-accent/15 text-accent"
                    : "bg-up/15 text-up"
                }`}
              >
                {settings.mode}
              </span>
              {user ? (
                <span className="text-sm text-muted">
                  {user.username}
                  {user.role === "admin" ? (
                    <span className="ml-1.5 text-xs text-muted/60">admin</span>
                  ) : null}
                </span>
              ) : null}
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <span
                className="rounded border border-line bg-shell px-2.5 py-1.5 font-mono text-[11px] text-muted/70"
                title={metadata.version}
              >
                {formatVersion(metadata.version)}
              </span>
              <button
                className="rounded border border-line bg-shell px-3 py-1.5 text-sm text-body transition hover:border-accent/40 hover:text-accent"
                onClick={() => void refresh()}
              >
                Refresh
              </button>
              {settings.mode !== "development" ? (
                <button
                  className="rounded bg-accent px-3 py-1.5 text-sm font-semibold text-ink transition hover:bg-accent/90"
                  onClick={clearToken}
                >
                  Sign out
                </button>
              ) : null}
            </div>
          </header>
          <main className="p-6">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
