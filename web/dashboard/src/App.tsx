import { Fragment, useEffect, useMemo, useState } from "react";
import logoUrl from "@repo-assets/logo.png";
import { api } from "@/lib/api";
import type {
  AppSection,
  DashboardCreateToolGroupInput,
  DashboardDiagnosticsResponse,
  DashboardOAuthAuthorizationRequired,
  DashboardOverviewResponse,
  DashboardPrompt,
  DashboardPromptsResponse,
  DashboardRegisterServerInput,
  DashboardResource,
  DashboardResourcesResponse,
  DashboardServer,
  DashboardServersResponse,
  DashboardToolGroup,
  DashboardToolGroupsResponse,
  DashboardTool,
  DashboardToolsResponse,
} from "@/lib/types";
import { CopyButton } from "@/components/CopyButton";
import { EmptyStateCard } from "@/components/EmptyStateCard";
import { NavSidebar } from "@/components/NavSidebar";
import { SectionCard } from "@/components/SectionCard";
import { StatusBadge } from "@/components/StatusBadge";

function TrashIcon() {
  return (
    <svg aria-hidden="true" fill="none" height="18" viewBox="0 0 16 16" width="18">
      <path
        d="M2.75 4.25h10.5M6.25 2.75h3.5m-5.75 1.5.44 7.04A1.5 1.5 0 0 0 5.94 12.75h4.12a1.5 1.5 0 0 0 1.5-1.46L12 4.25"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
      <path d="M6.5 6.5v3.5M9.5 6.5v3.5" stroke="currentColor" strokeLinecap="round" strokeWidth="1.5" />
    </svg>
  );
}

type LoadState = "idle" | "loading" | "ready" | "error";
type FeedbackTone = "success" | "error";

interface DashboardData {
  overview?: DashboardOverviewResponse;
  servers?: DashboardServersResponse;
  tools?: DashboardToolsResponse;
  toolGroups?: DashboardToolGroupsResponse;
  prompts?: DashboardPromptsResponse;
  resources?: DashboardResourcesResponse;
  diagnostics?: DashboardDiagnosticsResponse;
}

interface FeedbackMessage {
  tone: FeedbackTone;
  message: string;
}

interface KeyValueRow {
  key: string;
  value: string;
}

interface RegisterServerFormState {
  name: string;
  description: string;
  transport: "stdio" | "streamable_http" | "sse";
  session_mode: "stateless" | "stateful";
  command: string;
  args_text: string;
  env_rows: KeyValueRow[];
  url: string;
  bearer_token: string;
  header_rows: KeyValueRow[];
}

interface RegisterOAuthState {
  authorization: DashboardOAuthAuthorizationRequired;
  hasOpenedBrowser: boolean;
  error: string;
}

interface ToolGroupFormState {
  name: string;
  description: string;
  selectedTools: string[];
}

interface SchemaFieldSummary {
  path: string;
  type: string;
  required: boolean;
  description?: string;
  enumValues?: string[];
  defaultValue?: string;
  note?: string;
}

const sectionMeta: Record<AppSection, { title: string; subtitle: string }> = {
  servers: {
    title: "Servers",
    subtitle: "",
  },
  tools: {
    title: "Tools",
    subtitle: "All discovered tools across registered servers.",
  },
  tool_groups: {
    title: "Tool Groups",
    subtitle: "",
  },
  prompts: {
    title: "Prompts",
    subtitle: "Prompt templates currently exposed through MCPJungle.",
  },
  resources: {
    title: "Resources",
    subtitle: "Resources registered and proxied through the gateway.",
  },
  diagnostics: {
    title: "System Info",
    subtitle: "",
  },
};

function shortVersion(version?: string) {
  if (!version) {
    return "";
  }
  const match = version.match(/v?\d+\.\d+\.\d+/);
  if (match) {
    return match[0];
  }
  return version.length > 16 ? version.slice(0, 16) : version;
}

function transportLabel(value?: string) {
  return value ? value.split("_").join(" ") : "unknown";
}

function toolDescription(tool: DashboardTool) {
  return tool.description || "No description";
}

function promptDescription(prompt: DashboardPrompt) {
  return prompt.description || "No description";
}

function resourceDescription(resource: DashboardResource) {
  return resource.description || "No description";
}

function prettyJSON(value?: Record<string, unknown>) {
  if (!value) {
    return "No schema available.";
  }
  return JSON.stringify(value, null, 2);
}

function prettyPromptArguments(value?: Array<Record<string, unknown>>) {
  if (!value || value.length === 0) {
    return "No arguments";
  }
  return JSON.stringify(value, null, 2);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function schemaTypeLabel(schema: Record<string, unknown>) {
  const type = schema.type;
  if (typeof type === "string") {
    return type;
  }
  if (Array.isArray(type) && type.every((item) => typeof item === "string")) {
    return type.join(" | ");
  }
  if (isRecord(schema.properties)) {
    return "object";
  }
  if (schema.items) {
    return "array";
  }
  if (Array.isArray(schema.enum) && schema.enum.length > 0) {
    return "enum";
  }
  return "unknown";
}

function formatSchemaValue(value: unknown) {
  if (value === undefined) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  return JSON.stringify(value);
}

function schemaNote(schema: Record<string, unknown>) {
  const notes: string[] = [];
  if (Array.isArray(schema.oneOf) && schema.oneOf.length > 0) {
    notes.push(`${schema.oneOf.length} oneOf variants`);
  }
  if (Array.isArray(schema.anyOf) && schema.anyOf.length > 0) {
    notes.push(`${schema.anyOf.length} anyOf variants`);
  }
  if (schema.additionalProperties === true) {
    notes.push("additional properties allowed");
  }
  return notes.join(", ");
}

function collectSchemaFields(
  schema: Record<string, unknown>,
  path: string,
  required: boolean,
  fields: SchemaFieldSummary[],
) {
  const entry: SchemaFieldSummary = {
    path,
    type: schemaTypeLabel(schema),
    required,
  };

  if (typeof schema.description === "string" && schema.description.trim()) {
    entry.description = schema.description;
  }
  if (Array.isArray(schema.enum) && schema.enum.length > 0) {
    entry.enumValues = schema.enum.map((value) => formatSchemaValue(value));
  }
  if (schema.default !== undefined) {
    entry.defaultValue = formatSchemaValue(schema.default);
  }
  const note = schemaNote(schema);
  if (note) {
    entry.note = note;
  }
  fields.push(entry);

  if (isRecord(schema.properties)) {
    const requiredFields = new Set(
      Array.isArray(schema.required) ? schema.required.filter((value): value is string => typeof value === "string") : [],
    );
    Object.entries(schema.properties).forEach(([key, value]) => {
      if (!isRecord(value)) {
        return;
      }
      const childPath = path ? `${path}.${key}` : key;
      collectSchemaFields(value, childPath, requiredFields.has(key), fields);
    });
  }

  if (schema.items && isRecord(schema.items)) {
    collectSchemaFields(schema.items, `${path}[]`, true, fields);
  }
}

function parseToolSchemaFields(schema?: Record<string, unknown>) {
  if (!schema) {
    return [] as SchemaFieldSummary[];
  }

  const fields: SchemaFieldSummary[] = [];
  if (isRecord(schema.properties)) {
    const requiredFields = new Set(
      Array.isArray(schema.required) ? schema.required.filter((value): value is string => typeof value === "string") : [],
    );
    Object.entries(schema.properties).forEach(([key, value]) => {
      if (!isRecord(value)) {
        return;
      }
      collectSchemaFields(value, key, requiredFields.has(key), fields);
    });
    return fields;
  }

  collectSchemaFields(schema, "(root)", true, fields);
  return fields;
}

function parsePromptArgumentFields(argumentsValue?: Array<Record<string, unknown>>) {
  if (!argumentsValue || argumentsValue.length === 0) {
    return [] as SchemaFieldSummary[];
  }

  const fields: SchemaFieldSummary[] = [];

  argumentsValue.forEach((argument, index) => {
    const name =
      (typeof argument.name === "string" && argument.name) ||
      (typeof argument.title === "string" && argument.title) ||
      `arg${index + 1}`;

    const entry: SchemaFieldSummary = {
      path: name,
      // Prompt arguments are string-like by default unless the backend explicitly provides a schema type.
      type: (() => {
        const explicitType = schemaTypeLabel(argument);
        return explicitType === "unknown" ? "string" : explicitType;
      })(),
      required: Boolean(argument.required),
    };

    if (typeof argument.description === "string" && argument.description.trim()) {
      entry.description = argument.description;
    }
    if (Array.isArray(argument.enum) && argument.enum.length > 0) {
      entry.enumValues = argument.enum.map((value) => formatSchemaValue(value));
    }
    if (argument.default !== undefined) {
      entry.defaultValue = formatSchemaValue(argument.default);
    }
    const note = schemaNote(argument);
    if (note) {
      entry.note = note;
    }
    fields.push(entry);

    if (isRecord(argument.properties)) {
      const requiredFields = new Set(
        Array.isArray(argument.required)
          ? argument.required.filter((value): value is string => typeof value === "string")
          : [],
      );
      Object.entries(argument.properties).forEach(([key, value]) => {
        if (!isRecord(value)) {
          return;
        }
        collectSchemaFields(value, `${name}.${key}`, requiredFields.has(key), fields);
      });
    }

    if (argument.items && isRecord(argument.items)) {
      collectSchemaFields(argument.items, `${name}[]`, true, fields);
    }
  });

  return fields;
}

function ChevronIcon({ expanded }: { expanded: boolean }) {
  return (
    <svg
      aria-hidden="true"
      className={`row-chevron ${expanded ? "is-expanded" : ""}`}
      fill="none"
      height="16"
      viewBox="0 0 16 16"
      width="16"
    >
      <path
        d="m5.5 3.75 4.25 4.25-4.25 4.25"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.5"
      />
    </svg>
  );
}

function createEmptyPair(): KeyValueRow {
  return { key: "", value: "" };
}

function createInitialRegisterForm(): RegisterServerFormState {
  return {
    name: "",
    description: "",
    transport: "streamable_http",
    session_mode: "stateless",
    command: "",
    args_text: "",
    env_rows: [createEmptyPair()],
    url: "",
    bearer_token: "",
    header_rows: [createEmptyPair()],
  };
}

function rowsToMap(rows: KeyValueRow[]) {
  const output: Record<string, string> = {};
  rows.forEach((row) => {
    const key = row.key.trim();
    if (!key) {
      return;
    }
    output[key] = row.value;
  });
  return output;
}

function splitArgs(input: string) {
  return input
    .split("\n")
    .map((value) => value.trim())
    .filter(Boolean);
}

function getRegisterValidationError(form: RegisterServerFormState) {
  if (!form.name.trim()) {
    return "Server name is required.";
  }
  if (form.transport === "stdio" && !form.command.trim()) {
    return "Command is required for stdio servers.";
  }
  if ((form.transport === "streamable_http" || form.transport === "sse") && !form.url.trim()) {
    return "Target URL is required for HTTP and SSE servers.";
  }
  return "";
}

function buildRegisterPayload(form: RegisterServerFormState): DashboardRegisterServerInput {
  const payload: DashboardRegisterServerInput = {
    name: form.name.trim(),
    description: form.description.trim(),
    transport: form.transport,
    session_mode: form.session_mode,
  };

  if (form.transport === "stdio") {
    payload.command = form.command.trim();
    payload.args = splitArgs(form.args_text);
    const env = rowsToMap(form.env_rows);
    if (Object.keys(env).length > 0) {
      payload.env = env;
    }
    return payload;
  }

  payload.url = form.url.trim();
  if (form.bearer_token.trim()) {
    payload.bearer_token = form.bearer_token.trim();
  }
  if (form.transport === "streamable_http") {
    const headers = rowsToMap(form.header_rows);
    if (Object.keys(headers).length > 0) {
      payload.headers = headers;
    }
  }
  return payload;
}

function createInitialToolGroupForm(): ToolGroupFormState {
  return {
    name: "",
    description: "",
    selectedTools: [],
  };
}

export default function App() {
  const [section, setSection] = useState<AppSection>("servers");
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [errorMessage, setErrorMessage] = useState("");
  const [feedback, setFeedback] = useState<FeedbackMessage | null>(null);
  const [data, setData] = useState<DashboardData>({});
  const [serverFilter, setServerFilter] = useState("");
  const [toolFilter, setToolFilter] = useState("");
  const [toolServerFilter, setToolServerFilter] = useState("all");
  const [promptFilter, setPromptFilter] = useState("");
  const [toolGroupToolFilter, setToolGroupToolFilter] = useState("");
  const [toolGroupToolServerFilter, setToolGroupToolServerFilter] = useState("all");
  const [expandedServer, setExpandedServer] = useState<string | null>(null);
  const [expandedTool, setExpandedTool] = useState<string | null>(null);
  const [expandedToolGroup, setExpandedToolGroup] = useState<string | null>(null);
  const [expandedPrompt, setExpandedPrompt] = useState<string | null>(null);
  const [registerOpen, setRegisterOpen] = useState(false);
  const [registerForm, setRegisterForm] = useState<RegisterServerFormState>(createInitialRegisterForm());
  const [registerError, setRegisterError] = useState("");
  const [registerOAuth, setRegisterOAuth] = useState<RegisterOAuthState | null>(null);
  const [toolGroupOpen, setToolGroupOpen] = useState(false);
  const [toolGroupForm, setToolGroupForm] = useState<ToolGroupFormState>(createInitialToolGroupForm());
  const [toolGroupError, setToolGroupError] = useState("");
  const [busyKeys, setBusyKeys] = useState<Record<string, boolean>>({});

  async function loadDashboardData(silent = false) {
    if (!silent) {
      setLoadState("loading");
    }
    setErrorMessage("");
    try {
      const [overview, servers, tools, toolGroups, prompts, resources, diagnostics] = await Promise.all([
        api.overview(),
        api.servers(),
        api.tools(),
        api.toolGroups(),
        api.prompts(),
        api.resources(),
        api.diagnostics(),
      ]);
      setData({ overview, servers, tools, toolGroups, prompts, resources, diagnostics });
      setExpandedTool((current) =>
        current && tools.tools.some((tool) => tool.canonical_name === current) ? current : null,
      );
      setExpandedToolGroup((current) =>
        current && toolGroups.tool_groups.some((group) => group.name === current) ? current : null,
      );
      setExpandedPrompt((current) =>
        current && prompts.prompts.some((prompt) => prompt.canonical_name === current) ? current : null,
      );
      setLoadState("ready");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unknown error";
      setErrorMessage(message);
      setLoadState("error");
    }
  }

  useEffect(() => {
    void loadDashboardData();
  }, []);

  const filteredServers = useMemo(() => {
    const servers = data.servers?.servers ?? [];
    if (!serverFilter.trim()) {
      return servers;
    }
    const term = serverFilter.toLowerCase();
    return servers.filter(
      (server) =>
        server.name.toLowerCase().includes(term) ||
        server.transport.toLowerCase().includes(term) ||
        server.connection_summary.toLowerCase().includes(term),
    );
  }, [data.servers?.servers, serverFilter]);

  const filteredTools = useMemo(() => {
    let tools = data.tools?.tools ?? [];
    if (toolServerFilter !== "all") {
      tools = tools.filter((tool) => tool.server === toolServerFilter);
    }
    if (!toolFilter.trim()) {
      return tools;
    }
    const term = toolFilter.toLowerCase();
    return tools.filter(
      (tool) =>
        tool.name.toLowerCase().includes(term) ||
        tool.server.toLowerCase().includes(term) ||
        tool.canonical_name.toLowerCase().includes(term) ||
        toolDescription(tool).toLowerCase().includes(term),
    );
  }, [data.tools?.tools, toolFilter, toolServerFilter]);

  const uniqueToolServers = useMemo(() => {
    const servers = new Set((data.tools?.tools ?? []).map((tool) => tool.server));
    return Array.from(servers).sort();
  }, [data.tools?.tools]);

  const availableToolGroupTools = useMemo(() => {
    let tools = data.tools?.tools ?? [];
    if (toolGroupToolServerFilter !== "all") {
      tools = tools.filter((tool) => tool.server === toolGroupToolServerFilter);
    }
    if (!toolGroupToolFilter.trim()) {
      return tools;
    }
    const term = toolGroupToolFilter.toLowerCase();
    return tools.filter(
      (tool) =>
        tool.name.toLowerCase().includes(term) ||
        tool.canonical_name.toLowerCase().includes(term) ||
        tool.server.toLowerCase().includes(term) ||
        toolDescription(tool).toLowerCase().includes(term),
    );
  }, [data.tools?.tools, toolGroupToolFilter, toolGroupToolServerFilter]);

  const filteredPrompts = useMemo(() => {
    const prompts = data.prompts?.prompts ?? [];
    if (!promptFilter.trim()) {
      return prompts;
    }
    const term = promptFilter.toLowerCase();
    return prompts.filter(
      (prompt) =>
        prompt.name.toLowerCase().includes(term) ||
        prompt.canonical_name.toLowerCase().includes(term) ||
        prompt.server.toLowerCase().includes(term) ||
        promptDescription(prompt).toLowerCase().includes(term),
    );
  }, [data.prompts?.prompts, promptFilter]);

  const overview = data.overview;
  const diagnostics = data.diagnostics;
  const currentSectionMeta = sectionMeta[section];

  function setBusy(key: string, value: boolean) {
    setBusyKeys((current) => {
      const next = { ...current };
      if (value) {
        next[key] = true;
      } else {
        delete next[key];
      }
      return next;
    });
  }

  function isBusy(key: string) {
    return Boolean(busyKeys[key]);
  }

  async function runMutation(key: string, action: () => Promise<void>, successMessage: string) {
    setFeedback(null);
    setBusy(key, true);
    try {
      await action();
      await loadDashboardData(true);
      setFeedback({ tone: "success", message: successMessage });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Request failed";
      setFeedback({ tone: "error", message });
      throw error;
    } finally {
      setBusy(key, false);
    }
  }

  function updateRegisterField<K extends keyof RegisterServerFormState>(field: K, value: RegisterServerFormState[K]) {
    setRegisterForm((current) => ({ ...current, [field]: value }));
  }

  function updateKeyValueRow(
    field: "env_rows" | "header_rows",
    index: number,
    key: "key" | "value",
    value: string,
  ) {
    setRegisterForm((current) => {
      const rows = current[field].map((row, rowIndex) =>
        rowIndex === index ? { ...row, [key]: value } : row,
      );
      return { ...current, [field]: rows };
    });
  }

  function addKeyValueRow(field: "env_rows" | "header_rows") {
    setRegisterForm((current) => ({ ...current, [field]: [...current[field], createEmptyPair()] }));
  }

  function removeKeyValueRow(field: "env_rows" | "header_rows", index: number) {
    setRegisterForm((current) => {
      const rows = current[field].filter((_, rowIndex) => rowIndex !== index);
      return { ...current, [field]: rows.length > 0 ? rows : [createEmptyPair()] };
    });
  }

  function openRegisterModal() {
    setRegisterForm(createInitialRegisterForm());
    setRegisterError("");
    setRegisterOAuth(null);
    setRegisterOpen(true);
  }

  function closeRegisterModal() {
    setRegisterOpen(false);
    setRegisterError("");
    setRegisterOAuth(null);
    setRegisterForm(createInitialRegisterForm());
  }

  function resetRegisterOAuthStep(message = "") {
    setRegisterOAuth(null);
    setRegisterError(message);
  }

  function openToolGroupModal() {
    setToolGroupForm(createInitialToolGroupForm());
    setToolGroupError("");
    setToolGroupToolFilter("");
    setToolGroupToolServerFilter("all");
    setToolGroupOpen(true);
  }

  function closeToolGroupModal() {
    setToolGroupOpen(false);
    setToolGroupForm(createInitialToolGroupForm());
    setToolGroupError("");
  }

  function toggleToolGroupSelection(canonicalName: string) {
    setToolGroupForm((current) => ({
      ...current,
      selectedTools: current.selectedTools.includes(canonicalName)
        ? current.selectedTools.filter((name) => name !== canonicalName)
        : [...current.selectedTools, canonicalName],
    }));
  }

  function removeToolGroupSelection(canonicalName: string) {
    setToolGroupForm((current) => ({
      ...current,
      selectedTools: current.selectedTools.filter((name) => name !== canonicalName),
    }));
  }

  async function submitRegisterServer() {
    const validationError = getRegisterValidationError(registerForm);
    if (validationError) {
      setRegisterError(validationError);
      return;
    }

    setRegisterError("");
    try {
      setFeedback(null);
      setBusy("register-server", true);
      const response = await api.registerServer(buildRegisterPayload(registerForm));
      if (response.authorization_required) {
        setRegisterOAuth({
          authorization: response.authorization_required,
          hasOpenedBrowser: false,
          error: "",
        });
        setFeedback(null);
        return;
      }
      await loadDashboardData(true);
      setFeedback({ tone: "success", message: `Server ${registerForm.name.trim()} registered.` });
      closeRegisterModal();
    } catch (error) {
      const message = error instanceof Error ? error.message : "Request failed";
      setRegisterError(message);
      setFeedback({ tone: "error", message });
    } finally {
      setBusy("register-server", false);
    }
  }

  function startRegisterOAuth() {
    if (!registerOAuth) {
      return;
    }
    window.open(registerOAuth.authorization.authorization_url, "_blank", "noopener,noreferrer");
    setRegisterOAuth((current) =>
      current
        ? {
            ...current,
            hasOpenedBrowser: true,
            error: "",
          }
        : current,
    );
  }

  useEffect(() => {
    if (!registerOAuth?.hasOpenedBrowser) {
      return;
    }

    let cancelled = false
    const sessionID = registerOAuth.authorization.session_id

    async function pollOAuthSession() {
      try {
        const response = await api.getOAuthSession(sessionID)
        if (cancelled) {
          return
        }
        if (response.status === "pending") {
          return
        }
        if (response.status === "completed") {
          await loadDashboardData(true)
          if (cancelled) {
            return
          }
          setFeedback({
            tone: "success",
            message: `Server ${response.server_name ?? registerForm.name.trim()} registered.`,
          })
          closeRegisterModal()
          return
        }

        setRegisterOAuth((current) =>
          current
            ? {
                ...current,
                hasOpenedBrowser: false,
                error: response.error || "OAuth authorization could not be completed. Start registration again.",
              }
            : current,
        )
      } catch (error) {
        if (cancelled) {
          return
        }
        const message = error instanceof Error ? error.message : "Failed to check OAuth authorization state."
        setRegisterOAuth((current) =>
          current
            ? {
                ...current,
                hasOpenedBrowser: false,
                error: message,
              }
            : current,
        )
      }
    }

    void pollOAuthSession()
    const timer = window.setInterval(() => {
      void pollOAuthSession()
    }, 2000)

    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [registerOAuth?.authorization.session_id, registerOAuth?.hasOpenedBrowser])

  async function toggleServerEnabled(server: DashboardServer) {
    const nextEnabled = !server.enabled;
    await runMutation(
      `server-toggle:${server.name}`,
      async () => {
        await api.setServerEnabled(server.name, nextEnabled);
      },
      `${server.name} ${nextEnabled ? "enabled" : "disabled"}.`,
    );
  }

  async function deleteServer(server: DashboardServer) {
    const confirmed = window.confirm(
      `Delete server "${server.name}"? This removes the registration and all discovered tools, prompts, and resources from MCPJungle.`,
    );
    if (!confirmed) {
      return;
    }
    await runMutation(
      `server-delete:${server.name}`,
      async () => {
        await api.deleteServer(server.name);
      },
      `${server.name} deleted.`,
    );
    if (expandedServer === server.name) {
      setExpandedServer(null);
    }
  }

  async function toggleToolEnabled(tool: DashboardTool) {
    const nextEnabled = !tool.enabled;
    await runMutation(
      `tool-toggle:${tool.canonical_name}`,
      async () => {
        await api.setToolEnabled(tool.canonical_name, nextEnabled);
      },
      `${tool.canonical_name} ${nextEnabled ? "enabled" : "disabled"}.`,
    );
  }

  async function togglePromptEnabled(prompt: DashboardPrompt) {
    const nextEnabled = !prompt.enabled;
    await runMutation(
      `prompt-toggle:${prompt.canonical_name}`,
      async () => {
        await api.setPromptEnabled(prompt.canonical_name, nextEnabled);
      },
      `${prompt.canonical_name} ${nextEnabled ? "enabled" : "disabled"}.`,
    );
  }

  async function submitToolGroup() {
    const name = toolGroupForm.name.trim();
    if (!name) {
      setToolGroupError("Group name is required.");
      return;
    }
    if (toolGroupForm.selectedTools.length === 0) {
      setToolGroupError("Select at least one tool.");
      return;
    }
    if ((data.toolGroups?.tool_groups ?? []).some((group) => group.name === name)) {
      setToolGroupError("A tool group with that name already exists.");
      return;
    }

    setToolGroupError("");
    setFeedback(null);
    setBusy("tool-group-create", true);
    try {
      const payload: DashboardCreateToolGroupInput = {
        name,
        description: toolGroupForm.description.trim(),
        tools: toolGroupForm.selectedTools,
      };
      await api.createToolGroup(payload);
      await loadDashboardData(true);
      setFeedback({ tone: "success", message: `Tool group ${name} created.` });
      closeToolGroupModal();
      setSection("tool_groups");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Request failed";
      setToolGroupError(message);
      setFeedback({ tone: "error", message });
    } finally {
      setBusy("tool-group-create", false);
    }
  }

  async function deleteToolGroup(group: DashboardToolGroup) {
    const confirmed = window.confirm(`Delete tool group "${group.name}"?`);
    if (!confirmed) {
      return;
    }
    await runMutation(
      `tool-group-delete:${group.name}`,
      async () => {
        await api.deleteToolGroup(group.name);
      },
      `${group.name} deleted.`,
    );
    if (expandedToolGroup === group.name) {
      setExpandedToolGroup(null);
    }
  }

  return (
    <div className="app-shell">
      <NavSidebar active={section} logoUrl={logoUrl} onSelect={setSection} />
      <main className="main-shell">
        <header className="topbar">
          <div>
            <h1>{currentSectionMeta.title}</h1>
            {currentSectionMeta.subtitle ? (
              <p className="topbar-subtitle">{currentSectionMeta.subtitle}</p>
            ) : null}
          </div>
          <div className="topbar-meta">
            {overview?.version ? (
              <span className="version-chip">{`Server version ${shortVersion(overview.version)}`}</span>
            ) : null}
            {overview?.endpoints[0] ? (
              <div className="topbar-endpoint">
                <span className="topbar-endpoint-label">Endpoint</span>
                <code title={overview.endpoints[0].url}>{overview.endpoints[0].url}</code>
                <CopyButton ariaLabel="Copy endpoint" title="Copy endpoint" value={overview.endpoints[0].url} />
              </div>
            ) : null}
          </div>
        </header>

        {feedback ? (
          <section className={`feedback-banner feedback-${feedback.tone}`}>
            <strong>{feedback.tone === "success" ? "Updated" : "Request failed"}</strong>
            <span>{feedback.message}</span>
          </section>
        ) : null}

        {loadState === "loading" ? (
          <section className="loading-screen panel">
            <h2>Loading dashboard</h2>
            <p>Querying local MCPJungle state, servers, tools, prompts, and resources.</p>
          </section>
        ) : null}

        {loadState === "error" ? (
          <section className="loading-screen panel error-screen">
            <h2>Dashboard API unavailable</h2>
            <p>Failed to load dashboard data from the local server.</p>
            <code>{errorMessage}</code>
          </section>
        ) : null}

        {loadState === "ready" ? (
          <div className="content-grid">
            {section === "servers" && data.servers ? (
              <>
                {overview ? (
                  <section className="dense-metrics-grid">
                    <div className="metric-card compact-metric">
                      <span>Servers</span>
                      <strong>{overview.server_count}</strong>
                    </div>
                    <div className="metric-card compact-metric">
                      <span>Tools</span>
                      <strong>{overview.tool_count}</strong>
                    </div>
                    <div className="metric-card compact-metric">
                      <span>Prompts</span>
                      <strong>{overview.prompt_count}</strong>
                    </div>
                    <div className="metric-card compact-metric">
                      <span>Resources</span>
                      <strong>{overview.resource_count}</strong>
                    </div>
                  </section>
                ) : null}

                <SectionCard
                  title="Servers"
                  subtitle="Registered MCP servers"
                  action={
                    <div className="toolbar-cluster">
                      <input
                        className="table-filter compact-filter"
                        onChange={(event) => setServerFilter(event.target.value)}
                        placeholder="Search servers"
                        value={serverFilter}
                      />
                      <button className="primary-action" onClick={openRegisterModal} type="button">
                        + Add Server
                      </button>
                    </div>
                  }
                >
                  {data.servers.empty_state && filteredServers.length === 0 ? (
                    <EmptyStateCard emptyState={data.servers.empty_state} />
                  ) : (
                    <div className="server-list compact-server-list">
                      {filteredServers.map((server) => {
                        const expanded = expandedServer === server.name;
                        return (
                          <article
                            className={`server-row compact-server-row ${
                              server.enabled ? "" : "server-row-disabled"
                            }`}
                            key={server.name}
                          >
                            <div className="server-row-head compact-server-head">
                              <div className="server-row-layout">
                                <button
                                  className="server-expand-button"
                                  onClick={() => setExpandedServer(expanded ? null : server.name)}
                                  type="button"
                                >
                                  <div className="server-head-main">
                                    <h3>{server.name}</h3>
                                    <p>{server.connection_summary}</p>
                                  </div>
                                </button>
                                <div className="server-row-meta compact-server-meta">
                                  <div className="server-meta-cell">
                                    <code>{transportLabel(server.transport)}</code>
                                  </div>
                                  <div className="server-meta-cell">
                                    <StatusBadge
                                      text={server.enabled ? "Enabled" : "Disabled"}
                                      tone={server.enabled ? "good" : "muted"}
                                    />
                                  </div>
                                  <div className="server-meta-cell server-tool-count">
                                    <strong>{server.tool_count} tools</strong>
                                  </div>
                                  <div className="server-meta-cell">
                                    <button
                                      className="secondary-action server-action-button"
                                      disabled={isBusy(`server-toggle:${server.name}`)}
                                      onClick={() => void toggleServerEnabled(server)}
                                      type="button"
                                    >
                                      {isBusy(`server-toggle:${server.name}`)
                                        ? "Saving..."
                                        : server.enabled
                                          ? "Disable"
                                          : "Enable"}
                                    </button>
                                  </div>
                                  <div className="server-meta-cell">
                                    <button
                                      aria-label="Delete server"
                                      className="danger-action server-action-button icon-button danger-icon-button"
                                      disabled={isBusy(`server-delete:${server.name}`)}
                                      onClick={() => void deleteServer(server)}
                                      title="Delete server"
                                      type="button"
                                    >
                                      <TrashIcon />
                                    </button>
                                  </div>
                                </div>
                              </div>
                            </div>
                            {expanded ? (
                              <div className="server-detail">
                                {!server.enabled ? (
                                  <p className="detail-note">
                                    This server is registered but currently not exposed to MCP clients.
                                  </p>
                                ) : null}
                                <dl>
                                  <div>
                                    <dt>Target</dt>
                                    <dd>
                                      <div className="detail-copy-row">
                                        <code className="detail-target-code">
                                          {server.config_summary.target ??
                                            server.config_summary.command ??
                                            "Unknown"}
                                        </code>
                                        {server.config_summary.target ||
                                        server.config_summary.command ? (
                                          <CopyButton
                                            ariaLabel="Copy target"
                                            title="Copy target"
                                            value={
                                              server.config_summary.target ??
                                              server.config_summary.command ??
                                              ""
                                            }
                                          />
                                        ) : null}
                                      </div>
                                    </dd>
                                  </div>
                                  <div>
                                    <dt>Session mode</dt>
                                    <dd>
                                      <code>{server.config_summary.session_mode ?? "Unknown"}</code>
                                    </dd>
                                  </div>
                                  <div>
                                    <dt>Header keys</dt>
                                    <dd>
                                      <code>{server.config_summary.header_keys?.join(", ") || "None"}</code>
                                    </dd>
                                  </div>
                                  <div>
                                    <dt>Env keys</dt>
                                    <dd>
                                      <code>{server.config_summary.env_keys?.join(", ") || "None"}</code>
                                    </dd>
                                  </div>
                                </dl>
                              </div>
                            ) : null}
                          </article>
                        );
                      })}
                    </div>
                  )}
                </SectionCard>
              </>
            ) : null}

            {section === "tools" && data.tools ? (
              <SectionCard
                title="Tools"
                subtitle="Discovered tools across registered servers"
                action={
                  <div className="toolbar-cluster">
                    <input
                      className="table-filter compact-filter"
                      onChange={(event) => setToolFilter(event.target.value)}
                      placeholder="Search tools"
                      value={toolFilter}
                    />
                    <select
                      className="table-filter compact-filter compact-select"
                      onChange={(event) => setToolServerFilter(event.target.value)}
                      value={toolServerFilter}
                    >
                      <option value="all">All servers</option>
                      {uniqueToolServers.map((server) => (
                        <option key={server} value={server}>
                          {server}
                        </option>
                      ))}
                    </select>
                  </div>
                }
              >
                {data.tools.empty_state && filteredTools.length === 0 ? (
                  <EmptyStateCard emptyState={data.tools.empty_state} />
                ) : (
                  <div className="tools-table-wrap">
                    <table className="data-table compact-table tools-table">
                      <thead>
                        <tr>
                          <th aria-hidden="true" className="expand-column"></th>
                          <th>Tool</th>
                          <th>Canonical name</th>
                          <th>Server</th>
                          <th>Description</th>
                          <th>Status</th>
                          <th>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {filteredTools.map((tool) => {
                          const muted = !tool.enabled || !tool.server_enabled;
                          const expanded = expandedTool === tool.canonical_name;
                          const fields = parseToolSchemaFields(tool.input_schema);
                          return (
                            <Fragment key={tool.canonical_name}>
                              <tr
                                aria-expanded={expanded}
                                className={`${expanded ? "is-selected" : ""} ${muted ? "is-muted" : ""} tool-summary-row`}
                                onClick={() =>
                                  setExpandedTool(expanded ? null : tool.canonical_name)
                                }
                              >
                                <td className="expand-column">
                                  <ChevronIcon expanded={expanded} />
                                </td>
                                <td>
                                  <div className="table-primary">{tool.name}</div>
                                </td>
                                <td>
                                  <code className="identifier-code" title={tool.canonical_name}>
                                    {tool.canonical_name}
                                  </code>
                                </td>
                                <td>{tool.server}</td>
                                <td>
                                  <div className="clamped-description" title={toolDescription(tool)}>
                                    {toolDescription(tool)}
                                  </div>
                                </td>
                                <td>
                                  <div className="tool-state-line">
                                    <StatusBadge
                                      text={tool.enabled ? "Enabled" : "Disabled"}
                                      tone={tool.enabled ? "good" : "muted"}
                                    />
                                    {!tool.server_enabled ? (
                                      <StatusBadge text="Server disabled" tone="warn" />
                                    ) : null}
                                  </div>
                                </td>
                                <td>
                                  <div
                                    className="row-actions"
                                    onClick={(event) => event.stopPropagation()}
                                  >
                                    <CopyButton
                                      ariaLabel="Copy canonical name"
                                      title="Copy canonical name"
                                      value={tool.canonical_name}
                                    />
                                    <button
                                      className="secondary-action"
                                      disabled={isBusy(`tool-toggle:${tool.canonical_name}`)}
                                      onClick={() => void toggleToolEnabled(tool)}
                                      type="button"
                                    >
                                      {isBusy(`tool-toggle:${tool.canonical_name}`)
                                        ? "Saving..."
                                        : tool.enabled
                                          ? "Disable"
                                          : "Enable"}
                                    </button>
                                  </div>
                                </td>
                              </tr>
                              {expanded ? (
                                <tr className="tool-expanded-row">
                                  <td className="tool-expanded-cell" colSpan={7}>
                                    <div className="tool-detail-panel">
                                      <div className="tool-detail-header">
                                        <p className="panel-label">Tool details</p>
                                      </div>

                                      <dl className="tool-detail-meta">
                                        <div className="tool-detail-description">
                                          <dt>Description</dt>
                                          <dd>{toolDescription(tool)}</dd>
                                        </div>
                                      </dl>

                                      <div className="tool-schema-section">
                                        <div className="tool-schema-header">
                                          <h4>Input fields</h4>
                                        </div>
                                        {fields.length > 0 ? (
                                          <div className="schema-field-list">
                                            {fields.map((field) => (
                                              <article className="schema-field-card" key={field.path}>
                                                <div className="schema-field-head">
                                                  <code>{field.path}</code>
                                                  <span className="schema-type-pill">
                                                    <code>{field.type}</code>
                                                  </span>
                                                </div>
                                                <dl className="schema-field-meta">
                                                  <div>
                                                    <dt>Required</dt>
                                                    <dd>{field.required ? "yes" : "no"}</dd>
                                                  </div>
                                                  {field.description ? (
                                                    <div>
                                                      <dt>Description</dt>
                                                      <dd>{field.description}</dd>
                                                    </div>
                                                  ) : null}
                                                  {field.enumValues?.length ? (
                                                    <div>
                                                      <dt>Enum</dt>
                                                      <dd>
                                                        <code>{field.enumValues.join(", ")}</code>
                                                      </dd>
                                                    </div>
                                                  ) : null}
                                                  {field.defaultValue ? (
                                                    <div>
                                                      <dt>Default</dt>
                                                      <dd>
                                                        <code>{field.defaultValue}</code>
                                                      </dd>
                                                    </div>
                                                  ) : null}
                                                  {field.note ? (
                                                    <div>
                                                      <dt>Notes</dt>
                                                      <dd>{field.note}</dd>
                                                    </div>
                                                  ) : null}
                                                </dl>
                                              </article>
                                            ))}
                                          </div>
                                        ) : (
                                          <p className="empty-inline">No structured input fields were provided.</p>
                                        )}
                                      </div>

                                      <details className="raw-schema-disclosure">
                                        <summary>Raw schema</summary>
                                        <div className="raw-schema-code-wrap">
                                          <CopyButton
                                            ariaLabel="Copy raw schema"
                                            title="Copy raw schema"
                                            value={prettyJSON(tool.input_schema)}
                                          />
                                          <pre className="schema-code">
                                          <code>{prettyJSON(tool.input_schema)}</code>
                                          </pre>
                                        </div>
                                      </details>
                                    </div>
                                  </td>
                                </tr>
                              ) : null}
                            </Fragment>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </SectionCard>
            ) : null}

            {section === "tool_groups" && data.toolGroups ? (
              <SectionCard
                title="Configured tool groups"
                subtitle=""
                action={
                  <button className="primary-action" onClick={openToolGroupModal} type="button">
                    + Add Tool Group
                  </button>
                }
              >
                {data.toolGroups.empty_state && data.toolGroups.tool_groups.length === 0 ? (
                  <EmptyStateCard emptyState={data.toolGroups.empty_state} />
                ) : (
                  <div className="tools-table-wrap">
                    <table className="data-table compact-table prompts-table">
                      <thead>
                        <tr>
                          <th aria-hidden="true" className="expand-column"></th>
                          <th>Group</th>
                          <th>Tools</th>
                          <th>Description</th>
                          <th>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {data.toolGroups.tool_groups.map((group) => {
                          const expanded = expandedToolGroup === group.name;
                          return (
                            <Fragment key={group.name}>
                              <tr
                                aria-expanded={expanded}
                                className={`${expanded ? "is-selected" : ""} tool-summary-row`}
                                onClick={() => setExpandedToolGroup(expanded ? null : group.name)}
                              >
                                <td className="expand-column">
                                  <ChevronIcon expanded={expanded} />
                                </td>
                                <td>
                                  <div className="table-primary">{group.name}</div>
                                </td>
                                <td>
                                  <strong>{group.tool_count}</strong>
                                </td>
                                <td>
                                  <div className="clamped-description" title={group.description || "No description"}>
                                    {group.description || "No description"}
                                  </div>
                                </td>
                                <td>
                                  <div className="row-actions" onClick={(event) => event.stopPropagation()}>
                                    <button
                                      aria-label="Delete tool group"
                                      className="danger-action icon-button danger-icon-button"
                                      disabled={isBusy(`tool-group-delete:${group.name}`)}
                                      onClick={() => void deleteToolGroup(group)}
                                      title="Delete tool group"
                                      type="button"
                                    >
                                      <TrashIcon />
                                    </button>
                                  </div>
                                </td>
                              </tr>
                              {expanded ? (
                                <tr className="tool-expanded-row">
                                  <td className="tool-expanded-cell" colSpan={5}>
                                    <div className="tool-detail-panel">
                                      <div className="tool-detail-header">
                                        <p className="panel-label">Tool group details</p>
                                      </div>
                                      {group.description ? (
                                        <dl className="tool-detail-meta">
                                          <div className="tool-detail-description">
                                            <dt>Description</dt>
                                            <dd>{group.description}</dd>
                                          </div>
                                        </dl>
                                      ) : null}
                                      <div className="tool-schema-section">
                                        <div className="tool-schema-header">
                                          <h4>MCP endpoints</h4>
                                        </div>
                                        <div className="tool-group-endpoints">
                                          <div className="tool-group-endpoint-row">
                                            <span className="tool-group-endpoint-label">Streamable HTTP</span>
                                            <div className="tool-group-endpoint-value">
                                              <code className="detail-target-code" title={group.streamable_http_endpoint}>
                                                {group.streamable_http_endpoint}
                                              </code>
                                              <CopyButton
                                                ariaLabel="Copy Streamable HTTP endpoint"
                                                title="Copy Streamable HTTP endpoint"
                                                value={group.streamable_http_endpoint}
                                              />
                                            </div>
                                          </div>
                                          <div className="tool-group-endpoint-row">
                                            <span className="tool-group-endpoint-label">SSE</span>
                                            <div className="tool-group-endpoint-stack">
                                              <div className="tool-group-endpoint-value">
                                                <code className="detail-target-code" title={group.sse_endpoint}>
                                                  {group.sse_endpoint}
                                                </code>
                                                <CopyButton
                                                  ariaLabel="Copy SSE endpoint"
                                                  title="Copy SSE endpoint"
                                                  value={group.sse_endpoint}
                                                />
                                              </div>
                                              <div className="tool-group-endpoint-value">
                                                <code className="detail-target-code" title={group.sse_message_endpoint}>
                                                  {group.sse_message_endpoint}
                                                </code>
                                                <CopyButton
                                                  ariaLabel="Copy SSE message endpoint"
                                                  title="Copy SSE message endpoint"
                                                  value={group.sse_message_endpoint}
                                                />
                                              </div>
                                            </div>
                                          </div>
                                        </div>
                                      </div>

                                      <div className="tool-schema-section">
                                        <div className="tool-schema-header">
                                          <h4>Included tools</h4>
                                        </div>
                                        {group.tools.length > 0 ? (
                                          <div className="schema-field-list">
                                            {group.tools.map((tool) => (
                                              <article className="schema-field-card" key={tool.canonical_name}>
                                                <div className="schema-field-head">
                                                  <code>{tool.canonical_name}</code>
                                                  <span className="schema-type-pill">
                                                    <code>{tool.server}</code>
                                                  </span>
                                                </div>
                                                <dl className="schema-field-meta">
                                                  {tool.description ? (
                                                    <div>
                                                      <dt>Description</dt>
                                                      <dd>{tool.description}</dd>
                                                    </div>
                                                  ) : null}
                                                </dl>
                                              </article>
                                            ))}
                                          </div>
                                        ) : (
                                          <p className="empty-inline">No tools in this group.</p>
                                        )}
                                      </div>
                                    </div>
                                  </td>
                                </tr>
                              ) : null}
                            </Fragment>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </SectionCard>
            ) : null}

            {section === "prompts" && data.prompts ? (
              <SectionCard
                title="Prompts"
                subtitle="Discovered prompt templates"
                action={
                  <div className="toolbar-cluster">
                    <input
                      className="table-filter compact-filter"
                      onChange={(event) => setPromptFilter(event.target.value)}
                      placeholder="Search prompts"
                      value={promptFilter}
                    />
                  </div>
                }
              >
                {data.prompts.empty_state && filteredPrompts.length === 0 ? (
                  <EmptyStateCard emptyState={data.prompts.empty_state} />
                ) : (
                  <div className="tools-table-wrap">
                    <table className="data-table compact-table prompts-table">
                      <thead>
                        <tr>
                          <th aria-hidden="true" className="expand-column"></th>
                          <th>Prompt</th>
                          <th>Canonical name</th>
                          <th>Server</th>
                          <th>Description</th>
                          <th>Status</th>
                          <th>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {filteredPrompts.map((prompt) => {
                          const muted = !prompt.enabled || !prompt.server_enabled;
                          const expanded = expandedPrompt === prompt.canonical_name;
                          const fields = parsePromptArgumentFields(prompt.arguments);
                          return (
                            <Fragment key={prompt.canonical_name}>
                              <tr
                                aria-expanded={expanded}
                                className={`${expanded ? "is-selected" : ""} ${muted ? "is-muted" : ""} tool-summary-row`}
                                onClick={() =>
                                  setExpandedPrompt(expanded ? null : prompt.canonical_name)
                                }
                              >
                                <td className="expand-column">
                                  <ChevronIcon expanded={expanded} />
                                </td>
                                <td>
                                  <div className="table-primary">{prompt.name}</div>
                                </td>
                                <td>
                                  <code className="identifier-code" title={prompt.canonical_name}>
                                    {prompt.canonical_name}
                                  </code>
                                </td>
                                <td>{prompt.server}</td>
                                <td>
                                  <div className="clamped-description" title={promptDescription(prompt)}>
                                    {promptDescription(prompt)}
                                  </div>
                                </td>
                                <td>
                                  <div className="tool-state-line">
                                    <StatusBadge
                                      text={prompt.enabled ? "Enabled" : "Disabled"}
                                      tone={prompt.enabled ? "good" : "muted"}
                                    />
                                    {!prompt.server_enabled ? (
                                      <StatusBadge text="Server disabled" tone="warn" />
                                    ) : null}
                                  </div>
                                </td>
                                <td>
                                  <div
                                    className="row-actions"
                                    onClick={(event) => event.stopPropagation()}
                                  >
                                    <CopyButton
                                      ariaLabel="Copy canonical name"
                                      title="Copy canonical name"
                                      value={prompt.canonical_name}
                                    />
                                    <button
                                      className="secondary-action"
                                      disabled={isBusy(`prompt-toggle:${prompt.canonical_name}`)}
                                      onClick={() => void togglePromptEnabled(prompt)}
                                      type="button"
                                    >
                                      {isBusy(`prompt-toggle:${prompt.canonical_name}`)
                                        ? "Saving..."
                                        : prompt.enabled
                                          ? "Disable"
                                          : "Enable"}
                                    </button>
                                  </div>
                                </td>
                              </tr>
                              {expanded ? (
                                <tr className="tool-expanded-row">
                                  <td className="tool-expanded-cell" colSpan={7}>
                                    <div className="tool-detail-panel">
                                      <div className="tool-detail-header">
                                        <p className="panel-label">Prompt details</p>
                                      </div>

                                      <dl className="tool-detail-meta">
                                        <div className="tool-detail-description">
                                          <dt>Description</dt>
                                          <dd>{promptDescription(prompt)}</dd>
                                        </div>
                                      </dl>

                                      <div className="tool-schema-section">
                                        <div className="tool-schema-header">
                                          <h4>Arguments</h4>
                                        </div>
                                        {fields.length > 0 ? (
                                          <div className="schema-field-list">
                                            {fields.map((field) => (
                                              <article className="schema-field-card" key={field.path}>
                                                <div className="schema-field-head">
                                                  <code>{field.path}</code>
                                                  <span className="schema-type-pill">
                                                    <code>{field.type}</code>
                                                  </span>
                                                </div>
                                                <dl className="schema-field-meta">
                                                  <div>
                                                    <dt>Required</dt>
                                                    <dd>{field.required ? "yes" : "no"}</dd>
                                                  </div>
                                                  {field.description ? (
                                                    <div>
                                                      <dt>Description</dt>
                                                      <dd>{field.description}</dd>
                                                    </div>
                                                  ) : null}
                                                  {field.enumValues?.length ? (
                                                    <div>
                                                      <dt>Enum</dt>
                                                      <dd>
                                                        <code>{field.enumValues.join(", ")}</code>
                                                      </dd>
                                                    </div>
                                                  ) : null}
                                                  {field.defaultValue ? (
                                                    <div>
                                                      <dt>Default</dt>
                                                      <dd>
                                                        <code>{field.defaultValue}</code>
                                                      </dd>
                                                    </div>
                                                  ) : null}
                                                  {field.note ? (
                                                    <div>
                                                      <dt>Notes</dt>
                                                      <dd>{field.note}</dd>
                                                    </div>
                                                  ) : null}
                                                </dl>
                                              </article>
                                            ))}
                                          </div>
                                        ) : (
                                          <p className="empty-inline">No arguments.</p>
                                        )}
                                      </div>

                                      <details className="raw-schema-disclosure">
                                        <summary>Raw arguments</summary>
                                        <div className="raw-schema-actions">
                                          <CopyButton
                                            ariaLabel="Copy raw arguments"
                                            title="Copy raw arguments"
                                            value={prettyPromptArguments(prompt.arguments)}
                                          />
                                        </div>
                                        <pre className="schema-code">
                                          <code>{prettyPromptArguments(prompt.arguments)}</code>
                                        </pre>
                                      </details>
                                    </div>
                                  </td>
                                </tr>
                              ) : null}
                            </Fragment>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </SectionCard>
            ) : null}

            {section === "resources" && data.resources ? (
              <SectionCard title="Resources" subtitle="Discovered MCP resources">
                {data.resources.empty_state && data.resources.resources.length === 0 ? (
                  <EmptyStateCard emptyState={data.resources.empty_state} />
                ) : (
                  <table className="data-table compact-table resources-table">
                    <thead>
                      <tr>
                        <th>Name</th>
                        <th>URI</th>
                        <th>Server</th>
                        <th>MIME</th>
                        <th>Description</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.resources.resources.map((resource) => (
                        <tr key={resource.uri}>
                          <td>{resource.name}</td>
                          <td>
                            <div className="inline-copy resource-uri-cell">
                              <code className="identifier-code" title={resource.uri}>
                                {resource.uri}
                              </code>
                              <CopyButton
                                ariaLabel="Copy resource URI"
                                title="Copy resource URI"
                                value={resource.uri}
                              />
                            </div>
                          </td>
                          <td>{resource.server}</td>
                          <td>
                            <code>{resource.mime_type || "Unknown"}</code>
                          </td>
                          <td>{resourceDescription(resource)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </SectionCard>
            ) : null}

            {section === "diagnostics" && diagnostics ? (
              <>
                <SectionCard title="System Info" subtitle="Runtime details">
                  <div className="diagnostics-grid compact-diagnostics-grid">
                    <div className="diag-card compact-metric">
                      <span>Version</span>
                      <strong>{shortVersion(diagnostics.version)}</strong>
                    </div>
                    <div className="diag-card compact-metric">
                      <span>Mode</span>
                      <strong>{diagnostics.mode}</strong>
                    </div>
                    <div className="diag-card compact-metric">
                      <span>Database</span>
                      <strong>{diagnostics.database}</strong>
                    </div>
                  </div>
                </SectionCard>

                <SectionCard title="Runtime details" subtitle="System information">
                  <dl className="diagnostic-list compact-diagnostic-list">
                    <div>
                      <dt>Full build</dt>
                      <dd>
                        <code>{diagnostics.version}</code>
                      </dd>
                    </div>
                    <div>
                      <dt>Global MCP Endpoint</dt>
                      <dd>
                        <code>{diagnostics.primary_endpoint}</code>
                      </dd>
                    </div>
                    <div>
                      <dt>Enabled transports</dt>
                      <dd>
                        <code>{diagnostics.enabled_transports.join(", ")}</code>
                      </dd>
                    </div>
                  </dl>
                </SectionCard>
              </>
            ) : null}
          </div>
        ) : null}

        {toolGroupOpen ? (
          <div className="modal-backdrop" onClick={closeToolGroupModal} role="presentation">
            <section className="modal-panel" onClick={(event) => event.stopPropagation()}>
              <div className="modal-header">
                <div>
                  <p className="panel-label">Tool Groups</p>
                  <h2>Add Tool Group</h2>
                </div>
                <button className="secondary-action" onClick={closeToolGroupModal} type="button">
                  Close
                </button>
              </div>

              <div className="modal-form">
                <label className="form-field">
                  <span>Group name</span>
                  <input
                    className="table-filter form-input"
                    onChange={(event) => setToolGroupForm((current) => ({ ...current, name: event.target.value }))}
                    placeholder="coding"
                    value={toolGroupForm.name}
                  />
                </label>

                <label className="form-field">
                  <span>Description</span>
                  <input
                    className="table-filter form-input"
                    onChange={(event) =>
                      setToolGroupForm((current) => ({ ...current, description: event.target.value }))
                    }
                    placeholder="Tools useful for coding workflows"
                    value={toolGroupForm.description}
                  />
                </label>

                <div className="tool-group-builder">
                  <div className="tool-group-selector panel">
                    <div className="tool-group-selector-header">
                      <strong>Available tools</strong>
                    </div>
                    {(data.tools?.tools.length ?? 0) > 0 ? (
                      <>
                        <div className="toolbar-cluster">
                          <input
                            className="table-filter compact-filter"
                            onChange={(event) => setToolGroupToolFilter(event.target.value)}
                            placeholder="Search tools"
                            value={toolGroupToolFilter}
                          />
                          <select
                            className="table-filter compact-filter compact-select"
                            onChange={(event) => setToolGroupToolServerFilter(event.target.value)}
                            value={toolGroupToolServerFilter}
                          >
                            <option value="all">All servers</option>
                            {uniqueToolServers.map((server) => (
                              <option key={server} value={server}>
                                {server}
                              </option>
                            ))}
                          </select>
                        </div>
                        <div className="tool-pick-list">
                          {availableToolGroupTools.map((tool) => {
                            const selected = toolGroupForm.selectedTools.includes(tool.canonical_name);
                            return (
                              <button
                                className={`tool-pick-item ${selected ? "is-selected" : ""}`}
                                key={tool.canonical_name}
                                onClick={() => toggleToolGroupSelection(tool.canonical_name)}
                                type="button"
                              >
                                <div className="table-primary">{tool.name}</div>
                                <code className="identifier-code" title={tool.canonical_name}>
                                  {tool.canonical_name}
                                </code>
                                <div className="table-secondary">{tool.server}</div>
                              </button>
                            );
                          })}
                        </div>
                      </>
                    ) : (
                      <p className="empty-inline">Register MCP servers first so tools are available to group.</p>
                    )}
                  </div>

                  <div className="tool-group-selector panel">
                    <div className="tool-group-selector-header">
                      <strong>Selected tools</strong>
                    </div>
                    {toolGroupForm.selectedTools.length > 0 ? (
                      <div className="selected-tool-list">
                        {toolGroupForm.selectedTools.map((toolName) => (
                          <button
                            className="selected-tool-chip"
                            key={toolName}
                            onClick={() => removeToolGroupSelection(toolName)}
                            type="button"
                          >
                            <code>{toolName}</code>
                            <span>Remove</span>
                          </button>
                        ))}
                      </div>
                    ) : (
                      <p className="empty-inline">Select at least one tool.</p>
                    )}
                  </div>
                </div>

                {toolGroupError ? <p className="form-error">{toolGroupError}</p> : null}
              </div>

              <div className="modal-footer">
                <button className="secondary-action" onClick={closeToolGroupModal} type="button">
                  Cancel
                </button>
                <button
                  className="primary-action"
                  disabled={isBusy("tool-group-create")}
                  onClick={() => void submitToolGroup()}
                  type="button"
                >
                  {isBusy("tool-group-create") ? "Saving..." : "+ Add Tool Group"}
                </button>
              </div>
            </section>
          </div>
        ) : null}

        {registerOpen ? (
          <div className="modal-backdrop" onClick={closeRegisterModal} role="presentation">
            <section className="modal-panel" onClick={(event) => event.stopPropagation()}>
              <div className="modal-header">
                <div>
                  <p className="panel-label">Add server</p>
                  <h2>{registerOAuth ? "Complete OAuth authorization" : "Register an MCP server"}</h2>
                </div>
                <button className="secondary-action" onClick={closeRegisterModal} type="button">
                  Close
                </button>
              </div>

              {registerOAuth ? (
                <div className="modal-form oauth-step">
                  <p className="oauth-message">
                    This MCP server requires OAuth authorization. Continue in your browser to complete
                    registration.
                  </p>
                  <div className="oauth-status-card">
                    <div>
                      <span className="oauth-status-label">Status</span>
                      <strong>
                        {registerOAuth.hasOpenedBrowser
                          ? "Waiting for OAuth authorization..."
                          : "Authorization required"}
                      </strong>
                    </div>
                    {registerOAuth.authorization.expires_at ? (
                      <p className="oauth-status-meta">
                        Expires {new Date(registerOAuth.authorization.expires_at).toLocaleTimeString()}
                      </p>
                    ) : null}
                  </div>
                  {registerOAuth.error ? <p className="form-error">{registerOAuth.error}</p> : null}
                </div>
              ) : (
                <>
                  <div className="modal-form">
                    <label className="form-field">
                      <span>Server name</span>
                      <input
                        className="table-filter form-input"
                        onChange={(event) => updateRegisterField("name", event.target.value)}
                        placeholder={registerForm.transport === "streamable_http" ? "context7" : "filesystem"}
                        value={registerForm.name}
                      />
                    </label>

                    <label className="form-field">
                      <span>Description</span>
                      <input
                        className="table-filter form-input"
                        onChange={(event) => updateRegisterField("description", event.target.value)}
                        placeholder={
                          registerForm.transport === "streamable_http"
                            ? "context7 mcp server"
                            : "Local filesystem access"
                        }
                        value={registerForm.description}
                      />
                    </label>

                    <div className="form-grid">
                      <label className="form-field">
                        <span>Transport</span>
                        <select
                          className="table-filter form-input compact-select"
                          onChange={(event) =>
                            updateRegisterField(
                              "transport",
                              event.target.value as RegisterServerFormState["transport"],
                            )
                          }
                          value={registerForm.transport}
                        >
                          <option value="stdio">stdio</option>
                          <option value="streamable_http">streamable_http</option>
                          <option value="sse">sse</option>
                        </select>
                      </label>

                      <label className="form-field">
                        <span>Session mode</span>
                        <select
                          className="table-filter form-input compact-select"
                          onChange={(event) =>
                            updateRegisterField(
                              "session_mode",
                              event.target.value as RegisterServerFormState["session_mode"],
                            )
                          }
                          value={registerForm.session_mode}
                        >
                          <option value="stateless">stateless</option>
                          <option value="stateful">stateful</option>
                        </select>
                      </label>
                    </div>

                    {registerForm.transport === "stdio" ? (
                      <>
                        <label className="form-field">
                          <span>Command</span>
                          <input
                            className="table-filter form-input"
                            onChange={(event) => updateRegisterField("command", event.target.value)}
                            placeholder="npx"
                            value={registerForm.command}
                          />
                        </label>

                        <label className="form-field">
                          <span>Arguments</span>
                          <textarea
                            className="table-filter form-input form-textarea"
                            onChange={(event) => updateRegisterField("args_text", event.target.value)}
                            placeholder="-y&#10;@modelcontextprotocol/server-filesystem"
                            value={registerForm.args_text}
                          />
                        </label>

                        <div className="form-field">
                          <span>Environment variables</span>
                          <div className="key-value-list">
                            {registerForm.env_rows.map((row, index) => (
                              <div className="key-value-row" key={`env-${index}`}>
                                <input
                                  className="table-filter form-input"
                                  onChange={(event) =>
                                    updateKeyValueRow("env_rows", index, "key", event.target.value)
                                  }
                                  placeholder="KEY"
                                  value={row.key}
                                />
                                <input
                                  className="table-filter form-input"
                                  onChange={(event) =>
                                    updateKeyValueRow("env_rows", index, "value", event.target.value)
                                  }
                                  placeholder="value"
                                  value={row.value}
                                />
                                <button
                                  className="secondary-action"
                                  onClick={() => removeKeyValueRow("env_rows", index)}
                                  type="button"
                                >
                                  Remove
                                </button>
                              </div>
                            ))}
                          </div>
                          <button
                            className="secondary-action inline-action"
                            onClick={() => addKeyValueRow("env_rows")}
                            type="button"
                          >
                            Add env var
                          </button>
                        </div>
                      </>
                    ) : (
                      <>
                        <label className="form-field">
                          <span>Target URL</span>
                          <input
                            className="table-filter form-input"
                            onChange={(event) => updateRegisterField("url", event.target.value)}
                            placeholder={
                              registerForm.transport === "streamable_http"
                                ? "https://mcp.context7.com/mcp"
                                : "http://127.0.0.1:8000/mcp"
                            }
                            value={registerForm.url}
                          />
                        </label>

                        <label className="form-field">
                          <span>Bearer token</span>
                          <input
                            className="table-filter form-input"
                            onChange={(event) => updateRegisterField("bearer_token", event.target.value)}
                            placeholder="Optional"
                            type="password"
                            value={registerForm.bearer_token}
                          />
                        </label>

                        {registerForm.transport === "streamable_http" ? (
                          <div className="form-field">
                            <span>Headers</span>
                            <div className="key-value-list">
                              {registerForm.header_rows.map((row, index) => (
                                <div className="key-value-row" key={`header-${index}`}>
                                  <input
                                    className="table-filter form-input"
                                    onChange={(event) =>
                                      updateKeyValueRow("header_rows", index, "key", event.target.value)
                                    }
                                    placeholder="Header"
                                    value={row.key}
                                  />
                                  <input
                                    className="table-filter form-input"
                                    onChange={(event) =>
                                      updateKeyValueRow("header_rows", index, "value", event.target.value)
                                    }
                                    placeholder="Value"
                                    value={row.value}
                                  />
                                  <button
                                    className="secondary-action"
                                    onClick={() => removeKeyValueRow("header_rows", index)}
                                    type="button"
                                  >
                                    Remove
                                  </button>
                                </div>
                              ))}
                            </div>
                            <button
                              className="secondary-action inline-action"
                              onClick={() => addKeyValueRow("header_rows")}
                              type="button"
                            >
                              Add header
                            </button>
                          </div>
                        ) : null}
                      </>
                    )}

                    {registerError ? <p className="form-error">{registerError}</p> : null}
                  </div>
                </>
              )}

              <div className="modal-footer">
                {registerOAuth ? (
                  <>
                    <button
                      className="secondary-action"
                      onClick={() => resetRegisterOAuthStep("Start registration again to retry OAuth.")}
                      type="button"
                    >
                      Start over
                    </button>
                    <button className="primary-action" onClick={startRegisterOAuth} type="button">
                      {registerOAuth.hasOpenedBrowser ? "Open OAuth again" : "Continue OAuth"}
                    </button>
                  </>
                ) : (
                  <>
                    <button className="secondary-action" onClick={closeRegisterModal} type="button">
                      Cancel
                    </button>
                    <button
                      className="primary-action"
                      disabled={isBusy("register-server")}
                      onClick={() => void submitRegisterServer()}
                      type="button"
                    >
                      {isBusy("register-server") ? "Registering..." : "+ Add Server"}
                    </button>
                  </>
                )}
              </div>
            </section>
          </div>
        ) : null}
      </main>
    </div>
  );
}
