import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useAppContext } from "../App";
import { ConfirmDialog } from "../components/ui/ConfirmDialog";
import { EmptyState } from "../components/ui/EmptyState";
import { api } from "../lib/api";
import type { CreateOrUpdateUserResponse, CurrentUser, McpServer } from "../lib/types";

type DrawerMode = "create" | "rotate" | "permissions";

type CreateFormState = {
  username: string;
  access_token: string;
  allow_list: string[];
};

type RotateFormState = {
  use_custom: boolean;
  access_token: string;
};

export function UsersPage() {
  const { token, isAdminEquivalent, settings } = useAppContext();
  const queryClient = useQueryClient();

  const [drawerMode, setDrawerMode] = useState<DrawerMode | null>(null);
  const [drawerTarget, setDrawerTarget] = useState<CurrentUser | null>(null);
  const [createForm, setCreateForm] = useState<CreateFormState>({ username: "", access_token: "", allow_list: ["*"] });
  const [rotateForm, setRotateForm] = useState<RotateFormState>({ use_custom: false, access_token: "" });
  const [permAllowList, setPermAllowList] = useState<string[]>(["*"]);
  const [formError, setFormError] = useState("");
  const [pendingDelete, setPendingDelete] = useState("");
  const [revealedTokens, setRevealedTokens] = useState<Record<string, string>>({});

  const usersQuery = useQuery({
    queryKey: ["users"],
    queryFn: () => api.users(token || undefined),
    enabled: isAdminEquivalent && settings.mode !== "development",
  });

  const serversQuery = useQuery({
    queryKey: ["servers"],
    queryFn: () => api.servers(token || undefined),
    enabled: isAdminEquivalent,
  });

  const createMutation = useMutation({
    mutationFn: (body: { username: string; access_token?: string; allow_list?: string[] }) =>
      api.createUser(body, token || undefined),
    onSuccess: (result: CreateOrUpdateUserResponse) => {
      void queryClient.invalidateQueries({ queryKey: ["users"] });
      setRevealedTokens((prev) => ({ ...prev, [result.username]: result.access_token }));
      closeDrawer();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const rotateMutation = useMutation({
    mutationFn: ({
      username,
      access_token,
      rotate_access_token,
    }: {
      username: string;
      access_token?: string;
      rotate_access_token?: boolean;
    }) => api.updateUser(username, { access_token, rotate_access_token }, token || undefined),
    onSuccess: (result: CreateOrUpdateUserResponse) => {
      void queryClient.invalidateQueries({ queryKey: ["users"] });
      setRevealedTokens((prev) => ({ ...prev, [result.username]: result.access_token }));
      closeDrawer();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const permMutation = useMutation({
    mutationFn: ({ username, allow_list }: { username: string; allow_list: string[] }) =>
      api.updateUser(username, { allow_list, update_allow_list: true }, token || undefined),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["users"] });
      closeDrawer();
    },
    onError: (err: Error) => setFormError(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (username: string) => api.deleteUser(username, token || undefined),
    onSuccess: (_, username) => {
      setPendingDelete("");
      void queryClient.invalidateQueries({ queryKey: ["users"] });
      setRevealedTokens((prev) => {
        const next = { ...prev };
        delete next[username];
        return next;
      });
    },
    onError: (err: Error) => setFormError(err.message),
  });

  function openCreate() {
    setCreateForm({ username: "", access_token: "", allow_list: ["*"] });
    setFormError("");
    setDrawerTarget(null);
    setDrawerMode("create");
  }

  function openRotate(user: CurrentUser) {
    setRotateForm({ use_custom: false, access_token: "" });
    setFormError("");
    setDrawerTarget(user);
    setDrawerMode("rotate");
  }

  function openPermissions(user: CurrentUser) {
    // Server always sends allow_list now. undefined = old record with no value set (treat as wildcard).
    // [] = explicitly no access. Never coerce [] to ["*"].
    setPermAllowList(user.allow_list === undefined ? ["*"] : user.allow_list);
    setFormError("");
    setDrawerTarget(user);
    setDrawerMode("permissions");
  }

  function closeDrawer() {
    setDrawerMode(null);
    setDrawerTarget(null);
    setFormError("");
  }

  function submitCreate() {
    setFormError("");
    if (!createForm.username.trim()) {
      setFormError("Username is required");
      return;
    }
    createMutation.mutate({
      username: createForm.username.trim(),
      access_token: createForm.access_token.trim() || undefined,
      allow_list: createForm.allow_list,
    });
  }

  function submitRotate() {
    setFormError("");
    if (!drawerTarget) return;
    if (rotateForm.use_custom && !rotateForm.access_token.trim()) {
      setFormError("Enter a custom token or uncheck custom.");
      return;
    }
    if (rotateForm.use_custom) {
      rotateMutation.mutate({ username: drawerTarget.username, access_token: rotateForm.access_token.trim() });
    } else {
      rotateMutation.mutate({ username: drawerTarget.username, rotate_access_token: true });
    }
  }

  function submitPermissions() {
    if (!drawerTarget) return;
    permMutation.mutate({ username: drawerTarget.username, allow_list: permAllowList });
  }

  const isPending = createMutation.isPending || rotateMutation.isPending || permMutation.isPending;
  const users = usersQuery.data ?? [];
  const servers: McpServer[] = serversQuery.data ?? [];

  if (!isAdminEquivalent || settings.mode === "development") {
    return (
      <div className="rounded-panel border border-line bg-panel p-8">
        <p className="text-xs font-medium uppercase tracking-widest text-muted">Users</p>
        <h2 className="mt-3 text-xl font-semibold text-body">Enterprise mode required</h2>
        <p className="mt-3 max-w-lg text-sm leading-6 text-muted">
          Human user accounts are only available when the gateway is running in{" "}
          <span className="font-medium text-body">enterprise</span> or{" "}
          <span className="font-medium text-body">production</span> mode. In those modes, each
          human operator gets their own Bearer token and role (admin or standard user).
        </p>
        <p className="mt-4 text-sm leading-6 text-muted">
          You are currently in{" "}
          <span className="rounded bg-accent/15 px-1.5 py-0.5 font-mono text-xs text-accent">
            {settings.mode}
          </span>{" "}
          mode, which requires no credentials. To enable user management, reinitialize the
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
          <p className="text-xs uppercase tracking-[0.22em] text-muted">Enterprise accounts</p>
          <h2 className="mt-2 text-3xl font-semibold text-body">Users</h2>
        </div>
        <button
          className="rounded-md bg-accent px-4 py-3 text-sm font-semibold text-ink"
          onClick={openCreate}
        >
          New user
        </button>
      </div>

      {/* Create drawer */}
      {drawerMode === "create" ? (
        <div className="rounded-panel border border-accent/30 bg-panel p-6">
          <h3 className="text-base font-semibold text-body">Create user</h3>
          <div className="mt-4 space-y-4">
            <Field label="Username">
              <input
                className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
                value={createForm.username}
                onChange={(e) => setCreateForm((p) => ({ ...p, username: e.target.value }))}
              />
            </Field>
            <Field label="Access token (leave blank to auto-generate)">
              <input
                className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
                placeholder="Leave blank to auto-generate"
                value={createForm.access_token}
                onChange={(e) => setCreateForm((p) => ({ ...p, access_token: e.target.value }))}
              />
            </Field>
            <Field label="Server access (MCP client permissions)">
              <AllowListEditor
                value={createForm.allow_list}
                servers={servers}
                onChange={(v) => setCreateForm((p) => ({ ...p, allow_list: v }))}
              />
            </Field>
          </div>
          {formError ? <p className="mt-3 text-sm text-down">{formError}</p> : null}
          <div className="mt-5 flex gap-3">
            <button
              className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink disabled:opacity-40"
              disabled={isPending}
              onClick={submitCreate}
            >
              {isPending ? "Creating…" : "Create user"}
            </button>
            <button className="rounded-md border border-line px-4 py-2 text-sm text-body" onClick={closeDrawer}>
              Cancel
            </button>
          </div>
        </div>
      ) : null}

      {/* Rotate token drawer */}
      {drawerMode === "rotate" && drawerTarget ? (
        <div className="rounded-panel border border-accent/30 bg-panel p-6">
          <h3 className="text-base font-semibold text-body">Rotate token for {drawerTarget.username}</h3>
          <div className="mt-4 space-y-4">
            <label className="flex items-center gap-2 text-sm text-body">
              <input
                type="checkbox"
                checked={rotateForm.use_custom}
                onChange={(e) => {
                  setRotateForm((p) => ({ ...p, use_custom: e.target.checked, access_token: "" }));
                }}
              />
              Set a custom token instead
            </label>
            {rotateForm.use_custom ? (
              <Field label="Custom access token">
                <input
                  className="w-full rounded-ui border border-line bg-shell px-3 py-2 text-sm text-body focus:border-accent focus:outline-none"
                  value={rotateForm.access_token}
                  onChange={(e) => setRotateForm((p) => ({ ...p, access_token: e.target.value }))}
                />
              </Field>
            ) : (
              <p className="text-sm text-muted">A new token will be auto-generated.</p>
            )}
          </div>
          {formError ? <p className="mt-3 text-sm text-down">{formError}</p> : null}
          <div className="mt-5 flex gap-3">
            <button
              className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink disabled:opacity-40"
              disabled={isPending}
              onClick={submitRotate}
            >
              {isPending ? "Rotating…" : "Rotate token"}
            </button>
            <button className="rounded-md border border-line px-4 py-2 text-sm text-body" onClick={closeDrawer}>
              Cancel
            </button>
          </div>
        </div>
      ) : null}

      {/* Edit permissions drawer */}
      {drawerMode === "permissions" && drawerTarget ? (
        <div className="rounded-panel border border-accent/30 bg-panel p-6">
          <h3 className="text-base font-semibold text-body">
            Server access for <span className="text-accent">{drawerTarget.username}</span>
          </h3>
          <p className="mt-1 text-xs text-muted">
            Controls which MCP servers this user&apos;s self-created client tokens can access.
          </p>
          <div className="mt-4">
            <AllowListEditor
              value={permAllowList}
              servers={servers}
              onChange={setPermAllowList}
            />
          </div>
          {formError ? <p className="mt-3 text-sm text-down">{formError}</p> : null}
          <div className="mt-5 flex gap-3">
            <button
              className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-ink disabled:opacity-40"
              disabled={isPending}
              onClick={submitPermissions}
            >
              {isPending ? "Saving…" : "Save permissions"}
            </button>
            <button className="rounded-md border border-line px-4 py-2 text-sm text-body" onClick={closeDrawer}>
              Cancel
            </button>
          </div>
        </div>
      ) : null}

      {users.length === 0 && !usersQuery.isLoading ? (
        <EmptyState title="No users" message="Create standard users to grant enterprise access." />
      ) : (
        <div className="rounded-panel border border-line bg-panel overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-line">
                <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Username
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Role
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Server Access
                </th>
                <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Token
                </th>
                <th className="px-5 py-3 text-right text-xs font-medium uppercase tracking-[0.18em] text-muted">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {users.map((user, idx) => (
                <tr
                  key={user.username}
                  className={idx < users.length - 1 ? "border-b border-line" : ""}
                >
                  <td className="px-5 py-3 font-medium text-body">{user.username}</td>
                  <td className="px-5 py-3">
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                        user.role === "admin"
                          ? "bg-accent/10 text-accent"
                          : "bg-elevated text-muted"
                      }`}
                    >
                      {user.role}
                    </span>
                  </td>
                  <td className="px-5 py-3">
                    <AllowListBadges allowList={user.allow_list ?? []} />
                  </td>
                  <td className="max-w-xs px-5 py-3">
                    {revealedTokens[user.username] ? (
                      <div className="flex items-center gap-2">
                        <code className="truncate font-mono text-xs text-body">
                          {revealedTokens[user.username]}
                        </code>
                        <button
                          className="shrink-0 text-xs text-muted underline"
                          onClick={() =>
                            setRevealedTokens((prev) => {
                              const next = { ...prev };
                              delete next[user.username];
                              return next;
                            })
                          }
                        >
                          Dismiss
                        </button>
                      </div>
                    ) : (
                      <span className="text-xs text-muted">••••••••</span>
                    )}
                  </td>
                  <td className="px-5 py-3 text-right">
                    <div className="flex justify-end gap-2">
                      <button
                        className="rounded-md border border-line px-2.5 py-1 text-xs text-body transition hover:border-accent hover:text-accent"
                        onClick={() => openPermissions(user)}
                      >
                        Permissions
                      </button>
                      <button
                        className="rounded-md border border-line px-2.5 py-1 text-xs text-body transition hover:border-accent hover:text-accent"
                        onClick={() => openRotate(user)}
                      >
                        Rotate token
                      </button>
                      {user.role !== "admin" ? (
                        <button
                          className="rounded-md border border-down/30 px-2.5 py-1 text-xs text-down transition hover:bg-down/10"
                          onClick={() => setPendingDelete(user.username)}
                        >
                          Delete
                        </button>
                      ) : null}
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
          title="Delete user"
          message={`Delete user "${pendingDelete}"? Their access token will be immediately revoked.`}
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

/** Shows an allow list as compact badges */
function AllowListBadges({ allowList }: { allowList: string[] }) {
  // null/undefined → wildcard (unset)
  if (!allowList) {
    return <span className="rounded-full bg-up/10 px-2 py-0.5 text-xs font-medium text-up">All servers</span>;
  }
  // ["*"] → wildcard
  if (allowList.length === 1 && allowList[0] === "*") {
    return <span className="rounded-full bg-up/10 px-2 py-0.5 text-xs font-medium text-up">All servers</span>;
  }
  // [] → explicitly no access
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

/**
 * AllowListEditor — lets admin set wildcard or a specific set of server names.
 * If servers list is available, shows checkboxes; otherwise falls back to a text input.
 */
function AllowListEditor({
  value,
  servers,
  onChange,
}: {
  value: string[];
  servers: McpServer[];
  onChange: (v: string[]) => void;
}) {
  const isWildcard = value.length === 1 && value[0] === "*";

  function toggleWildcard(toWildcard: boolean) {
    onChange(toWildcard ? ["*"] : []);
  }

  function toggleServer(name: string) {
    if (value.includes(name)) {
      const next = value.filter((s) => s !== name);
      onChange(next.length === 0 ? [] : next);
    } else {
      onChange([...value.filter((s) => s !== "*"), name]);
    }
  }

  return (
    <div className="space-y-3">
      <label className="flex items-center gap-2 text-sm text-body cursor-pointer select-none">
        <input
          type="checkbox"
          checked={isWildcard}
          onChange={(e) => toggleWildcard(e.target.checked)}
          className="accent-yellow-400"
        />
        <span>Allow access to <strong>all servers</strong> (wildcard)</span>
      </label>

      {!isWildcard && servers.length > 0 && (
        <div className="ml-1 rounded-ui border border-line bg-shell p-3">
          <p className="mb-2 text-xs text-muted">Select individual servers:</p>
          <div className="space-y-1.5 max-h-48 overflow-y-auto">
            {servers.map((s) => (
              <label key={s.name} className="flex items-center gap-2 text-sm text-body cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={value.includes(s.name)}
                  onChange={() => toggleServer(s.name)}
                  className="accent-yellow-400"
                />
                <span className="font-mono text-xs">{s.name}</span>
                {s.description && (
                  <span className="text-xs text-muted truncate">— {s.description}</span>
                )}
              </label>
            ))}
          </div>
          {value.length === 0 && (
            <p className="mt-2 text-xs text-down">No servers selected — user will have no access.</p>
          )}
        </div>
      )}

      {!isWildcard && servers.length === 0 && (
        <div className="ml-1 rounded-ui border border-line bg-shell p-3">
          <p className="text-xs text-muted">No servers registered yet. User will get wildcard access by default once servers are added.</p>
        </div>
      )}

      <p className="text-xs text-muted">
        {isWildcard
          ? "When this user creates an MCP client token, it will inherit access to all current and future servers."
          : value.length > 0
          ? `Client tokens created by this user will only be able to access: ${value.join(", ")}`
          : "Client tokens created by this user will have no server access."}
      </p>
    </div>
  );
}
