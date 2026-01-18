import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";

// Initialize theme before React renders to prevent flash of wrong theme
function initializeTheme() {
  const stored = localStorage.getItem("kerp_sidebar_collapsed");
  if (stored) {
    try {
      const { state } = JSON.parse(stored);
      const theme = state?.theme;
      const root = document.documentElement;
      root.classList.remove("light", "dark");
      if (theme === "system" || !theme) {
        const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches
          ? "dark"
          : "light";
        root.classList.add(systemTheme);
      } else {
        root.classList.add(theme);
      }
    } catch {
      // If parsing fails, use system preference
      const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light";
      document.documentElement.classList.add(systemTheme);
    }
  } else {
    // No stored preference, use system
    const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
    document.documentElement.classList.add(systemTheme);
  }
}

initializeTheme();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
