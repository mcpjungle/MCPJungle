import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useAppContext } from "../App";
import { AllowListEditor } from "../components/forms/AllowListEditor";
import { ConfirmDialog } from "../components/ui/ConfirmDialog";
import { EmptyState } from "../components/ui/EmptyState";
import { api } from "../lib/api";
import type { CurrentUser, Group, McpServer } from "../lib/types";

type DrawerMode = "create" | "edit";

type GroupFormState = {
  name: string;
  description: string;
  allow_list: string[];
};

const emptyForm: GroupFormState = {
  name: "",
  description: "",
  allow_list: ["*"],
};

export function GroupsPage() {
  const { token, isAdminEquivalent, settings } = useAppContext();
  const queryClient = useQueryClient();

  const [drawerMode, setDrawerMode] = useState<DrawerMode | null>(null);
  const [drawerTarget, setDrawerTarget] = useState<Group | null>(null);
  const [form, setForm] = useState<GroupFormState>(emptyForm);
  const [formError, setFormError] = useState("");
  const [pendingDelete, setPendingDelete] = useState("");

  const { data: groups = [], refetch } = useQuery({
    queryKey: ["groups"],
    queryFn: () => api.groups(token || undefined),
    enabled: isAdminEquivalent && settings.mode !== "development",
  });

  const { data: users = [] } = useQuery<CurrentUser[]>({
    queryKey: ["users"],
    queryFn: () => api.users(token || undefined),
    enabled: isAdminEquivalent && settings.mode !== "development",
  });

  const { data: servers = [] } = useQuery<McpServer[]>({
    queryKey: ["servers"],
    queryFn: () => api.servers(token || undefined),
    enabled: isAdminEquivalent,
  });

  const createMutation = useMutation({
    mutationFn: (body: GroupFormState) => api.createGroup(body, token || undefined),
    onSuccess: () => {
      void refetch();
      void queryClient.invalidateQueries({ queryKey: ["users"] });
      closeDrawer();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({ name, ...body }: GroupFormState & { name: string }) =>
      api.updateGroup(name, { description: body.description, allow_list: body.allow_list, update_allow_list: true }, token || undefined),
    onSuccess: () => {
      void refetch();
      void queryClient.invalidateQueries({ queryKey: ["users"] });
      closeDrawer();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => api.deleteGroup(name, token || undefined),
    onSuccess: () => {
      setPendingDelete("");
      void refetch();
      void queryClient.invalidateQueries({ queryKey: ["users"] });
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const assignMutation = useMutation({
    mutationFn: ({ groupName, username }: { groupName: string; username: string }) =>
      api.assignUserToGroup(groupName, username, token || undefined),
    onSuccess: () => {
      void refetch();
      void queryClient.invalidateQueries({ queryKey: ["users"] });
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const removeMemberMutation = useMutation({
    mutationFn: ({ groupName, username }: { groupName: string; username: string }) =>
      api.removeUserFromGroup(groupName, username, token || undefined),
    onSuccess: () => {
      void refetch();
      void queryClient.invalidateQueries({ queryKey: ["users"] });
    },
    onError: (err: Error) => setFormError(err.message),
  });

  function openCreate() {
    setForm(emptyForm);
    setFormError("");
    setDrawerTarget(null);
    setDrawerMode("create");
  }

  function openEdit(group: Group) {
    setForm({ name: group.name, description: group.description, allow_list: group.allow_list });
    setFormError("");
    setDrawerTarget(group);
    setDrawerMode("edit");
  }

  function closeDrawer() {
    setDrawerMode(null);
    setDrawerTarget(null);
    setFormError("");
  }

  function submitForm() {
    setFormError("");
    if (drawerMode === "create") {
      if (!form.name.trim()) {
        setFormError("Name is required");
        return;
      }
      createMutation.mutate({ name: form.name.trim(), description: form.description, allow_list: form.allow_list });
    } else if (drawerTarget) {
      updateMutation.mutate({ name: drawerTarget.name, description: form.description, allow_list: form.allow_list });
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending;

  // Members of the currently edited group
  const editGroupMembers = drawerTarget
    ? users.filter((u) => u.group_name === drawerTarget.name)
    : [];

  // Users not in this group (available to add)
  const availableUsers = drawerTarget
    ? users.filter((u) => u.group_name !== drawerTarget.name)
    : [];

  if (!isAdminEquivalent || settings.mode === "development") {
    return (
      <div className="rounded-panel border border-line bg-panel p-8">
        <p className="text-xs font-medium uppercase tracking-widest text-muted">Groups</p>
        <h2 className="mt-3 text-xl font-semibold text-body">Enterprise mode required</h2>
        <p className="mt-3 max-w-lg text-sm leading-6 text-muted">
          Group management is only available when the gateway is running in{" "}
          <span className="font-medium text-body">enterprise</span> or{" "}
          <span className="font-medium text-body">production</span> mode.
        </p>
        <p className="mt-4 text-sm leading-6 text-muted">
          You are currently in{" "}
          <span className="rounded bg-accent/15 px-1.5 py-0.5 font-mono text-xs text-accent">
            {settings.mode}
          </span>{" "}
          mode. To enable group management, reinitialize the gateway and select{" "}
          <span className="font-medium text-body">Enterprise</span> on the setup screen.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.22em] text-muted">Access management</p>
          <h2 className="mt-2 text-3xl font-semibold text-body">Groups</h2>
        </div>
        <button
          className="rounded-md bg-accent px-4 py-3 text-sm font-semibold text-ink"
          onClick={openCreate}
        >
          New group
        </button>
      </div>

      {/* Create drawer */}
      {drawerMode === "create" ? (
        <div className="rounded-panel border border-accent/30 bg-panel p-6">
          <h3 className="text-base font-semibold text-body">Create group</h3>
          <div className="mt-4 space-y-4">
            <Field label="Name">
              <input
                className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
                value={form.name}
                onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
              />
            </Field>
            <Field label="Description (optional)">
              <input
                className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
                value={form.description}
                onChange={(e) => setForm((p) => ({ ...p, description: e.target.value }))}
              />
            </Field>
            <Field label="Server access">
              <AllowListEditor
                value={form.allow_list}
                servers={servers}
                onChange={(v) => setForm((p) => ({ ...p, allow_list: v }))}
              />
            </Field>
          </div>
          {formError ? <p className="mt-3 text-sm text-down">{formError}</p> : null}
          <div className="mt-5 flex gap-3">
            <button
              className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink disabled:opacity-40"
              disabled={isPending}
              onClick={submitForm}
            >
              {isPending ? "Creating…" : "Create group"}
            </button>
            <button className="rounded-md border border-line px-4 py-2 text-sm text-body" onClick={closeDrawer}>
              Cancel
            </button>
          </div>
        </div>
      ) : null}

      {/* Edit drawer */}
      {drawerMode === "edit" && drawerTarget ? (
        <div className="rounded-panel border border-accent/30 bg-panel p-6">
          <h3 className="text-base font-semibold text-body">
            Edit group <span className="text-accent">{drawerTarget.name}</span>
          </h3>
          <div className="mt-4 space-y-4">
            <Field label="Name">
              <input
                className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body opacity-50 cursor-not-allowed"
                value={drawerTarget.name}
                readOnly
              />
            </Field>
            <Field label="Description (optional)">
              <input
                className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
                value={form.description}
                onChange={(e) => setForm((p) => ({ ...p, description: e.target.value }))}
              />
            </Field>
            <Field label="Server access">
              <AllowListEditor
                value={form.allow_list}
                servers={servers}
                onChange={(v) => setForm((p) => ({ ...p, allow_list: v }))}
              />
            </Field>
          </div>
          {formError ? <p className="mt-3 text-sm text-down">{formError}</p> : null}
          <div className="mt-5 flex gap-3">
            <button
              className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink disabled:opacity-40"
              disabled={isPending}
              onClick={submitForm}
            >
              {isPending ? "Saving…" : "Save changes"}
            </button>
            <button className="rounded-md border border-line px-4 py-2 text-sm text-body" onClick={closeDrawer}>
              Cancel
            </button>
          </div>

          {/* Members sub-section */}
          <div className="mt-8 border-t border-line pt-6">
            <h4 className="text-sm font-semibold text-body">Members</h4>
            <p className="mt-1 text-xs text-muted">Users assigned to this group inherit its server access list.</p>

            {editGroupMembers.length === 0 ? (
              <p className="mt-3 text-xs text-muted italic">No members yet.</p>
            ) : (
              <div className="mt-3 space-y-2">
                {editGroupMembers.map((member) => {
                  const hasOverride =
                    member.allow_list !== undefined &&
                    !(member.allow_list.length === 1 && member.allow_list[0] === "*") &&
                    JSON.stringify(member.allow_list) !== JSON.stringify(drawerTarget.allow_list);

                  return (
                    <div key={member.username} className="flex items-center gap-3 rounded-ui border border-line bg-shell px-3 py-2">
                      <span className="font-mono text-xs text-body">{member.username}</span>
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          member.role === "admin" ? "bg-accent/10 text-accent" : "bg-elevated text-muted"
                        }`}
                      >
                        {member.role}
                      </span>
                      {hasOverride ? (
                        <span className="rounded-full bg-yellow-500/10 px-2 py-0.5 text-xs font-medium text-yellow-400">
                          Has explicit override
                        </span>
                      ) : null}
                      <button
                        className="ml-auto rounded-md border border-down/30 px-2.5 py-1 text-xs text-down transition hover:bg-down/10"
                        onClick={() =>
                          removeMemberMutation.mutate({ groupName: drawerTarget.name, username: member.username })
                        }
                        disabled={removeMemberMutation.isPending}
                      >
                        Remove
                      </button>
                    </div>
                  );
                })}
              </div>
            )}

            {availableUsers.length > 0 ? (
              <div className="mt-4">
                <label className="block">
                  <span className="mb-1.5 block text-xs font-medium text-muted">Add member</span>
                  <select
                    className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
                    defaultValue=""
                    onChange={(e) => {
                      const username = e.target.value;
                      if (!username) return;
                      assignMutation.mutate({ groupName: drawerTarget.name, username });
                      e.target.value = "";
                    }}
                  >
                    <option value="" disabled>Select a user to add…</option>
                    {availableUsers.map((u) => (
                      <option key={u.username} value={u.username}>
                        {u.username} ({u.role})
                      </option>
                    ))}
                  </select>
                </label>
              </div>
            ) : null}
          </div>
        </div>
      ) : null}

      {groups.length === 0 ? (
        <EmptyState
          title="No groups yet"
          message="Create your first group to manage server access for teams."
        />
      ) : (
        <div className="rounded-panel border border-line bg-panel overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-line">
                <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Name
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Description
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Server Access
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Members
                </th>
                <th className="px-5 py-3 text-right text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {groups.map((group, idx) => (
                <tr key={group.name} className={idx < groups.length - 1 ? "border-b border-line" : ""}>
                  <td className="px-5 py-3 font-medium text-body font-mono text-sm">{group.name}</td>
                  <td className="px-5 py-3 text-muted text-sm">
                    {group.description || <span className="italic text-muted/50">—</span>}
                  </td>
                  <td className="px-5 py-3">
                    <AllowListBadges allowList={group.allow_list} />
                  </td>
                  <td className="px-5 py-3">
                    <span className="rounded-full bg-elevated px-2.5 py-0.5 text-xs font-medium text-body">
                      {group.member_count}
                    </span>
                  </td>
                  <td className="px-5 py-3 text-right">
                    <div className="flex justify-end gap-2">
                      <button
                        className="rounded-md border border-line px-2.5 py-1 text-xs text-body transition hover:border-accent hover:text-accent"
                        onClick={() => openEdit(group)}
                      >
                        Edit
                      </button>
                      <button
                        className="rounded-md border border-down/30 px-2.5 py-1 text-xs text-down transition hover:bg-down/10"
                        onClick={() => setPendingDelete(group.name)}
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {pendingDelete ? (
        <ConfirmDialog
          title="Delete group"
          message={`Delete group "${pendingDelete}"? Members will be unassigned from this group.`}
          confirmLabel="Delete"
          onConfirm={() => deleteMutation.mutate(pendingDelete)}
          onCancel={() => {
            setPendingDelete("");
            setFormError("");
          }}
        />
      ) : null}

      {deleteMutation.isError ? (
        <p className="text-sm text-down">{(deleteMutation.error as Error).message}</p>
      ) : null}
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

function AllowListBadges({ allowList }: { allowList: string[] }) {
  if (!allowList || (allowList.length === 1 && allowList[0] === "*")) {
    return <span className="rounded-full bg-up/10 px-2 py-0.5 text-xs font-medium text-up">All servers</span>;
  }
  if (allowList.length === 0) {
    return <span className="rounded-full bg-down/10 px-2 py-0.5 text-xs font-medium text-down">No access</span>;
  }
  return (
    <div className="flex flex-wrap gap-1">
      {allowList.slice(0, 3).map((s) => (
        <span key={s} className="rounded-full bg-elevated px-2 py-0.5 text-xs text-body">
          {s}
        </span>
      ))}
      {allowList.length > 3 && (
        <span className="rounded-full bg-elevated px-2 py-0.5 text-xs text-muted">
          +{allowList.length - 3} more
        </span>
      )}
    </div>
  );
}
