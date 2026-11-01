import { Spinner } from "@heroui/react";
import React, { Suspense, lazy } from "react";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";

import { NavigationFrame } from "@/components/NavigationFrame";
import { Sidebar } from "@/components/Sidebar";
import { TopBar } from "@/components/TopBar";
import { ROUTES } from "@/constants/routes";

const HomePage = lazy(() =>
  import("@/pages/HomePage").then((module) => ({ default: module.HomePage })),
);
const SettingsPage = lazy(() =>
  import("@/pages/SettingsPage").then((module) => ({
    default: module.SettingsPage,
  })),
);
const AboutPage = lazy(() =>
  import("@/pages/AboutPage").then((module) => ({ default: module.AboutPage })),
);
const PlaceholderPage = lazy(() =>
  import("@/pages/PlaceholderPage").then((module) => ({
    default: module.PlaceholderPage,
  })),
);

const App = () => {
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
            <div className="flex h-full w-full items-center justify-center">
              <Spinner size="lg" />
            </div>
          }
        >
          <Routes>
            <Route element={<HomePage />} path={ROUTES.home} />
            <Route
              element={
                <PlaceholderPage subtitleKey="tasks.subtitle" titleKey="tasks.title" />
              }
              path={ROUTES.tasks}
            />
            <Route
              element={
                <PlaceholderPage subtitleKey="logs.subtitle" titleKey="logs.title" />
              }
              path={ROUTES.logs}
            />
            <Route element={<SettingsPage />} path={ROUTES.settings} />
            <Route element={<AboutPage />} path={ROUTES.about} />
            <Route element={<Navigate replace to={ROUTES.home} />} path="*" />
          </Routes>
        </Suspense>
      </main>
    </div>
  );
};

export default App;
