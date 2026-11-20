# This docker image is used when you want mcpjungle to run STDIO MCP servers that rely on `uvx` or `npx` to start.

# Use the official uv image as base
FROM ghcr.io/astral-sh/uv:debian

# OCI image labels
LABEL org.opencontainers.image.source="https://github.com/mcpjungle/mcpjungle"
LABEL org.opencontainers.image.description="MCPJungle - Self-hosted MCP Gateway for developers and enterprises"
LABEL org.opencontainers.image.title="MCPJungle"
LABEL org.opencontainers.image.vendor="mcpjungle"

# Install Node.js
RUN apt-get update \
    && apt-get install -y curl gnupg \
    && curl -fsSL https://deb.nodesource.com/setup_24.x | bash - \
    && apt-get install -y nodejs \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Copy the binary built by goreleaser
COPY mcpjungle /mcpjungle

EXPOSE 8080
ENTRYPOINT ["/mcpjungle"]

# Run the Registry Server by default
CMD ["start"]