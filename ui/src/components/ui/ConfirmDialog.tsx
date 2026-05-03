type ConfirmDialogProps = {
  title: string;
  message: string;
  confirmLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({
  title,
  message,
  confirmLabel = "Confirm",
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/70 px-4">
      <div className="w-full max-w-md rounded-panel border border-line bg-panel p-6 shadow-panel">
        <h2 className="text-xl font-semibold text-body">{title}</h2>
        <p className="mt-3 text-sm leading-6 text-muted">{message}</p>
        <div className="mt-6 flex gap-3">
          <button className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink" onClick={onConfirm}>
            {confirmLabel}
          </button>
          <button className="rounded-md border border-line px-4 py-2 text-sm text-body" onClick={onCancel}>
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
