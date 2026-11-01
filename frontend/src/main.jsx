import "./style.css";

import React from "react";
import { createRoot } from "react-dom/client";
import { I18nextProvider } from "react-i18next";
import { HashRouter } from "react-router-dom";
import { ThemeProvider } from "next-themes";

import App from "./App";
import i18n, { i18nReady } from "./i18n";

const container = document.getElementById("root");

if (!container) {
  throw new Error("Root element was not found");
}

const root = createRoot(container);

const bootstrap = async () => {
  try {
    await i18nReady;
    root.render(
      <React.StrictMode>
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          storageKey="maago.theme"
        >
          <I18nextProvider i18n={i18n}>
            <HashRouter>
              <App />
            </HashRouter>
          </I18nextProvider>
        </ThemeProvider>
      </React.StrictMode>,
    );
  } catch (error) {
    console.error("[startup] failed to render", error);
    root.render(
      <div
        style={{
          display: "flex",
          height: "100vh",
          alignItems: "center",
          justifyContent: "center",
          fontFamily: "sans-serif",
        }}
      >
        Failed to start MaaGO. Please check the startup log.
      </div>,
    );
  }
};

void bootstrap();
