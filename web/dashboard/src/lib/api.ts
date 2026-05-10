import type {
  DashboardDiagnosticsResponse,
  DashboardOverviewResponse,
  DashboardPromptsResponse,
  DashboardResourcesResponse,
  DashboardServersResponse,
  DashboardToolsResponse,
} from "./types";

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path, {
    headers: { Accept: "application/json" },
  });
  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }
  return (await response.json()) as T;
}

export const api = {
  overview: () => getJSON<DashboardOverviewResponse>("/api/dashboard/overview"),
  servers: () => getJSON<DashboardServersResponse>("/api/dashboard/servers"),
  tools: () => getJSON<DashboardToolsResponse>("/api/dashboard/tools"),
  prompts: () => getJSON<DashboardPromptsResponse>("/api/dashboard/prompts"),
  resources: () => getJSON<DashboardResourcesResponse>("/api/dashboard/resources"),
  diagnostics: () => getJSON<DashboardDiagnosticsResponse>("/api/dashboard/diagnostics"),
};
