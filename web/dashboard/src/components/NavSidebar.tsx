import type { AppSection } from "@/lib/types";

const items: Array<{ key: AppSection; label: string }> = [
  { key: "servers", label: "Servers" },
  { key: "tools", label: "Tools" },
  { key: "prompts", label: "Prompts" },
  { key: "resources", label: "Resources" },
  { key: "diagnostics", label: "Diagnostics" },
];

export function NavSidebar({
  active,
  onSelect,
  logoUrl,
}: {
  active: AppSection;
  onSelect: (section: AppSection) => void;
  logoUrl: string;
}) {
  return (
    <aside className="sidebar">
      <div className="brand-lockup">
        <img alt="MCPJungle logo" className="brand-logo" src={logoUrl} />
        <p className="brand-title">MCPJungle</p>
      </div>
      <nav className="nav-list" aria-label="Dashboard sections">
        {items.map((item) => (
          <button
            className={`nav-item ${active === item.key ? "is-active" : ""}`}
            key={item.key}
            onClick={() => onSelect(item.key)}
            type="button"
          >
            {item.label}
          </button>
        ))}
      </nav>
      <a
        className="sidebar-link"
        href="https://github.com/mcpjungle/MCPJungle/issues"
        rel="noopener noreferrer"
        target="_blank"
      >
        <svg aria-hidden="true" fill="none" height="16" viewBox="0 0 16 16" width="16">
          <path
            d="M8 2.25a2 2 0 0 0-2 2v.6a3.5 3.5 0 0 0-1.75 3.03v.62l-.94.94a.75.75 0 0 0 .53 1.28h8.32a.75.75 0 0 0 .53-1.28l-.94-.94v-.62A3.5 3.5 0 0 0 10 4.85v-.6a2 2 0 0 0-2-2Z"
            stroke="currentColor"
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="1.2"
          />
          <path
            d="M6.5 11.75a1.5 1.5 0 0 0 3 0"
            stroke="currentColor"
            strokeLinecap="round"
            strokeWidth="1.2"
          />
        </svg>
        <span>Report Bugs</span>
      </a>
    </aside>
  );
}
