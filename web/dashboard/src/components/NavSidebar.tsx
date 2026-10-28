import { useRef } from "react";
import type { AppSection } from "@/lib/types";
import type { Theme } from "@/lib/theme";

const items: Array<{ key: AppSection; label: string }> = [
  { key: "servers", label: "Servers" },
  { key: "tools", label: "Tools" },
  { key: "tool_groups", label: "Tool Groups" },
  { key: "prompts", label: "Prompts" },
  { key: "resources", label: "Resources" },
  { key: "diagnostics", label: "System Info" },
];

function SunIcon() {
  return (
    <svg aria-hidden="true" fill="none" height="18" viewBox="0 0 16 16" width="18">
      <circle cx="8" cy="8" r="3.5" stroke="currentColor" strokeWidth="1.5" />
      <path
        d="M8 2v1M8 13v1M2 8h1M13 8h1M3.75 3.75l.7.7M11.55 11.55l.7.7M3.75 12.25l.7-.7M11.55 4.45l.7-.7"
        stroke="currentColor"
        strokeLinecap="round"
      />
    </svg>
  );
}

function MonitorIcon() {
  return (
    <svg aria-hidden="true" fill="none" height="18" viewBox="0 0 16 16" width="18">
      <rect x="2" y="3" width="12" height="8" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
      <path d="M5.5 13.5h5" stroke="currentColor" strokeLinecap="round" strokeWidth="1.5" />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg aria-hidden="true" fill="none" height="18" viewBox="0 0 16 16" width="18">
      <path
        d="M6.5 2C4.5 3 3 5 3 7.5c0 3 2.5 5.5 5.5 5.5 2.5 0 4.5-1.5 5.5-3.5-.5.2-1 .5-1.5.5-3 0-5.5-2.5-5.5-5.5 0-.5.2-1 .5-1.5z"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
    </svg>
  );
}

const themeOrder: Theme[] = ["light", "system", "dark"];

function ThemePillToggle({ theme, onSelect }: { theme: Theme; onSelect: (theme: Theme) => void }) {
  const buttonRefs = {
    light: useRef<HTMLButtonElement>(null),
    system: useRef<HTMLButtonElement>(null),
    dark: useRef<HTMLButtonElement>(null),
  };

  function handleKeyDown(e: React.KeyboardEvent) {
    const index = themeOrder.indexOf(theme);
    let next: Theme | null = null;

    if (e.key === "ArrowRight" || e.key === "ArrowDown") {
      e.preventDefault();
      next = themeOrder[(index + 1) % themeOrder.length];
    } else if (e.key === "ArrowLeft" || e.key === "ArrowUp") {
      e.preventDefault();
      next = themeOrder[(index - 1 + themeOrder.length) % themeOrder.length];
    }

    if (next) {
      onSelect(next);
      // Move DOM focus to the newly selected button (roving tabindex pattern)
      buttonRefs[next].current?.focus();
    }
  }

  return (
    <div
      className={`theme-pill-toggle theme-pill-toggle-${theme}`}
      role="radiogroup"
      aria-label="Theme selection"
      onKeyDown={handleKeyDown}
    >
      <span className="theme-pill-toggle-knob" aria-hidden="true" />
      <button
        ref={buttonRefs.light}
        type="button"
        className="theme-pill-toggle-option"
        onClick={() => onSelect("light")}
        aria-label="Light mode"
        role="radio"
        aria-checked={theme === "light"}
        tabIndex={theme === "light" ? 0 : -1}
      >
        <SunIcon />
      </button>
      <button
        ref={buttonRefs.system}
        type="button"
        className="theme-pill-toggle-option"
        onClick={() => onSelect("system")}
        aria-label="System theme"
        role="radio"
        aria-checked={theme === "system"}
        tabIndex={theme === "system" ? 0 : -1}
      >
        <MonitorIcon />
      </button>
      <button
        ref={buttonRefs.dark}
        type="button"
        className="theme-pill-toggle-option"
        onClick={() => onSelect("dark")}
        aria-label="Dark mode"
        role="radio"
        aria-checked={theme === "dark"}
        tabIndex={theme === "dark" ? 0 : -1}
      >
        <MoonIcon />
      </button>
    </div>
  );
}

export function NavSidebar({
  active,
  onSelect,
  logoUrl,
  theme,
  onThemeSelect,
}: {
  active: AppSection;
  onSelect: (section: AppSection) => void;
  logoUrl: string;
  theme: Theme;
  onThemeSelect: (theme: Theme) => void;
}) {
  return (
    <aside className="sidebar">
      <div className="brand-lockup">
        <img alt="MCPJungle logo" className="brand-logo" src={logoUrl} />
        <div className="brand-title-row">
          <p className="brand-title">MCPJungle</p>
          <span className="brand-beta" title="Dashboard frontend is currently in Beta">
            Beta
          </span>
        </div>
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
      <div className="sidebar-footer">
        <div className="theme-row">
          <span className="theme-row-label">Theme</span>
          <ThemePillToggle theme={theme} onSelect={onThemeSelect} />
        </div>
      </div>
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
      <a
        aria-label="Open MCPJungle documentation"
        className="sidebar-link"
        href="https://docs.mcpjungle.com/"
        rel="noopener noreferrer"
        target="_blank"
        title="Open MCPJungle documentation"
      >
        <svg aria-hidden="true" fill="none" height="16" viewBox="0 0 16 16" width="16">
          <path
            d="M4 2.75h6.25A1.75 1.75 0 0 1 12 4.5v8.25a.5.5 0 0 1-.78.41A3.25 3.25 0 0 0 9.5 12.5H4.75A1.75 1.75 0 0 1 3 10.75V3.75A1 1 0 0 1 4 2.75Z"
            stroke="currentColor"
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="1.2"
          />
          <path
            d="M5.25 5h4.5M5.25 7h4.5M5.25 9h2.75"
            stroke="currentColor"
            strokeLinecap="round"
            strokeWidth="1.2"
          />
        </svg>
        <span>Documentation</span>
      </a>
    </aside>
  );
}
