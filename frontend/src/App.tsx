import { Spinner } from "@heroui/react";
import React, { Suspense, lazy, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";

import { NavigationFrame } from "@/components/NavigationFrame";
import { Sidebar } from "@/components/Sidebar";
import { TopBar } from "@/components/TopBar";
import { ROUTES } from "@/constants/routes";
import { AppStoreProvider, useAppStore } from "@/store/AppStore";

const DashboardPage = lazy(() =>
  import("@/pages/DashboardPage").then((module) => ({ default: module.DashboardPage })),
);
const TasksPage = lazy(() =>
  import("@/pages/TasksPage").then((module) => ({ default: module.TasksPage })),
);
const LogsPage = lazy(() =>
  import("@/pages/LogsPage").then((module) => ({ default: module.LogsPage })),
);
const SettingsPage = lazy(() =>
  import("@/pages/SettingsPage").then((module) => ({ default: module.SettingsPage })),
);
const AboutPage = lazy(() =>
  import("@/pages/AboutPage").then((module) => ({ default: module.AboutPage })),
);

const GlobalError = () => {
  const { error, clearError } = useAppStore();

  useEffect(() => {
    if (!error) return;
    const timer = window.setTimeout(clearError, 6000);
    return () => window.clearTimeout(timer);
  }, [clearError, error]);

  if (!error) return null;

  return (
    <div className="fixed bottom-4 left-1/2 z-[120] flex -translate-x-1/2 items-center gap-3 rounded-full border border-red-500/30 bg-red-500/95 px-4 py-2 text-xs font-medium text-white shadow-lg">
      <span className="max-w-[70vw] truncate">{error}</span>
      <button
        className="rounded-full bg-white/20 px-2 py-0.5 text-[10px]"
        onClick={clearError}
        type="button"
      >
        ✕
      </button>
    </div>
  );
};

const Shell = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const tryNavigate = (path: string | number) => {
    if (typeof path === "number") {
      navigate(path);
      return;
    }
    navigate(path);
  };

  return (
    <div
      className="app-backdrop relative isolate flex h-dvh w-full flex-row overflow-hidden bg-background text-foreground"
      style={{ "--content-pt": "4.5rem" } as React.CSSProperties}
    >
      <NavigationFrame />
      <Sidebar tryNavigate={tryNavigate} />
      <TopBar tryNavigate={tryNavigate} />

      <main
        className="min-h-0 w-full flex-1 overflow-hidden pl-14"
        id="main-content"
        tabIndex={-1}
      >
        <Suspense
          fallback={
            <div className="flex h-full w-full flex-col items-center justify-center gap-3">
              <Spinner size="lg" />
              <span className="text-xs text-muted">{t("common.loading")}</span>
            </div>
          }
        >
          <Routes>
            <Route element={<DashboardPage />} path={ROUTES.home} />
            <Route element={<TasksPage />} path={ROUTES.tasks} />
            <Route element={<LogsPage />} path={ROUTES.logs} />
            <Route element={<SettingsPage />} path={ROUTES.settings} />
            <Route element={<AboutPage />} path={ROUTES.about} />
            <Route element={<Navigate replace to={ROUTES.home} />} path="*" />
          </Routes>
        </Suspense>
      </main>

      <GlobalError />
    </div>
  );
};

const App = () => (
  <AppStoreProvider>
    <Shell />
  </AppStoreProvider>
);

export default App;
