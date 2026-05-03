import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { RouterProvider } from "react-router-dom";

import { bootstrapApp } from "./lib/bootstrap";
import { clearStoredToken, getStoredToken, setStoredToken } from "./lib/auth";
import { createAppRouter } from "./lib/router";
import type { BootstrapReadyState, BootstrapState } from "./lib/types";
import { AuthGate } from "./components/auth/AuthGate";
import { InitGate } from "./components/auth/InitGate";
import { ErrorState } from "./components/ui/ErrorState";

type AppContextValue = BootstrapReadyState & {
  token: string;
  setToken: (token: string) => void;
  clearToken: () => void;
  refresh: () => Promise<void>;
  isAdminEquivalent: boolean;
};

const AppContext = createContext<AppContextValue | null>(null);

export function useAppContext(): AppContextValue {
  const value = useContext(AppContext);
  if (!value) {
    throw new Error("App context unavailable");
  }
  return value;
}

const router = createAppRouter();

export default function App() {
  const [token, updateToken] = useState(() => getStoredToken());
  const [state, setState] = useState<BootstrapState>({ stage: "loading" });

  async function refresh() {
    setState({ stage: "loading" });
    setState(await bootstrapApp(token));
  }

  useEffect(() => {
    void refresh();
  }, [token]);

  function persistToken(nextToken: string) {
    setStoredToken(nextToken);
    updateToken(nextToken);
  }

  function forgetToken() {
    clearStoredToken();
    updateToken("");
  }

  const contextValue = useMemo(() => {
    if (state.stage !== "ready") {
      return null;
    }

    return {
      ...state,
      token,
      setToken: persistToken,
      clearToken: forgetToken,
      refresh,
      isAdminEquivalent: state.settings.mode === "development" || state.user?.role === "admin",
    } satisfies AppContextValue;
  }, [state, token]);

  if (state.stage === "loading") {
    return <div className="grid min-h-screen place-items-center bg-shell text-body">Booting control surface...</div>;
  }

  if (state.stage === "init") {
    return <InitGate metadata={state.metadata} onReady={persistToken} onRefresh={refresh} />;
  }

  if (state.stage === "token") {
    return (
      <AuthGate
        metadata={state.metadata}
        message={state.message}
        currentToken={token}
        onSubmit={persistToken}
        onClear={forgetToken}
      />
    );
  }

  if (state.stage === "error" || !contextValue) {
    const message = state.stage === "error" ? state.message : "UI context was not created.";
    return <ErrorState title="UI bootstrap failed" message={message} onRetry={refresh} />;
  }

  return (
    <AppContext.Provider value={contextValue}>
      <RouterProvider router={router} />
    </AppContext.Provider>
  );
}
