# Documentation guide

Read the [root guide](../AGENTS.md) as well. These instructions apply to the Mintlify documentation in this directory.

## Sources and scope

- Documentation is maintained here and published at `https://docs.mcpjungle.com`. Prefer these local files when working on this checkout.
- [docs.json](docs.json) defines site configuration and navigation. Pages are `.mdx`; most use YAML frontmatter with `title` and `description`. Preserve the conventions of the page being edited.
- [developers/development.mdx](developers/development.mdx) covers backend development and releases; [developers/dashboard-development.mdx](developers/dashboard-development.mdx) covers the frontend and embedding workflow.
- Cross-check behavior against code and test assertions. Some prose can lag implementation; do not copy a stale prerequisite, endpoint, or capability claim into another page. Flag material disagreements and fix relevant documentation within the task's scope.
- Describe shipped behavior separately from planned features. Open issues, PRs, and upstream SDK capabilities do not establish support in MCPJungle.

## Writing conventions

- Use short, direct sentences, active voice, and second person for instructions.
- Use sentence case headings, **bold** for UI labels, and backticks for paths, flags, commands, configuration keys, and identifiers.
- Use MCPJungle for the product and `mcpjungle` for the executable. Distinguish upstream MCP servers, downstream MCP clients, and human management users.
- Distinguish development mode from enterprise mode, and upstream OAuth from downstream client-token authentication. Avoid implying that upstream credentials are isolated per downstream user.
- Keep setup steps reproducible. State prerequisites and working directories where needed; use placeholders for credentials and avoid commands that overwrite existing user configuration without explanation.
- Reuse nearby Mintlify components and code-block styles. Give examples that match actual CLI help, JSON field names, routes, and defaults.
- Link to the canonical explanation rather than duplicating long procedures or version numbers.

## Where to update

| Change | Relevant pages |
| --- | --- |
| CLI command or flag | `reference/cli-*.mdx` and the related guide |
| Registration/configuration schema | `reference/config-file.mdx` and registration guides |
| Environment variable or startup default | `reference/environment-variables.mdx` and relevant deployment pages |
| REST API contract | `reference/api-overview.mdx` and relevant usage examples |
| Authentication or permissions | `governance/` and relevant integration/deployment pages |
| Supported capability or limitation | `support-matrix.mdx`, `limitations.mdx`, and the affected guide |
| Dashboard behavior or build workflow | `guides/dashboard.mdx`, `developers/dashboard-development.mdx`, and developer setup as needed |

Check `core-concepts.mdx` and README examples if the change affects their explanations. Add new pages to `docs.json` when they should appear in site navigation. Use site-relative internal links without the `.mdx` suffix in published pages, matching existing examples.

## Validation

Run available Mintlify checks from `docs/`:

```bash
mint broken-links
mint dev
```

Use preview for changed MDX/components/layout and inspect code samples and navigation. If the Mintlify CLI is unavailable, report that limitation and check affected file targets, anchors, navigation entries, and syntax manually. Installing a skill or plugin is not a prerequisite for editing docs.

Do not run Go or frontend builds solely for prose changes. Verify commands against their implementation when changing examples; avoid running publishing commands or commands against a real deployment merely to validate documentation.
