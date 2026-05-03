import clsx from "clsx";
import { NavLink } from "react-router-dom";

import { useAppContext } from "../../App";

type NavItem = { to: string; label: string; icon: React.ReactNode };

function Icon({ d, d2 }: { d: string; d2?: string }) {
  return (
    <svg className="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.75} strokeLinecap="round" strokeLinejoin="round">
      <path d={d} />
      {d2 ? <path d={d2} /> : null}
    </svg>
  );
}

const baseItems: NavItem[] = [
  {
    to: "/",
    label: "Overview",
    icon: <Icon d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />,
  },
  {
    to: "/servers",
    label: "Servers",
    icon: <Icon d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />,
  },
  {
    to: "/tools",
    label: "Tools",
    icon: <Icon d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" d2="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />,
  },
];

const adminItems: NavItem[] = [
  {
    to: "/tool-groups",
    label: "Tool Groups",
    icon: <Icon d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />,
  },
  {
    to: "/clients",
    label: "Clients",
    icon: <Icon d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />,
  },
  {
    to: "/users",
    label: "Users",
    icon: <Icon d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />,
  },
  {
    to: "/groups",
    label: "Groups",
    icon: <Icon d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />,
  },
];

const systemItems: NavItem[] = [
  {
    to: "/settings",
    label: "Settings",
    icon: <Icon d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />,
  },
];

export function NavSidebar() {
  const { isAdminEquivalent, settings, user } = useAppContext();

  return (
    <aside className="flex h-full flex-col rounded-panel border border-line bg-[#0f1319] p-4">
      <div className="px-2 py-3">
        <div className="flex items-center gap-2.5">
          <img src="/ui/logo.svg" alt="MCPJungle" className="h-8 w-8 shrink-0" />
          <div>
            <p className="text-sm font-semibold leading-none text-body">MCPJungle</p>
            <p className="mt-1 text-[11px] text-muted">
              {settings.mode === "development" ? "dev mode" : user?.username ?? "gateway"}
            </p>
          </div>
        </div>
      </div>

      <nav className="mt-4 flex flex-col gap-0.5">
        {baseItems.map((item) => (
          <SidebarLink key={item.to} {...item} />
        ))}

        {isAdminEquivalent ? (
          <>
            <div className="px-2 pb-1.5 pt-5 text-[10px] font-medium uppercase tracking-[0.2em] text-muted/60">Admin</div>
            {adminItems.map((item) => (
              <SidebarLink key={item.to} {...item} />
            ))}
          </>
        ) : null}

        <div className="px-2 pb-1.5 pt-5 text-[10px] font-medium uppercase tracking-[0.2em] text-muted/60">System</div>
        {systemItems.map((item) => (
          <SidebarLink key={item.to} {...item} />
        ))}
      </nav>
    </aside>
  );
}

function SidebarLink({ to, label, icon }: NavItem) {
  return (
    <NavLink
      to={to}
      end={to === "/"}
      className={({ isActive }) =>
        clsx(
          "flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors",
          isActive
            ? "bg-accent text-ink"
            : "text-muted hover:bg-panel hover:text-body",
        )
      }
    >
      {icon}
      <span>{label}</span>
    </NavLink>
  );
}
