type EmptyStateProps = {
  title: string;
  message: string;
};

export function EmptyState({ title, message }: EmptyStateProps) {
  return (
    <div className="rounded-panel border border-dashed border-line bg-shell/60 p-8 text-center">
      <h2 className="text-lg font-semibold text-body">{title}</h2>
      <p className="mt-3 text-sm leading-6 text-muted">{message}</p>
    </div>
  );
}
