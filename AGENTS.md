# MCPJungle agent guide

## Project and scope

MCPJungle is a self-hosted MCP registry and gateway. One Go binary runs the HTTP server and the management CLI. The gateway exposes upstream tools, prompts, and resources through Streamable HTTP and legacy SSE; upstream connections also support STDIO. SQLite is the local default; PostgreSQL is supported for shared deployments.

Read [docs/AGENTS.md](docs/AGENTS.md) before changing documentation and [web/dashboard/AGENTS.md](web/dashboard/AGENTS.md) before changing the frontend. Those files supplement this guide in their directories. Explicit task instructions take precedence over repository guidance.

## Working principles

- Prefer the smallest change that solves the problem without breaking existing behavior. Reuse existing services and helpers before adding abstractions or dependencies.
- Inspect the working tree first and preserve unrelated changes. Keep fixes, tests, and documentation focused on the requested work.
- Read the affected implementation and nearby tests before editing. Follow local conventions rather than imposing a repository-wide cleanup.
- Format changed Go files with `gofmt`; use [.golangci.yml](.golangci.yml) for the repository's lint and formatter configuration. Avoid formatting unrelated files.
- Use local documentation for project context. Cross-check implementation, test assertions, and docs when they disagree; report material discrepancies and correct relevant docs within the task's scope.
- Keep these instructions durable: use repository-relative paths, link to detailed guides, and update commands when workflows change. Keep task logs, machine-specific settings, and roadmap proposals elsewhere.

## Code map

| Path | Responsibility |
| --- | --- |
| `main.go`, `cmd/` | Cobra commands, CLI workflows, server startup and dependency wiring |
| `client/` | HTTP management client used by the CLI |
| `internal/api/` | Gin routes, authentication middleware, request binding, HTTP responses |
| `internal/service/mcp/` | Registry lifecycle, MCP proxy handlers, upstream connections, sessions, OAuth |
| `internal/service/toolgroup/` | Group resolution and group MCP proxies |
| `internal/service/dashboard/` | Sanitized dashboard response shaping |
| `internal/model/`, `internal/db/`, `internal/migrations/` | GORM models, SQLite/PostgreSQL connections, startup migrations |
| `pkg/types/`, `pkg/apierrors/` | Shared API contracts and service error classification |
| `internal/configresolver/` | Configuration environment-placeholder expansion |
| `internal/telemetry/`, `pkg/version/` | Metrics and advertised build version |
| `internal/e2e/`, `pkg/testhelpers/` | End-to-end tests and shared test setup |
| `web/dashboard/`, `internal/dashboardui/` | Frontend source and Go asset embedding |
| `docs/` | Mintlify documentation source |

A typical management change crosses `cmd/` → `client/` → `internal/api/` → `internal/service/` → persistence or an upstream server. Follow the affected flow; update shared types and dashboard types when its contract changes. Keep business logic in existing services so CLI and dashboard workflows share behavior.

## Setup and commands

Use the Go version required by [go.mod](go.mod). Frontend setup needs Node.js and npm; use the committed `web/dashboard/package-lock.json`. Some Go tests use the CGO SQLite driver and need a C compiler even though production SQLite uses a different driver.

**Fresh clones need the generated dashboard bundle before Go builds or tests.** `internal/dashboardui/embed.go` embeds `internal/dashboardui/dist` at compile time. From the repository root:

```bash
bash ./scripts/build-dashboard.sh
go build ./...
go run . start --host 127.0.0.1
```

The dashboard script runs `npm ci` if `node_modules` is missing, builds the frontend, and copies its output into the embedded asset directory. Rebuild and restart the Go process after frontend changes. Do not hand-edit generated bundles or commit `dist/`, `node_modules/`, `vendor/`, or local binaries.

| Check | Command from repository root |
| --- | --- |
| Focused Go tests | `go test ./internal/service/mcp` (substitute the affected package) |
| Full Go suite, including E2E | `go test ./...` |
| Static analysis | `go vet ./...` |
| Repository lint | `golangci-lint run` |
| CLI integration checks | `bash ./scripts/test-mcpjungle.sh` |
| HTTP error-contract checks | `bash ./scripts/test-api-error-responses.sh` |

- Start with focused checks; run broader checks before handing off substantive code changes. See the verification guidance below and the PR template for submission requirements.
- E2E tests launch local HTTP servers and STDIO upstreams using `npx` and `@modelcontextprotocol/server-everything`. Tests using the shared E2E setup skip if `npx` is absent; package downloads may need network access.
- The CLI integration script needs `npx` and external connectivity, including Context7. Distinguish environment or upstream failures from regressions; report skipped or blocked checks.
- Use temporary databases and isolated test configuration for manual checks. The CLI can persist credentials in `~/.mcpjungle.conf`; do not overwrite an operator's existing configuration or use their database for tests.
- `go.mod` and `go.sum` own Go dependencies. Run `go mod tidy` after dependency changes. If an ignored local `vendor/` exists, refresh it with `go mod vendor` or use module-aware mode to avoid stale dependencies.

## Behavior to preserve

- **Modes and identities:** Development mode initializes automatically; enterprise mode requires initialization and authenticated access. Human users and MCP clients have different permissions. Preserve existing defaults, compatibility aliases, and initialization rules unless the task changes them.
- **Authorization:** Enforce MCP client server allow-lists during invocation as well as discovery, on global and group endpoints. Groups must not grant additional server permissions. Preserve the request context supplied by API middleware and the fail-closed behavior of `ProxyToolFilter`.
- **Upstream credentials:** Proxy handlers strip inbound request headers before forwarding tool, prompt, and resource requests. Preserve configured upstream bearer tokens/custom headers without forwarding downstream credentials. See `internal/service/mcp/proxy_header_test.go`.
- **Dashboard:** UI and dashboard APIs are development-only and return 404 in enterprise mode. Expose sanitized summaries; never return credential values, OAuth secrets, or raw sensitive upstream configuration.
- **Names and URIs:** Preserve `<server>__<tool>` / `<server>__<prompt>` canonical names, `mcpj://res/` resource URI encoding, and rewriting back to upstream identifiers during calls. Reuse existing conversion helpers.
- **Registry lifecycle:** Registration, enable/disable, deletion, and startup loading must keep persisted entities and in-memory global/group proxies consistent. Check tool-group callbacks when changing tool availability.
- **Sessions:** Use `getSession` and its cleanup/invalidation helpers. Stateless connections close after use; stateful connections are reused and invalidated when unhealthy. Preserve shutdown cleanup. Current stateful sessions are keyed by upstream server name, and upstream OAuth grants are gateway-scoped; do not assume per-client isolation.
- **Errors:** Wrap shared sentinels with `%w` and use `errors.Is`/`errors.As`. Reuse `handleServiceError` for the established HTTP mapping and retain machine-readable error codes consumed by clients.
- **Persistence:** Check SQLite and PostgreSQL compatibility for model/query changes, update `internal/migrations/migration.go` when needed, and consider existing data and GORM soft-delete behavior.
- **Protocol support:** Library capabilities do not automatically become gateway features. Group endpoints currently expose tools, not prompts or resources. Verify actual registration and routing before claiming support.

## Verification by change

- Add or extend meaningful regression tests for changed behavior; prefer existing helpers and local mock upstreams. Read assertions rather than inferring coverage from test names alone.
- For gateway/auth changes, cover allowed and denied access, relevant global/group endpoints, affected transports, and header handling.
- For registry/session changes, check persistence, exposed capabilities after mutations or startup, connection cleanup, and stateful reuse/invalidation as applicable.
- For dependency upgrades, review intervening releases and relevant source changes, then test affected behavior. Compilation alone does not establish compatibility; error identities, HTTP behavior, OAuth discovery, and defaults can change.
- For dashboard changes, follow its child guide, rebuild the embedded assets, run relevant Go API/E2E tests, and inspect the Go-served UI.
- For docs-only changes, check affected links, navigation, examples, and MDX as applicable; Go tests are unnecessary unless executable behavior also changed.
- Report commands run, their outcomes, and material verification gaps. Do not describe skipped tests as passing or disable checks to hide a failure.

## Documentation and contribution

- Update relevant local docs for changes to CLI flags, API behavior, configuration, authentication, or supported capabilities. Start with [development](docs/developers/development.mdx), [contributing](docs/developers/contributing.mdx), and the relevant reference/guide page.
- Prefer `docs/` for detailed guidance; the README includes a legacy reference section. Check version requirements against manifests and scripts rather than copying potentially stale prose.
- Follow [.github/pull_request_template.md](.github/pull_request_template.md) when preparing a PR. It requires linked issue/discussion context, full Go tests, the CLI integration script, relevant tests/docs, and human review of AI-generated code. Do not claim human review or maintainer priority confirmation on someone's behalf.
- For significant contributions, surface any missing maintainer alignment before submission. Continue user-authorized local work without opening unsolicited issues or treating every routine edit as a new approval gate.
- Release configuration lives in [.goreleaser.yaml](.goreleaser.yaml). It builds the dashboard, tests/lints, and distributes binaries, container images, and the separate `mcpjungle/homebrew-mcpjungle` tap. Only repository owners create releases; publishing or changing another repository must be part of the authorized task.
