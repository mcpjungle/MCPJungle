import { createBrowserRouter } from "react-router-dom";

import { AppShell } from "../components/layout/AppShell";
import { ClientsPage } from "../pages/ClientsPage";
import { DashboardPage } from "../pages/DashboardPage";
import { ServerEditorPage } from "../pages/ServerEditorPage";
import { ServersPage } from "../pages/ServersPage";
import { SettingsPage } from "../pages/SettingsPage";
import { ToolGroupsPage } from "../pages/ToolGroupsPage";
import { ToolsPage } from "../pages/ToolsPage";
import { UsersPage } from "../pages/UsersPage";

export function createAppRouter() {
  return createBrowserRouter(
    [
      {
        path: "/",
        element: <AppShell />,
        children: [
          { index: true, element: <DashboardPage /> },
          { path: "servers", element: <ServersPage /> },
          { path: "servers/new", element: <ServerEditorPage /> },
          { path: "servers/:name/edit", element: <ServerEditorPage /> },
          { path: "tools", element: <ToolsPage /> },
          { path: "tool-groups", element: <ToolGroupsPage /> },
          { path: "clients", element: <ClientsPage /> },
          { path: "users", element: <UsersPage /> },
          { path: "settings", element: <SettingsPage /> },
        ],
      },
    ],
    { basename: "/ui" },
  );
}
