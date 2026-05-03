import type {
  CreateOrUpdateUserResponse,
  CurrentUser,
  McpClient,
  McpClientWithToken,
  McpServer,
  MetadataResponse,
  RegisterServerInput,
  SettingsResponse,
  Tool,
  ToolGroup,
  ToolInvokeResult,
  UpdateClientInput,
  UpdateUserInput,
} from "./types";

type ApiOptions = {
  method?: string;
  token?: string;
  body?: unknown;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function readPayload(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) {
    return null;
  }

  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

export async function request<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const headers: Record<string, string> = {};
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`;
  }

  const response = await fetch(path, {
    method: options.method ?? "GET",
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  });

  if (!response.ok) {
    const payload = await readPayload(response);
    const message =
      typeof payload === "object" && payload !== null && "error" in payload
        ? String((payload as { error: unknown }).error)
        : typeof payload === "string"
          ? payload
          : `Request failed with status ${response.status}`;
    throw new ApiError(message, response.status);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export const api = {
  metadata: () => request<MetadataResponse>("/metadata"),
  settings: (token?: string) => request<SettingsResponse>("/api/v0/settings", { token }),
  whoAmI: (token: string) => request<CurrentUser>("/api/v0/users/whoami", { token }),
  servers: (token?: string) => request<McpServer[]>("/api/v0/servers", { token }),
  serverConfigs: (token?: string) => request<RegisterServerInput[]>("/api/v0/server_configs", { token }),
  createServer: (input: RegisterServerInput, token?: string, force = false) =>
    request<McpServer>(`/api/v0/servers${force ? "?force=true" : ""}`, {
      method: "POST",
      token,
      body: input,
    }),
  enableServer: (name: string, token?: string) =>
    request(`/api/v0/servers/${encodeURIComponent(name)}/enable`, { method: "POST", token }),
  disableServer: (name: string, token?: string) =>
    request(`/api/v0/servers/${encodeURIComponent(name)}/disable`, { method: "POST", token }),
  deleteServer: (name: string, token?: string) =>
    request(`/api/v0/servers/${encodeURIComponent(name)}`, { method: "DELETE", token }),
  tools: (token?: string, server?: string) =>
    request<Tool[]>(`/api/v0/tools${server ? `?server=${encodeURIComponent(server)}` : ""}`, { token }),
  invokeTool: (token: string | undefined, body: Record<string, unknown>) =>
    request<ToolInvokeResult>("/api/v0/tools/invoke", { method: "POST", token, body }),
  enableTool: (entity: string, token?: string) =>
    request(`/api/v0/tools/enable?entity=${encodeURIComponent(entity)}`, { method: "POST", token }),
  disableTool: (entity: string, token?: string) =>
    request(`/api/v0/tools/disable?entity=${encodeURIComponent(entity)}`, { method: "POST", token }),
  toolGroups: (token?: string) => request<ToolGroup[]>("/api/v0/tool-groups", { token }),
  getToolGroup: (name: string, token?: string) =>
    request<ToolGroup>(`/api/v0/tool-groups/${encodeURIComponent(name)}`, { token }),
  getToolGroupEffectiveTools: (name: string, token?: string) =>
    request<string[]>(`/api/v0/tool-groups/${encodeURIComponent(name)}/effective-tools`, { token }),
  createToolGroup: (body: Omit<ToolGroup, "name"> & { name: string }, token?: string) =>
    request<ToolGroup>("/api/v0/tool-groups", { method: "POST", token, body }),
  updateToolGroup: (name: string, body: Partial<Omit<ToolGroup, "name">>, token?: string) =>
    request<ToolGroup>(`/api/v0/tool-groups/${encodeURIComponent(name)}`, { method: "PUT", token, body }),
  deleteToolGroup: (name: string, token?: string) =>
    request(`/api/v0/tool-groups/${encodeURIComponent(name)}`, { method: "DELETE", token }),
  clients: (token?: string) => request<McpClient[]>("/api/v0/clients", { token }),
  createClient: (body: Pick<McpClient, "name" | "description" | "allow_list"> & { access_token?: string }, token?: string) =>
    request<McpClientWithToken>("/api/v0/clients", { method: "POST", token, body }),
  createSelfClient: (name: string, token?: string) =>
    request<McpClientWithToken>("/api/v0/clients/self", { method: "POST", token, body: { name } }),
  applyClientConfig: (mcpToken: string, targets: string[], userToken?: string) =>
    request<{ output: string }>("/api/v0/clients/self/apply-config", {
      method: "POST", token: userToken,
      body: { token: mcpToken, targets, host: window.location.origin },
    }),
  updateClient: (name: string, input: UpdateClientInput, token?: string) =>
    request<McpClientWithToken>(`/api/v0/clients/${encodeURIComponent(name)}`, { method: "PUT", token, body: input }),
  deleteClient: (name: string, token?: string) =>
    request(`/api/v0/clients/${encodeURIComponent(name)}`, { method: "DELETE", token }),
  users: (token?: string) => request<CurrentUser[]>("/api/v0/users", { token }),
  createUser: (body: { username: string; access_token?: string; allow_list?: string[] }, token?: string) =>
    request<CreateOrUpdateUserResponse>("/api/v0/users", { method: "POST", token, body }),
  updateUser: (username: string, input: UpdateUserInput, token?: string) =>
    request<CreateOrUpdateUserResponse>(`/api/v0/users/${encodeURIComponent(username)}`, {
      method: "PUT",
      token,
      body: input,
    }),
  deleteUser: (username: string, token?: string) =>
    request(`/api/v0/users/${encodeURIComponent(username)}`, { method: "DELETE", token }),
};
