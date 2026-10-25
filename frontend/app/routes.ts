import { type RouteConfig, index, route } from "@react-router/dev/routes";

/**
 * Strangerly has three real screens: the landing page, the call screen, and
 * the honest copy. Everything else is a 404 that offers a way back.
 */
export default [
  index("routes/home.tsx"),
  route("chat", "routes/chat.tsx"),
  route("privacy", "routes/privacy.tsx"),
  route("guidelines", "routes/guidelines.tsx"),
  route("about", "routes/about.tsx"),
  route("*", "routes/not-found.tsx"),
] satisfies RouteConfig;
