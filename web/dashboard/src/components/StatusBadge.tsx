export function StatusBadge({ tone, text }: { tone: string; text: string }) {
  return <span className={`status-badge status-${tone}`}>{text}</span>;
}
