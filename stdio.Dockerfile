# This docker image is used when you want mcpjungle to run STDIO MCP servers that rely on `uvx` or `npx` to start.
#
# Build mcpjungle from source so local docker builds do not depend on a
# pre-generated binary in the build context.
FROM golang:1.24-bookworm AS builder

WORKDIR /src

RUN apt-get update \
    && apt-get install -y curl ca-certificates gnupg \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Cache backend and dashboard dependencies.
COPY go.mod go.sum ./
RUN go mod download

COPY web/dashboard/package*.json ./web/dashboard/
RUN cd web/dashboard && npm ci

COPY . .

# Build dashboard assets and embed them in the binary.
RUN bash ./scripts/build-dashboard.sh \
    && CGO_ENABLED=0 GOWORK=off go build -trimpath -ldflags="-s -w" -o /out/mcpjungle ./main.go

# Runtime image with uvx and npx support.
FROM ghcr.io/astral-sh/uv:debian

# OCI image labels
LABEL org.opencontainers.image.source="https://github.com/mcpjungle/mcpjungle"
LABEL org.opencontainers.image.description="MCPJungle - Self-hosted MCP Gateway for developers and enterprises"
LABEL org.opencontainers.image.title="MCPJungle"
LABEL org.opencontainers.image.vendor="mcpjungle"

# Install Node.js
RUN apt-get update \
    && apt-get install -y curl gnupg \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from the builder stage
COPY --from=builder /out/mcpjungle /mcpjungle

EXPOSE 8080
ENTRYPOINT ["/mcpjungle"]

# Run the Registry Server by default
CMD ["start"]