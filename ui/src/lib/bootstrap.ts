import { api, ApiError } from "./api";
import type { BootstrapState, HealthResponse } from "./types";

async function fetchHealth(): Promise<HealthResponse> {
  const response = await fetch("/health");
  if (!response.ok) {
    throw new Error(`Health check failed with status ${response.status}`);
  }
  return (await response.json()) as HealthResponse;
}

export async function bootstrapApp(token: string): Promise<BootstrapState> {
  try {
    const [health, metadata] = await Promise.all([fetchHealth(), api.metadata()]);

    try {
      const settings = await api.settings(token || undefined);
      if (settings.mode === "development") {
        return { stage: "ready", health, metadata, settings, user: null };
      }

      if (!token) {
        return {
          stage: "token",
          health,
          metadata,
          message: "Enterprise mode requires user access token.",
        };
      }

      const user = await api.whoAmI(token);
      return { stage: "ready", health, metadata, settings, user };
    } catch (error) {
      if (error instanceof ApiError && error.status === 403 && error.message.includes("server is not initialized")) {
        return { stage: "init", health, metadata };
      }

      if (error instanceof ApiError && error.status === 401) {
        return {
          stage: "token",
          health,
          metadata,
          message: "Provide valid enterprise token to unlock management UI.",
        };
      }

      throw error;
    }
  } catch (error) {
    return {
      stage: "error",
      message: error instanceof Error ? error.message : "Bootstrap failed",
    };
  }
}
