# Contribution Guidelines

## Development

### Code of Conduct

We are committed to providing a friendly, safe, and welcoming environment for all contributors. Please be respectful and inclusive in all interactions.

### Getting Started

1. **Fork the repository** on GitHub.
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/MCPJungle.git
   cd MCPJungle
   ```
3. **Add the upstream repository** as a remote:
   ```bash
   git remote add upstream https://github.com/mcpjungle/MCPJungle.git
   ```
4. **Create a new branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

### Development Environment Setup

See the [DEVELOPMENT.md](DEVELOPMENT.md) file for detailed setup instructions.

### Making Changes

#### Coding Standards

- **Go Code**:
  - Follow the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
  - Use `gofmt` to format your code
  - Run `go vet` before submitting your changes
  - Ensure all tests pass with `go test ./...`
  - Add tests for new functionality


### Before Making Significant Changes

1. **If you plan on opening a PR for a significant change, first open a Discussion to align with the community on the changes you plan on making.**
This helps ensure your effort is well-directed and aligns with project priorities.

2. **Sending Pull Requests with large amounts of AI-generated Code is discouraged.**
This makes it very hard for the maintainers to review your code.

For any code that you generate using AI tools, please make sure to:
1. Run it. Does it all work as expected?
2. Run tests. Do they all pass?
3. Review it yourself. Does it fit into the rest of the codebase? Can it be reduced?

### Pull Request Process

1. **Update your fork** with the latest changes from upstream:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push your changes** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create a Pull Request** from your fork to the main repository.

4. **Write a good commit description** with all required information.

5. **Address review comments** if requested by maintainers.

6. **Update your PR** if needed:
   ```bash
   git add .
   git commit -m "address review comments"
   git push origin feature/your-feature-name
   ```

7. Once approved, a maintainer will merge your PR.

### Documentation

- Update documentation for any changes to APIs, CLIs, or user-facing features
- Add examples for new features
- Update the README if necessary
- Add comments to your code explaining complex logic

### Dependency Management

- We rely on `go.mod` + `go.sum` as the single source of truth for dependencies
- The `vendor/` directory is not committed to reduce repo size
- For offline builds, use `go mod vendor` locally but don't commit the changes
- CI/Docker builds use module-aware mode with Go proxy

### Releasing

Only project maintainers can create releases. The current process is:

1. A new release is always made from the `main` branch. Keep it up-to-date and clean.
2. Create a tag for the release
3. Build and publish artifacts using goreleaser

### Community

- Join our [Discord server](https://discord.gg/CapV4Z3krk) for discussions
- Help answer questions in GitHub issues
- Review pull requests from other contributors
- Participate in GitHub Discussions for proposals and questions

## License

By contributing to this project, you agree that your contributions will be licensed under the project's license.

## Questions?

If you have any questions about contributing, please open an issue or reach out to the maintainers on Discord.

---

We're excited to see your contributions! If you have any questions, don't hesitate to reach out. 🎉
