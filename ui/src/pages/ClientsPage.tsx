import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useAppContext } from "../App";
import { AllowListEditor } from "../components/forms/AllowListEditor";
import { ConfirmDialog } from "../components/ui/ConfirmDialog";
import { EmptyState } from "../components/ui/EmptyState";
import { api } from "../lib/api";
import type { McpClient, McpClientWithToken, McpServer } from "../lib/types";

type DrawerMode = "create" | "edit";

type ClientFormState = {
  name: string;
  description: string;
  allow_list: string[];
  access_token: string;
  rotate_access_token: boolean;
};

const emptyForm: ClientFormState = {
  name: "",
  description: "",
  allow_list: [],
  access_token: "",
  rotate_access_token: false,
};

function tokenModeLabel(form: ClientFormState) {
  if (form.rotate_access_token) return "Rotate (generate new)";
  if (form.access_token) return "Custom token";
  return "Keep existing";
}

export function ClientsPage() {
  const { token, isAdminEquivalent, settings } = useAppContext();
  const queryClient = useQueryClient();

  const [drawerMode, setDrawerMode] = useState<DrawerMode | null>(null);
  const [editTarget, setEditTarget] = useState<McpClient | null>(null);
  const [form, setForm] = useState<ClientFormState>(emptyForm);
  const [formError, setFormError] = useState("");
  const [pendingDelete, setPendingDelete] = useState("");
  // Newly returned tokens (create or rotate) — show once, clear on next action
  const [revealedTokens, setRevealedTokens] = useState<Record<string, string>>({});

  const clientsQuery = useQuery({
    queryKey: ["clients"],
    queryFn: () => api.clients(token || undefined),
    enabled: isAdminEquivalent && settings.mode !== "development",
  });

  const serversQuery = useQuery({
    queryKey: ["servers"],
    queryFn: () => api.servers(token || undefined),
    enabled: isAdminEquivalent,
  });

  const createMutation = useMutation({
    mutationFn: (body: Parameters<typeof api.createClient>[0]) =>
      api.createClient(body, token || undefined),
    onSuccess: (result: McpClientWithToken) => {
      void queryClient.invalidateQueries({ queryKey: ["clients"] });
      setRevealedTokens((prev) => ({ ...prev, [result.name]: result.access_token }));
      closeDrawer();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({ name, ...rest }: { name: string } & Parameters<typeof api.updateClient>[1]) =>
      api.updateClient(name, rest, token || undefined),
    onSuccess: (result: McpClientWithToken, variables) => {
      void queryClient.invalidateQueries({ queryKey: ["clients"] });
      // Only reveal updated token if it was actually changed
      if (variables.rotate_access_token || variables.access_token) {
        setRevealedTokens((prev) => ({ ...prev, [result.name]: result.access_token }));
      }
      closeDrawer();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => api.deleteClient(name, token || undefined),
    onSuccess: (_, name) => {
      setPendingDelete("");
      void queryClient.invalidateQueries({ queryKey: ["clients"] });
      setRevealedTokens((prev) => {
        const next = { ...prev };
        delete next[name];
        return next;
      });
    },
  });

  function openCreate() {
    setForm(emptyForm);
    setFormError("");
    setEditTarget(null);
    setDrawerMode("create");
  }

  function openEdit(client: McpClient) {
    setForm({
      name: client.name,
      description: client.description ?? "",
      allow_list: client.allow_list ?? [],
      access_token: "",
      rotate_access_token: false,
    });
    setFormError("");
    setEditTarget(client);
    setDrawerMode("edit");
  }

  function closeDrawer() {
    setDrawerMode(null);
    setEditTarget(null);
    setFormError("");
  }

  function submitForm() {
    setFormError("");
    if (form.rotate_access_token && form.access_token.trim()) {
      setFormError("Provide either a custom token or rotate — not both.");
      return;
    }
    if (drawerMode === "create") {
      if (!form.name.trim()) {
        setFormError("Name is required");
        return;
      }
      createMutation.mutate({
        name: form.name.trim(),
        description: form.description,
        allow_list: form.allow_list,
        access_token: form.access_token.trim() || undefined,
      });
    } else if (editTarget) {
      updateMutation.mutate({
        name: editTarget.name,
        description: form.description,
        allow_list: form.allow_list,
        access_token: form.access_token.trim() || undefined,
        rotate_access_token: form.rotate_access_token,
      });
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending;
  const clients = clientsQuery.data ?? [];
  const servers: McpServer[] = serversQuery.data ?? [];

  if (!isAdminEquivalent || settings.mode === "development") {
    return (
      <div className="rounded-panel border border-line bg-panel p-8">
        <p className="text-xs font-medium uppercase tracking-widest text-muted">Clients</p>
        <h2 className="mt-3 text-xl font-semibold text-body">Enterprise mode required</h2>
        <p className="mt-3 max-w-lg text-sm leading-6 text-muted">
          MCP client management is only available when the gateway is running in{" "}
          <span className="font-medium text-body">enterprise</span> or{" "}
          <span className="font-medium text-body">production</span> mode. In those modes, every
          caller must present a Bearer token, and clients define which tools each caller can
          access via an allow-list.
        </p>
        <p className="mt-4 text-sm leading-6 text-muted">
          You are currently in{" "}
          <span className="rounded bg-accent/15 px-1.5 py-0.5 font-mono text-xs text-accent">
            {settings.mode}
          </span>{" "}
          mode, which has no auth enforcement. To enable client management, reinitialize the
          gateway and select <span className="font-medium text-body">Enterprise</span> on the
          setup screen.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.22em] text-muted">MCP consumers</p>
          <h2 className="mt-2 text-3xl font-semibold text-body">Clients</h2>
        </div>
        <button
          className="rounded-md bg-accent px-4 py-3 text-sm font-semibold text-ink"
          onClick={openCreate}
        >
          New client
        </button>
      </div>

      {drawerMode ? (
        <ClientForm
          form={form}
          setForm={setForm}
          title={drawerMode === "create" ? "Create client" : `Edit ${editTarget?.name}`}
          isEdit={drawerMode === "edit"}
          onSubmit={submitForm}
          onCancel={closeDrawer}
          isPending={isPending}
          error={formError}
          servers={servers}
        />
      ) : null}

      {clients.length === 0 && !clientsQuery.isLoading ? (
        <EmptyState title="No clients yet" message="Create a client to issue MCP access tokens." />
      ) : (
        <div className="space-y-4">
          {clients.map((client) => (
            <div key={client.name} className="rounded-panel border border-line bg-panel p-5">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="min-w-0 flex-1">
                  <h3 className="text-base font-semibold text-body">{client.name}</h3>
                  {client.description ? (
                    <p className="mt-1 text-sm text-muted">{client.description}</p>
                  ) : null}
                  <div className="mt-2 flex flex-wrap items-center gap-1.5">
                    <span className="text-xs text-muted">allow list:</span>
                    {(client.allow_list ?? []).length > 0 ? (
                      (client.allow_list ?? []).map((entry) => (
                        <span
                          key={entry}
                          className="rounded border border-line bg-elevated px-2 py-0.5 font-mono text-xs text-body"
                        >
                          {entry}
                        </span>
                      ))
                    ) : (
                      <span className="text-xs text-muted italic">empty (no access)</span>
                    )}
                  </div>
                  {revealedTokens[client.name] ? (
                    <div className="mt-3 rounded-ui border border-accent/30 bg-accent/5 px-3 py-2">
                      <p className="mb-1 text-xs font-medium text-accent">Token (shown once)</p>
                      <code className="break-all font-mono text-xs text-body">
                        {revealedTokens[client.name]}
                      </code>
                      <button
                        className="ml-3 text-xs text-muted underline"
                        onClick={() =>
                          setRevealedTokens((prev) => {
                            const next = { ...prev };
                            delete next[client.name];
                            return next;
                          })
                        }
                      >
                        Dismiss
                      </button>
                    </div>
                  ) : null}
                </div>
                <div className="flex shrink-0 gap-2">
                  <button
                    className="rounded-md border border-line px-3 py-1.5 text-xs text-body transition hover:border-accent hover:text-accent"
                    onClick={() => openEdit(client)}
                  >
                    Edit
                  </button>
                  <button
                    className="rounded-md border border-down/30 px-3 py-1.5 text-xs text-down transition hover:bg-down/10"
                    onClick={() => setPendingDelete(client.name)}
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {pendingDelete ? (
        <ConfirmDialog
          title="Delete client"
          message={`Delete client "${pendingDelete}"? This immediately revokes its access token.`}
          confirmLabel="Delete"
          onConfirm={() => deleteMutation.mutate(pendingDelete)}
          onCancel={() => setPendingDelete("")}
        />
      ) : null}
    </div>
  );
}

type ClientFormProps = {
  form: ClientFormState;
  setForm: React.Dispatch<React.SetStateAction<ClientFormState>>;
  title: string;
  isEdit: boolean;
  onSubmit: () => void;
  onCancel: () => void;
  isPending: boolean;
  error: string;
  servers?: McpServer[];
};

function ClientForm({ form, setForm, title, isEdit, onSubmit, onCancel, isPending, error, servers = [] }: ClientFormProps) {
  function update<K extends keyof ClientFormState>(key: K, value: ClientFormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  return (
    <div className="rounded-panel border border-accent/30 bg-panel p-6">
      <h3 className="text-base font-semibold text-body">{title}</h3>
      <div className="mt-4 space-y-4">
        {!isEdit ? (
          <Field label="Name">
            <input
              className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
              value={form.name}
              onChange={(e) => update("name", e.target.value)}
            />
          </Field>
        ) : null}
        <Field label="Description">
          <input
            className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
            value={form.description}
            onChange={(e) => update("description", e.target.value)}
          />
        </Field>
        <Field label="Allow list">
          <AllowListEditor
            value={form.allow_list}
            onChange={(next) => update("allow_list", next)}
            servers={servers}
          />
        </Field>
        <Field
          label={isEdit ? "Token management" : "Access token (leave blank to auto-generate)"}
        >
          {isEdit ? (
            <div className="space-y-3">
              <label className="flex items-center gap-2 text-sm text-body">
                <input
                  type="checkbox"
                  checked={form.rotate_access_token}
                  onChange={(e) => {
                    update("rotate_access_token", e.target.checked);
                    if (e.target.checked) update("access_token", "");
                  }}
                />
                Rotate token (generate new)
              </label>
              {!form.rotate_access_token ? (
                <input
                  className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
                  placeholder="Or enter custom token"
                  value={form.access_token}
                  onChange={(e) => update("access_token", e.target.value)}
                />
              ) : null}
              <p className="text-xs text-muted">Current: {tokenModeLabel(form)}</p>
            </div>
          ) : (
            <input
              className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
              placeholder="Leave blank to auto-generate"
              value={form.access_token}
              onChange={(e) => update("access_token", e.target.value)}
            />
          )}
        </Field>
      </div>
      {error ? <p className="mt-3 text-sm text-down">{error}</p> : null}
      <div className="mt-5 flex gap-3">
        <button
          className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink disabled:opacity-40"
          disabled={isPending}
          onClick={onSubmit}
        >
          {isPending ? "Saving…" : isEdit ? "Save changes" : "Create client"}
        </button>
        <button
          className="rounded-md border border-line px-4 py-2 text-sm text-body"
          onClick={onCancel}
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-xs font-medium text-muted">{label}</span>
      {children}
    </label>
  );
}
