import type { McpServer } from "../../lib/types";

type ServerCardProps = {
  server: McpServer;
  canManage: boolean;
  onEnable: (name: string) => void;
  onDisable: (name: string) => void;
  onDelete: (name: string) => void;
};

export function ServerCard({ server, canManage, onEnable, onDisable, onDelete }: ServerCardProps) {
  return (
    <article className="rounded-panel border border-line bg-[#11161d] p-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h3 className="text-lg font-semibold text-body">{server.name}</h3>
            <span className="rounded-full border border-line px-2 py-1 text-[11px] uppercase tracking-[0.18em] text-accent">
              {server.transport}
            </span>
            <span className="rounded-full border border-line px-2 py-1 text-[11px] uppercase tracking-[0.18em] text-muted">
              {server.session_mode}
            </span>
          </div>
          <p className="mt-3 text-sm leading-6 text-muted">{server.description || "No description yet."}</p>
          <p className="mt-3 text-xs text-muted">
            {server.url ? `URL: ${server.url}` : `Command: ${server.command ?? "n/a"}`}
          </p>
        </div>
        {canManage ? (
          <div className="flex flex-wrap gap-2">
            <button className="rounded-md border border-line bg-shell px-3 py-2 text-sm text-up" onClick={() => onEnable(server.name)}>
              Enable
            </button>
            <button
              className="rounded-md border border-line bg-shell px-3 py-2 text-sm text-body"
              onClick={() => onDisable(server.name)}
            >
              Disable
            </button>
            <button className="rounded-md border border-down/40 bg-shell px-3 py-2 text-sm text-down" onClick={() => onDelete(server.name)}>
              Delete
            </button>
          </div>
        ) : null}
      </div>
    </article>
  );
}
