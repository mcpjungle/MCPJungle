import type { Tool } from "../../lib/types";

type ToolCardProps = {
  tool: Tool;
  canManage: boolean;
  active: boolean;
  onInvoke: (tool: Tool) => void;
  onToggle: (tool: Tool) => void;
};

export function ToolCard({ tool, canManage, active, onInvoke, onToggle }: ToolCardProps) {
  return (
    <article className={`rounded-panel border p-5 ${active ? "border-accent bg-accent/5" : "border-line bg-[#11161d]"}`}>
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h3 className="text-lg font-semibold text-body">{tool.name}</h3>
            <span className={`rounded-full px-2 py-1 text-[11px] uppercase tracking-[0.18em] ${tool.enabled ? "bg-up/10 text-up" : "bg-down/10 text-down"}`}>
              {tool.enabled ? "Enabled" : "Disabled"}
            </span>
          </div>
          <p className="mt-3 text-sm leading-6 text-muted">{tool.description || "No description yet."}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button className="rounded-md bg-accent px-3 py-2 text-sm font-semibold text-ink" onClick={() => onInvoke(tool)}>
            Invoke
          </button>
          {canManage ? (
            <button className="rounded-md border border-line bg-shell px-3 py-2 text-sm text-body" onClick={() => onToggle(tool)}>
              {tool.enabled ? "Disable" : "Enable"}
            </button>
          ) : null}
        </div>
      </div>
    </article>
  );
}
