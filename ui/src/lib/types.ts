export type ServerMode = "development" | "enterprise" | "production";

export type HealthResponse = {
  status: string;
};

export type MetadataResponse = {
  version: string;
};

export type SettingsResponse = {
  initialized: boolean;
  mode: ServerMode;
  version: string;
};

export type CurrentUser = {
  username: string;
  role: string;
  allow_list?: string[];
  group_id?: number;
  group_name?: string;
};

export type Group = {
  name: string;
  description: string;
  allow_list: string[];
  member_count: number;
};

export type BootstrapReadyState = {
  stage: "ready";
  health: HealthResponse;
  metadata: MetadataResponse;
  settings: SettingsResponse;
  user: CurrentUser | null;
};

export type BootstrapState =
  | { stage: "loading" }
  | { stage: "init"; health: HealthResponse; metadata: MetadataResponse }
  | { stage: "token"; health: HealthResponse; metadata: MetadataResponse; message: string }
  | BootstrapReadyState
  | { stage: "error"; message: string };

export type McpServer = {
  name: string;
  transport: string;
  description: string;
  url?: string;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  session_mode: string;
};

export type RegisterServerInput = {
  name: string;
  transport: string;
  description: string;
  url?: string;
  bearer_token?: string;
  headers?: Record<string, string>;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  session_mode?: string;
};

export type ToolInputSchema = {
  type: string;
  properties?: Record<string, { type?: string; description?: string; default?: unknown }>;
  required?: string[];
};

export type Tool = {
  name: string;
  enabled: boolean;
  description: string;
  input_schema: ToolInputSchema;
};

export type ToolInvokeResult = {
  isError?: boolean;
  content: Array<Record<string, unknown>>;
  structuredContent?: unknown;
};

export type ToolGroup = {
  name: string;
  description: string;
  included_tools?: string[];
  included_servers?: string[];
  excluded_tools?: string[];
};

export type McpClient = {
  name: string;
  description: string;
  access_token?: string;
  allow_list: string[];
  owner_username?: string;
};

export type McpClientWithToken = McpClient & { access_token: string };

export type UpdateClientInput = {
  description: string;
  allow_list: string[];
  access_token?: string;
  rotate_access_token?: boolean;
};

export type UpdateUserInput = {
  access_token?: string;
  rotate_access_token?: boolean;
  allow_list?: string[] | null; // null = clear override (revert to group), [] = no access, [...] = explicit
  update_allow_list?: boolean;
};

export type CreateOrUpdateUserResponse = {
  username: string;
  role: string;
  access_token: string;
  allow_list?: string[];
};
