import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./styles.css";
import { applyTheme, getInitialTheme } from "@/lib/theme";

const initialTheme = getInitialTheme();
applyTheme(initialTheme);

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App initialTheme={initialTheme} />
  </React.StrictMode>,
);
