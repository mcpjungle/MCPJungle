import { useState } from "react";

function CopyIcon({ copied }: { copied: boolean }) {
  if (copied) {
    return (
      <svg aria-hidden="true" fill="none" height="18" viewBox="0 0 16 16" width="18">
        <path
          d="M3.75 8.5 6.25 11l6-6"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.5"
        />
      </svg>
    );
  }

  return (
    <svg aria-hidden="true" fill="none" height="18" viewBox="0 0 16 16" width="18">
      <rect
        height="8.5"
        rx="1.5"
        stroke="currentColor"
        strokeWidth="1.25"
        width="7.5"
        x="5"
        y="3"
      />
      <path
        d="M3.5 9.5h-.25A1.25 1.25 0 0 1 2 8.25v-5A1.25 1.25 0 0 1 3.25 2h5A1.25 1.25 0 0 1 9.5 3.25v.25"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="1.25"
      />
    </svg>
  );
}

export function CopyButton({
  value,
  ariaLabel = "Copy",
  title,
}: {
  value: string;
  ariaLabel?: string;
  title?: string;
}) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  }

  return (
    <button
      aria-label={copied ? "Copied" : ariaLabel}
      className="copy-button icon-button"
      onClick={handleCopy}
      title={copied ? "Copied" : title ?? ariaLabel}
      type="button"
    >
      <CopyIcon copied={copied} />
    </button>
  );
}
