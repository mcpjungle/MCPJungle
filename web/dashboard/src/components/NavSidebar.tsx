import type { AppSection } from "@/lib/types";

const items: Array<{ key: AppSection; label: string }> = [
  { key: "overview", label: "Overview" },
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
    </aside>
  );
}
