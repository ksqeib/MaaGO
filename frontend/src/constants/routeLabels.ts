import { ROUTES } from "./routes";

export const ROUTE_LABEL_KEYS: Record<string, string> = {
  [ROUTES.home]: "nav.home",
  [ROUTES.tasks]: "nav.tasks",
  [ROUTES.logs]: "nav.logs",
  [ROUTES.settings]: "nav.settings",
  [ROUTES.about]: "nav.about",
};

export const getRouteLabelKey = (pathname: string): string =>
  ROUTE_LABEL_KEYS[pathname] ?? "nav.breadcrumb";
