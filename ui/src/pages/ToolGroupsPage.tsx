import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useAppContext } from "../App";
import { ConfirmDialog } from "../components/ui/ConfirmDialog";
import { EmptyState } from "../components/ui/EmptyState";
import { api } from "../lib/api";
import type { ToolGroup } from "../lib/types";

type GroupFormState = {
  name: string;
  description: string;
  included_tools: string;
  included_servers: string;
  excluded_tools: string;
};

const emptyForm: GroupFormState = {
  name: "",
  description: "",
  included_tools: "",
  included_servers: "",
  excluded_tools: "",
};

function parseList(raw: string): string[] {
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

function toFormState(group: ToolGroup): GroupFormState {
  return {
    name: group.name,
    description: group.description ?? "",
    included_tools: (group.included_tools ?? []).join(", "),
    included_servers: (group.included_servers ?? []).join(", "),
    excluded_tools: (group.excluded_tools ?? []).join(", "),
  };
}

function TokenChip({ value }: { value: string }) {
  return (
    <span className="inline-block rounded border border-line bg-elevated px-2 py-0.5 font-mono text-xs text-body">
      {value}
    </span>
  );
}

export function ToolGroupsPage() {
  const { token, isAdminEquivalent } = useAppContext();
  const queryClient = useQueryClient();

  const [editTarget, setEditTarget] = useState<ToolGroup | null>(null);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<GroupFormState>(emptyForm);
  const [formError, setFormError] = useState("");
  const [pendingDelete, setPendingDelete] = useState("");
  const [previewGroup, setPreviewGroup] = useState("");

  const groupsQuery = useQuery({
    queryKey: ["tool-groups"],
    queryFn: () => api.toolGroups(token || undefined),
    enabled: isAdminEquivalent,
  });

  const effectiveQuery = useQuery({
    queryKey: ["tool-groups-effective", previewGroup],
    queryFn: () => api.getToolGroupEffectiveTools(previewGroup, token || undefined),
    enabled: Boolean(previewGroup),
  });

  const createMutation = useMutation({
    mutationFn: (body: Omit<ToolGroup, "name"> & { name: string }) =>
      api.createToolGroup(body, token || undefined),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["tool-groups"] });
      closeForm();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({ name, body }: { name: string; body: Partial<Omit<ToolGroup, "name">> }) =>
      api.updateToolGroup(name, body, token || undefined),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["tool-groups"] });
      closeForm();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => api.deleteToolGroup(name, token || undefined),
    onSuccess: () => {
      setPendingDelete("");
      void queryClient.invalidateQueries({ queryKey: ["tool-groups"] });
      if (previewGroup === pendingDelete) setPreviewGroup("");
    },
  });

  function openCreate() {
    setForm(emptyForm);
    setFormError("");
    setCreating(true);
    setEditTarget(null);
  }

  function openEdit(group: ToolGroup) {
    setForm(toFormState(group));
    setFormError("");
    setEditTarget(group);
    setCreating(false);
  }

  function closeForm() {
    setCreating(false);
    setEditTarget(null);
    setFormError("");
  }

  function submitForm() {
    setFormError("");
    if (!form.name.trim()) {
      setFormError("Name is required");
      return;
    }
    const body = {
      description: form.description,
      included_tools: parseList(form.included_tools),
      included_servers: parseList(form.included_servers),
      excluded_tools: parseList(form.excluded_tools),
    };
    if (editTarget) {
      updateMutation.mutate({ name: editTarget.name, body });
    } else {
      createMutation.mutate({ name: form.name.trim(), ...body });
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending;
  const groups = groupsQuery.data ?? [];

  if (!isAdminEquivalent) {
    return (
      <EmptyState title="Admin only" message="Tool group management is restricted to admin-equivalent sessions." />
    );
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.22em] text-muted">Scoped views</p>
          <h2 className="mt-2 text-3xl font-semibold text-body">Tool Groups</h2>
        </div>
        <button
          className="rounded-md bg-accent px-4 py-3 text-sm font-semibold text-ink"
          onClick={openCreate}
        >
          New group
        </button>
      </div>

      {creating || editTarget ? (
        <GroupForm
          form={form}
          setForm={setForm}
          title={editTarget ? `Edit ${editTarget.name}` : "Create tool group"}
          onSubmit={submitForm}
          onCancel={closeForm}
          isPending={isPending}
          error={formError}
          isEdit={Boolean(editTarget)}
        />
      ) : null}

      {groups.length === 0 && !groupsQuery.isLoading ? (
        <EmptyState title="No groups yet" message="Create a group to scope which tools a client can see." />
      ) : (
        <div className="space-y-4">
          {groups.map((group) => (
            <div
              key={group.name}
              className="rounded-panel border border-line bg-panel p-5"
            >
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-3">
                    <h3 className="text-base font-semibold text-body">{group.name}</h3>
                  </div>
                  {group.description ? (
                    <p className="mt-1 text-sm text-muted">{group.description}</p>
                  ) : null}
                  <div className="mt-3 space-y-2">
                    {(group.included_tools ?? []).length > 0 ? (
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="text-xs text-muted">tools:</span>
                        {(group.included_tools ?? []).map((t) => (
                          <TokenChip key={t} value={t} />
                        ))}
                      </div>
                    ) : null}
                    {(group.included_servers ?? []).length > 0 ? (
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="text-xs text-muted">servers:</span>
                        {(group.included_servers ?? []).map((s) => (
                          <TokenChip key={s} value={s} />
                        ))}
                      </div>
                    ) : null}
                    {(group.excluded_tools ?? []).length > 0 ? (
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="text-xs text-down">excluded:</span>
                        {(group.excluded_tools ?? []).map((t) => (
                          <TokenChip key={t} value={t} />
                        ))}
                      </div>
                    ) : null}
                  </div>
                </div>
                <div className="flex shrink-0 gap-2">
                  <button
                    className="rounded-md border border-line px-3 py-1.5 text-xs text-body transition hover:border-accent hover:text-accent"
                    onClick={() =>
                      setPreviewGroup(previewGroup === group.name ? "" : group.name)
                    }
                  >
                    {previewGroup === group.name ? "Hide preview" : "Effective tools"}
                  </button>
                  <button
                    className="rounded-md border border-line px-3 py-1.5 text-xs text-body transition hover:border-accent hover:text-accent"
                    onClick={() => openEdit(group)}
                  >
                    Edit
                  </button>
                  <button
                    className="rounded-md border border-down/30 px-3 py-1.5 text-xs text-down transition hover:bg-down/10"
                    onClick={() => setPendingDelete(group.name)}
                  >
                    Delete
                  </button>
                </div>
              </div>

              {previewGroup === group.name ? (
                <div className="mt-4 border-t border-line pt-4">
                  <p className="mb-2 text-xs uppercase tracking-[0.18em] text-accent">Effective tools</p>
                  {effectiveQuery.isLoading ? (
                    <p className="text-xs text-muted">Loading…</p>
                  ) : (effectiveQuery.data ?? []).length > 0 ? (
                    <div className="flex flex-wrap gap-1.5">
                      {(effectiveQuery.data ?? []).map((t) => (
                        <TokenChip key={t} value={t} />
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-muted">No effective tools resolved for this group.</p>
                  )}
                </div>
              ) : null}
            </div>
          ))}
        </div>
      )}

      {pendingDelete ? (
        <ConfirmDialog
          title="Delete tool group"
          message={`Delete "${pendingDelete}"? This cannot be undone.`}
          confirmLabel="Delete"
          onConfirm={() => deleteMutation.mutate(pendingDelete)}
          onCancel={() => setPendingDelete("")}
        />
      ) : null}
    </div>
  );
}

type GroupFormProps = {
  form: GroupFormState;
  setForm: React.Dispatch<React.SetStateAction<GroupFormState>>;
  title: string;
  onSubmit: () => void;
  onCancel: () => void;
  isPending: boolean;
  error: string;
  isEdit: boolean;
};

function GroupForm({ form, setForm, title, onSubmit, onCancel, isPending, error, isEdit }: GroupFormProps) {
  function update(key: keyof GroupFormState, value: string) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  return (
    <div className="rounded-panel border border-accent/30 bg-panel p-6">
      <h3 className="text-base font-semibold text-body">{title}</h3>
      <div className="mt-4 grid gap-4 md:grid-cols-2">
        <Field label="Name">
          <input
            className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none disabled:opacity-50"
            value={form.name}
            disabled={isEdit}
            onChange={(e) => update("name", e.target.value)}
          />
        </Field>
        <Field label="Description">
          <input
            className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
            value={form.description}
            onChange={(e) => update("description", e.target.value)}
          />
        </Field>
        <Field label="Included tools (comma-separated)">
          <input
            className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
            placeholder="github__search_code, context7__*"
            value={form.included_tools}
            onChange={(e) => update("included_tools", e.target.value)}
          />
        </Field>
        <Field label="Included servers (comma-separated)">
          <input
            className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
            placeholder="github, context7"
            value={form.included_servers}
            onChange={(e) => update("included_servers", e.target.value)}
          />
        </Field>
        <Field label="Excluded tools (comma-separated)">
          <input
            className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
            placeholder="github__delete_repo"
            value={form.excluded_tools}
            onChange={(e) => update("excluded_tools", e.target.value)}
          />
        </Field>
      </div>
      {error ? <p className="mt-3 text-sm text-down">{error}</p> : null}
      <div className="mt-5 flex gap-3">
        <button
          className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink disabled:opacity-40"
          disabled={isPending}
          onClick={onSubmit}
        >
          {isPending ? "Saving…" : isEdit ? "Save changes" : "Create group"}
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
