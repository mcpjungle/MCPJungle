import { FormEvent, useMemo, useState } from "react";

import type { ToolInputSchema } from "../../lib/types";

type JSONSchemaFormProps = {
  schema: ToolInputSchema;
  onSubmit: (values: Record<string, unknown>) => void;
  submitLabel?: string;
};

export function JSONSchemaForm({ schema, onSubmit, submitLabel = "Run tool" }: JSONSchemaFormProps) {
  const properties = useMemo(() => schema.properties ?? {}, [schema.properties]);
  const required = schema.required ?? [];
  const [formState, setFormState] = useState<Record<string, unknown>>({});

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSubmit(formState);
  }

  if (!Object.keys(properties).length) {
    return (
      <button className="rounded-md bg-accent px-4 py-3 text-sm font-semibold text-ink" onClick={() => onSubmit({})}>
        {submitLabel}
      </button>
    );
  }

  return (
    <form className="space-y-4" onSubmit={handleSubmit}>
      {Object.entries(properties).map(([key, property]) => {
        const type = property.type ?? "string";
        const isRequired = required.includes(key);

        if (type === "boolean") {
          return (
            <label className="flex items-center gap-3 text-sm text-body" key={key}>
              <input
                type="checkbox"
                checked={Boolean(formState[key])}
                onChange={(event) => setFormState((current) => ({ ...current, [key]: event.target.checked }))}
              />
              <span>{property.description ?? key}</span>
            </label>
          );
        }

        return (
          <label className="block" key={key}>
            <span className="mb-2 block text-sm font-medium text-body">
              {key}
              {isRequired ? <span className="ml-2 text-accent">*</span> : null}
            </span>
            <input
              className="w-full rounded-ui border border-line bg-shell px-4 py-3 text-sm text-body outline-none transition focus:border-accent"
              type={type === "number" || type === "integer" ? "number" : "text"}
              value={String(formState[key] ?? "")}
              onChange={(event) =>
                setFormState((current) => ({
                  ...current,
                  [key]:
                    type === "number" || type === "integer" ? Number(event.target.value || 0) : event.target.value,
                }))
              }
              placeholder={property.description ?? key}
            />
          </label>
        );
      })}

      <button className="rounded-md bg-accent px-4 py-3 text-sm font-semibold text-ink" type="submit">
        {submitLabel}
      </button>
    </form>
  );
}
