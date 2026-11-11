# Dashboard guide

Read the [root guide](../../AGENTS.md) as well. This directory contains the development-mode dashboard, built with React, TypeScript, and Vite. Detailed workflow: [dashboard development](../../docs/developers/dashboard-development.mdx).

## Source map and conventions

- `src/App.tsx`: sections, tables, forms, and local UI state.
- `src/components/`: shared UI, including navigation and copy buttons.
- `src/lib/api.ts`: dashboard API helpers; `src/lib/types.ts`: frontend contracts.
- `src/styles.css`: shared styling; `vite.config.ts`: aliases, build output, and local API proxy.
- Follow existing React patterns and strict TypeScript settings. Reuse components and helpers before adding dependencies, a router, or a new state-management system.
- Keep the compact developer-oriented UI: restrained colors and borders, readable tables, monospace identifiers, nearby copy actions, and expandable details. Use structured fields before raw JSON; keep debug JSON collapsed.
- Preserve accessible labels, focus/keyboard behavior, loading/empty states, and visible mutation errors when adding interactions.

## Local development and embedding

Use npm and the committed lockfile. From this directory:

```bash
npm ci
npm run dev
```

Run the backend separately from the repository root:

```bash
bash ./scripts/build-dashboard.sh
go run . start --host 127.0.0.1
```

- Vite serves live source at `http://localhost:5173` and proxies `/api` to the backend at port 8080.
- The Go server at `http://localhost:8080/` serves compiled assets embedded into its binary; it does not read frontend source at runtime.
- After frontend changes, run `bash ./scripts/build-dashboard.sh` from the repository root and rebuild/restart Go. `npm run build` alone does not refresh `internal/dashboardui/dist`.
- Never hand-edit or commit `web/dashboard/dist`, `internal/dashboardui/dist`, or `node_modules`. If dependencies change, update `package.json` and `package-lock.json` together and install the updated dependencies.

## Backend contracts and security

- Dashboard routes live in `internal/api/server.go`; handlers are in `internal/api/dashboard*.go`, with read-side response shaping in `internal/service/dashboard/service.go`.
- Reuse existing service methods for mutations. Keep `src/lib/types.ts` and API helpers consistent with backend response contracts.
- Use relative `/api/dashboard/...` requests through existing API helpers. Preserve error handling, including the API's JSON `error` field.
- Dashboard UI and APIs intentionally return 404 in enterprise mode. Preserve that boundary when changing routes.
- Show sanitized summaries, such as environment/header key names, rather than secret values. Do not expose tokens, OAuth codes/client secrets, authorization headers, or raw sensitive server configuration in UI responses or diagnostics.
- Preserve the OAuth callback/polling flow; reuse the server-side registration and pending-session handling instead of moving credentials into browser storage.

## Validation

From this directory, run `npm run typecheck`. From the repository root, run:

```bash
bash ./scripts/build-dashboard.sh
go test ./internal/api ./internal/e2e
```

The build script also typechecks through `npm run build`. There are currently no frontend `test` or `lint` scripts; use the actual scripts in `package.json`.

For UI changes, inspect the affected interaction in the browser, including errors and empty states. Verify the embedded UI on port 8080 as well as Vite when the build/serving path is relevant. Backend tests alone do not verify visual or browser behavior. Run relevant MCP service tests for registration/OAuth changes and broader checks per the root guide; report any checks you could not run.
