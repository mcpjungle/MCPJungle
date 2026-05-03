type ErrorStateProps = {
  title: string;
  message: string;
  onRetry?: () => void;
};

export function ErrorState({ title, message, onRetry }: ErrorStateProps) {
  return (
    <div className="grid min-h-screen place-items-center bg-shell px-6 text-body">
      <div className="w-full max-w-xl rounded-panel border border-line bg-panel p-8">
        <p className="text-xs uppercase tracking-[0.24em] text-down">Failure state</p>
        <h1 className="mt-3 text-3xl font-semibold text-body">{title}</h1>
        <p className="mt-4 text-sm leading-6 text-muted">{message}</p>
        {onRetry ? (
          <button className="mt-6 rounded-md bg-accent px-4 py-3 text-sm font-semibold text-ink" onClick={onRetry}>
            Retry
          </button>
        ) : null}
      </div>
    </div>
  );
}
