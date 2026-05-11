import type {
  DashboardDiagnosticsResponse,
  DashboardOverviewResponse,
  DashboardPromptsResponse,
  DashboardRegisterServerInput,
  DashboardResourcesResponse,
  DashboardServersResponse,
  DashboardToolsResponse,
} from "./types";

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.headers ?? {}),
    },
  });
  if (!response.ok) {
    let message = `Request failed: ${response.status}`;
    try {
      const payload = (await response.json()) as { error?: string };
      if (payload.error) {
        message = payload.error;
      }
    } catch {
      // keep the fallback message
    }
    throw new Error(message);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

export const api = {
  overview: () => requestJSON<DashboardOverviewResponse>("/api/dashboard/overview"),
  servers: () => requestJSON<DashboardServersResponse>("/api/dashboard/servers"),
  tools: () => requestJSON<DashboardToolsResponse>("/api/dashboard/tools"),
  prompts: () => requestJSON<DashboardPromptsResponse>("/api/dashboard/prompts"),
  resources: () => requestJSON<DashboardResourcesResponse>("/api/dashboard/resources"),
  diagnostics: () => requestJSON<DashboardDiagnosticsResponse>("/api/dashboard/diagnostics"),
  registerServer: (body: DashboardRegisterServerInput) =>
    requestJSON("/api/dashboard/servers", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  deleteServer: (name: string) =>
    requestJSON(`/api/dashboard/servers/${encodeURIComponent(name)}`, {
      method: "DELETE",
    }),
  setServerEnabled: (name: string, enabled: boolean) =>
    requestJSON(`/api/dashboard/servers/${encodeURIComponent(name)}/enabled`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    }),
  setToolEnabled: (name: string, enabled: boolean) =>
    requestJSON(`/api/dashboard/tools/${encodeURIComponent(name)}/enabled`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    }),
  setPromptEnabled: (name: string, enabled: boolean) =>
    requestJSON(`/api/dashboard/prompts/${encodeURIComponent(name)}/enabled`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    }),
};
