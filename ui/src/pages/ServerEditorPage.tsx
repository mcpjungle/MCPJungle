import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { z } from "zod";

import { useAppContext } from "../App";
import { api } from "../lib/api";
import type { RegisterServerInput } from "../lib/types";

const formSchema = z
  .object({
    name: z.string().min(1, "Name is required"),
    transport: z.enum(["streamable_http", "stdio", "sse"]),
    description: z.string(),
    session_mode: z.enum(["stateless", "stateful"]),
    url: z.string().optional(),
    command: z.string().optional(),
    bearer_token: z.string().optional(),
  })
  .superRefine((value, ctx) => {
    if ((value.transport === "streamable_http" || value.transport === "sse") && !value.url) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: "URL is required for remote transports", path: ["url"] });
    }
    if (value.transport === "stdio" && !value.command) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: "Command is required for stdio", path: ["command"] });
    }
  });

const emptyForm: RegisterServerInput = {
  name: "",
  transport: "streamable_http",
  description: "",
  session_mode: "stateless",
};

export function ServerEditorPage() {
  const { token, isAdminEquivalent } = useAppContext();
  const { name } = useParams();
  const navigate = useNavigate();
  const isEdit = Boolean(name);

  const configsQuery = useQuery({
    queryKey: ["server-configs"],
    queryFn: () => api.serverConfigs(token || undefined),
    enabled: isAdminEquivalent,
  });

  const [form, setForm] = useState<RegisterServerInput>(emptyForm);
  const [headersText, setHeadersText] = useState("{}");
  const [argsText, setArgsText] = useState("");
  const [envText, setEnvText] = useState("{}");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!isEdit || !configsQuery.data) {
      return;
    }

    const matched = configsQuery.data.find((candidate) => candidate.name === name);
    if (!matched) {
      return;
    }

    setForm({
      ...matched,
      headers: matched.headers ?? {},
      env: matched.env ?? {},
      args: matched.args ?? [],
      session_mode: matched.session_mode ?? "stateless",
    });
    setHeadersText(JSON.stringify(matched.headers ?? {}, null, 2));
    setEnvText(JSON.stringify(matched.env ?? {}, null, 2));
    setArgsText((matched.args ?? []).join("\n"));
  }, [configsQuery.data, isEdit, name]);

  const mutation = useMutation({
    mutationFn: (payload: RegisterServerInput) => api.createServer(payload, token || undefined, isEdit),
    onSuccess: () => navigate("/servers"),
    onError: (err: Error) => setError(err.message),
  });

  const title = useMemo(() => (isEdit ? `Edit ${name}` : "Register server"), [isEdit, name]);

  function updateField<K extends keyof RegisterServerInput>(key: K, value: RegisterServerInput[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function handleSubmit() {
    setError("");

    try {
      const parsed = formSchema.parse({
        name: form.name,
        transport: form.transport,
        description: form.description,
        session_mode: form.session_mode ?? "stateless",
        url: form.url,
        command: form.command,
        bearer_token: form.bearer_token,
      });

      let headers: Record<string, string> | undefined;
      let env: Record<string, string> | undefined;

      if (headersText.trim() && headersText.trim() !== "{}") {
        try {
          headers = JSON.parse(headersText) as Record<string, string>;
        } catch {
          setError("Headers JSON is invalid");
          return;
        }
      }

      if (form.transport === "stdio" && envText.trim() && envText.trim() !== "{}") {
        try {
          env = JSON.parse(envText) as Record<string, string>;
        } catch {
          setError("Env JSON is invalid");
          return;
        }
      }

      const payload: RegisterServerInput = {
        ...form,
        ...parsed,
        headers,
        env,
        args: argsText
          .split("\n")
          .map((line) => line.trim())
          .filter(Boolean),
      };

      mutation.mutate(payload);
    } catch (caught) {
      if (caught instanceof Error && caught.message.includes("ZodError")) {
        try {
          const issues = JSON.parse(caught.message) as Array<{ path: string[]; message: string }>;
          setError(issues.map((i) => `${i.path.join(".")}: ${i.message}`).join("; "));
        } catch {
          setError(caught.message);
        }
      } else {
        setError(caught instanceof Error ? caught.message : "Unable to save server");
      }
    }
  }

  if (!isAdminEquivalent) {
    return <div className="rounded-panel border border-line bg-panel p-6 text-muted">Admin access required.</div>;
  }

  if (isEdit && configsQuery.isLoading) {
    return <div className="rounded-panel border border-line bg-panel p-6 text-sm text-muted">Loading server config…</div>;
  }

  if (isEdit && configsQuery.isError) {
    return (
      <div className="rounded-panel border border-down/30 bg-panel p-6 text-sm text-down">
        Failed to load server config: {(configsQuery.error as Error).message}
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div>
        <p className="text-xs uppercase tracking-[0.22em] text-muted">Server form</p>
        <h2 className="mt-2 text-3xl font-semibold text-body">{title}</h2>
      </div>
      <div className="grid gap-5 xl:grid-cols-2">
        <section className="rounded-panel border border-line bg-[#11161d] p-6">
          <div className="grid gap-4">
            <Field label="Name">
              <input
                className="w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                value={form.name}
                onChange={(event) => updateField("name", event.target.value)}
              />
            </Field>
            <Field label="Transport">
              <select
                className="w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                value={form.transport}
                onChange={(event) => updateField("transport", event.target.value)}
              >
                <option value="streamable_http">streamable_http</option>
                <option value="stdio">stdio</option>
                <option value="sse">sse</option>
              </select>
            </Field>
            <Field label="Description">
              <textarea
                className="min-h-28 w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                value={form.description}
                onChange={(event) => updateField("description", event.target.value)}
              />
            </Field>
            <Field label="Session mode">
              <select
                className="w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                value={form.session_mode}
                onChange={(event) => updateField("session_mode", event.target.value)}
              >
                <option value="stateless">stateless</option>
                <option value="stateful">stateful</option>
              </select>
            </Field>
            {form.transport === "stdio" ? (
              <>
                <Field label="Command">
                  <input
                    className="w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                    value={form.command ?? ""}
                    onChange={(event) => updateField("command", event.target.value)}
                  />
                </Field>
                <Field label="Args (one per line)">
                  <textarea
                    className="min-h-28 w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                    value={argsText}
                    onChange={(event) => setArgsText(event.target.value)}
                  />
                </Field>
                <Field label="Env JSON">
                  <textarea
                    className="min-h-40 w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                    value={envText}
                    onChange={(event) => setEnvText(event.target.value)}
                  />
                </Field>
              </>
            ) : (
              <>
                <Field label="URL">
                  <input
                    className="w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                    value={form.url ?? ""}
                    onChange={(event) => updateField("url", event.target.value)}
                  />
                </Field>
                <Field label="Bearer token">
                  <input
                    className="w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                    value={form.bearer_token ?? ""}
                    onChange={(event) => updateField("bearer_token", event.target.value)}
                  />
                </Field>
                {form.transport === "streamable_http" ? (
                  <Field label="Headers JSON">
                    <textarea
                      className="min-h-40 w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body"
                      value={headersText}
                      onChange={(event) => setHeadersText(event.target.value)}
                    />
                  </Field>
                ) : null}
              </>
            )}
            {error ? <p className="text-sm text-down">{error}</p> : null}
            <button className="rounded-md bg-accent px-4 py-3 text-sm font-semibold text-ink" onClick={handleSubmit}>
              {mutation.isPending ? "Saving..." : isEdit ? "Re-register server" : "Register server"}
            </button>
          </div>
        </section>
        <section className="rounded-panel border border-line bg-[#0f141b] p-6">
          <p className="text-sm font-semibold text-body">Form behavior</p>
          <ul className="mt-4 space-y-3 text-sm leading-6 text-muted">
            <li>Remote transports require URL. `streamable_http` may also forward custom headers.</li>
            <li>`stdio` expects command, optional args, and environment JSON.</li>
            <li>Edit route uses `force=true` and re-registers existing server name.</li>
          </ul>
        </section>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-2 block text-sm font-medium text-body">{label}</span>
      {children}
    </label>
  );
}
