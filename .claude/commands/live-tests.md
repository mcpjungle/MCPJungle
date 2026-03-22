Run the MCPJungle live integration tests.

There are two layers of live testing — run both:

## 1. Shell integration tests

`scripts/test-mcpjungle.sh` — builds the binary, starts a Docker Compose stack,
registers real MCP servers (context7, filesystem), exercises CLI commands, and
verifies stateful session reuse. Run it first:

```bash
cd /Users/admin/work/opensource/MCPJungeFork && ./scripts/test-mcpjungle.sh
```

## 2. Go live tests (TestLive_*)

`internal/e2e/live/` — connects to a running MCPJungle instance and tests
all transport types (stdio, SSE, streamable_http), tool groups, and enterprise ACL.

Run via the dedicated setup script:

```bash
# Dev tests only (enterprise tests skipped):
cd /Users/admin/work/opensource/MCPJungeFork && ./scripts/run-live-tests.sh

# Dev + enterprise tests:
cd /Users/admin/work/opensource/MCPJungeFork && ./scripts/run-live-tests.sh --enterprise
```

`scripts/run-live-tests.sh` handles everything automatically:
- Builds the binary
- Starts server-everything upstreams (SSE on :3001, streamableHttp on :3002)
- Starts MCPJungle dev on :8080 and registers all three upstreams
- (With --enterprise) Starts MCPJungle enterprise on :8081, initialises it, creates test clients
- Runs `go test ./internal/e2e/live/ -run TestLive -v -timeout 120s`
- Cleans up all background processes on exit

When acting as Claude: run both scripts using the Bash tool, report a summary of
passed/failed tests. If any test fails, show the full failure output and suggest a fix.
