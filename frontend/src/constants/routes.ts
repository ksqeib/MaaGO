export const ROUTES = {
  home: "/",
  tasks: "/tasks",
  logs: "/logs",
  settings: "/settings",
  about: "/about",
} as const;

export const isRouteActive = (pathname: string, route: string): boolean => {
  if (!route) return false;
  if (route === ROUTES.home) return pathname === ROUTES.home;
  return pathname === route || pathname.startsWith(`${route}/`);
};
