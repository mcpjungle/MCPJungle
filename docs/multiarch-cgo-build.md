# Optional CGO/SQLite multi-architecture build

The existing Dockerfile and stdio.Dockerfile remain the default image paths.
Dockerfile.multiarch is an optional source-owned example for building
linux/amd64 and linux/arm64 from the same pinned repository revision.

It uses Node 22 locked dashboard build, including the repository-level
assets/ Vite alias; Go 1.25 CGO and the architecture-specific GNU C compiler
and libc headers for SQLite; and a target-platform runtime based on the pinned
MCPJungle 0.4.6 STDIO image. It does not install host-level QEMU or binfmt.

Build each architecture without external publishing using buildx:

  docker buildx build --platform linux/amd64 --load -f Dockerfile.multiarch -t mcpjungle:multiarch-amd64 .
  docker buildx build --platform linux/arm64 --load -f Dockerfile.multiarch -t mcpjungle:multiarch-arm64 .

Inspect the architecture with docker image inspect, then run a disposable
health/MCP smoke test under a network and privilege policy appropriate to the
test environment. Use a pinned source commit and compare the staged build
source across architectures. Binary/image architecture labels are not proof
of Git source identity; retain an immutable source revision and artifact digest.

Runtime image default is non-root UID/GID 65532:65532. Environments with
privileged volume ownership must explicitly select the required runtime UID.
