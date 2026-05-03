import { useState } from "react";

type ServerHint = { name: string; description?: string };

type AllowListEditorProps = {
  value: string[];
  onChange: (next: string[]) => void;
  servers?: ServerHint[];
};

export function AllowListEditor({ value, onChange, servers = [] }: AllowListEditorProps) {
  const [draft, setDraft] = useState("");

  const isWildcard = value.length === 1 && value[0] === "*";

  function toggleWildcard(toWildcard: boolean) {
    onChange(toWildcard ? ["*"] : []);
  }

  function toggleServer(name: string) {
    const without = value.filter((v) => v !== "*");
    if (without.includes(name)) {
      onChange(without.filter((v) => v !== name));
    } else {
      onChange([...without, name]);
    }
  }

  function add() {
    const trimmed = draft.trim();
    if (!trimmed) return;
    if (trimmed === "*") { toggleWildcard(true); setDraft(""); return; }
    if (value.includes(trimmed)) { setDraft(""); return; }
    onChange([...value.filter((v) => v !== "*"), trimmed]);
    setDraft("");
  }

  function remove(item: string) {
    onChange(value.filter((v) => v !== item));
  }

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === "Enter") { e.preventDefault(); add(); }
  }

  return (
    <div className="space-y-3">
      {/* Wildcard toggle */}
      <label className="flex cursor-pointer select-none items-center gap-2 text-sm text-body">
        <input
          type="checkbox"
          checked={isWildcard}
          onChange={(e) => toggleWildcard(e.target.checked)}
          className="accent-yellow-400"
        />
        <span>Allow access to <strong>all servers</strong> (wildcard <code className="font-mono text-xs">*</code>)</span>
      </label>

      {/* Server checkboxes when servers are known */}
      {!isWildcard && servers.length > 0 && (
        <div className="rounded-ui border border-line bg-shell p-3">
          <p className="mb-2 text-xs text-muted">Select servers:</p>
          <div className="max-h-48 space-y-1.5 overflow-y-auto">
            {servers.map((s) => (
              <label key={s.name} className="flex cursor-pointer select-none items-center gap-2 text-sm text-body">
                <input
                  type="checkbox"
                  checked={value.includes(s.name)}
                  onChange={() => toggleServer(s.name)}
                  className="accent-yellow-400"
                />
                <span className="font-mono text-xs">{s.name}</span>
                {s.description && (
                  <span className="truncate text-xs text-muted">— {s.description}</span>
                )}
              </label>
            ))}
          </div>
        </div>
      )}

      {/* Manual text entry (always available as fallback) */}
      {!isWildcard && (
        <div className="flex gap-2">
          <input
            className="flex-1 rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body placeholder:text-muted focus:border-accent focus:outline-none"
            placeholder="Or type a server name and press Enter"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={handleKey}
          />
          <button
            type="button"
            className="rounded-md bg-accent px-3 py-2 text-sm font-semibold text-ink disabled:opacity-40"
            disabled={!draft.trim()}
            onClick={add}
          >
            Add
          </button>
        </div>
      )}

      {/* Current selection tags */}
      {!isWildcard && value.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {value.map((item) => (
            <span
              key={item}
              className="flex items-center gap-1.5 rounded-md border border-line bg-elevated px-2.5 py-1 text-xs text-body"
            >
              {item}
              <button
                type="button"
                className="text-muted transition hover:text-down"
                onClick={() => remove(item)}
                aria-label={`Remove ${item}`}
              >
                ✕
              </button>
            </span>
          ))}
        </div>
      )}

      {!isWildcard && value.length === 0 && (
        <p className="text-xs text-down">No servers selected — this client will have no access.</p>
      )}

      <p className="text-xs text-muted">
        {isWildcard
          ? "This client can access all registered MCP servers."
          : value.length > 0
          ? `Access restricted to: ${value.join(", ")}`
          : "No access granted yet."}
      </p>
    </div>
  );
}
