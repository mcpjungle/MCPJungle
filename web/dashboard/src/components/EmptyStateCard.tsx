import { CopyButton } from "./CopyButton";
import type { DashboardEmptyState } from "@/lib/types";

export function EmptyStateCard({ emptyState }: { emptyState: DashboardEmptyState }) {
  return (
    <section className="panel empty-state">
      <div>
        <p className="panel-label">Empty state</p>
        <h3>{emptyState.title}</h3>
        <p>{emptyState.description}</p>
      </div>
      {emptyState.commands && emptyState.commands.length > 0 ? (
        <div className="command-list">
          {emptyState.commands.map((command) => (
            <div className="command-chip" key={command}>
              <code>{command}</code>
              <CopyButton ariaLabel="Copy command" title="Copy command" value={command} />
            </div>
          ))}
        </div>
      ) : null}
    </section>
  );
}
