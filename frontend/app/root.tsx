import {
  isRouteErrorResponse,
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from "react-router";

import type { Route } from "./+types/root";
import "./app.css";
import { CallProvider } from "../lib/call";
import { MediaProvider } from "../lib/media";
import { SessionProvider } from "../lib/session";

export const links: Route.LinksFunction = () => [
  { rel: "icon", href: "/favicon.svg", type: "image/svg+xml" },
  { rel: "apple-touch-icon", href: "/favicon.svg" },
  // The two faces above the fold. Preloading them removes the flash of
  // fallback type on first paint; everything else is discovered normally.
  {
    rel: "preload",
    href: "/fonts/Fraunces-latin.woff2",
    as: "font",
    type: "font/woff2",
    crossOrigin: "anonymous",
  },
  {
    rel: "preload",
    href: "/fonts/InterVar-latin.woff2",
    as: "font",
    type: "font/woff2",
    crossOrigin: "anonymous",
  },
];

export const meta: Route.MetaFunction = () => [
  { title: "Strangerly — One click away from someone new." },
  {
    name: "description",
    content:
      "Random one-to-one video chat. Choose a name, allow your camera, meet one stranger for a real conversation. No account, no profile, nothing recorded.",
  },
  { name: "theme-color", content: "#fbfaf7" },
  { property: "og:type", content: "website" },
  { property: "og:title", content: "Strangerly — One click away from someone new." },
  {
    property: "og:description",
    content:
      "A single conversation with one stranger. No account, no profile, no recording — and Skip is locked for five seconds so nobody is dropped before they are seen.",
  },
  { property: "og:site_name", content: "Strangerly" },
  { name: "twitter:card", content: "summary" },
];

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <Meta />
        <Links />
      </head>
      <body className="flex min-h-screen flex-col bg-paper text-ink antialiased">
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

/**
 * Providers sit above the router outlet on purpose: the media stream and the
 * call state belong to the visit, not to a route, so moving between screens
 * never switches the camera off or drops a match.
 */
export default function App() {
  return (
    <SessionProvider>
      <MediaProvider>
        <CallProvider>
          <Outlet />
        </CallProvider>
      </MediaProvider>
    </SessionProvider>
  );
}

/** Shown while the SPA shell hydrates. */
export function HydrateFallback() {
  return (
    <div className="fixed inset-0 flex items-center justify-center bg-paper">
      <div className="size-9 animate-spin rounded-full border-[1.5px] border-signal border-t-transparent" />
    </div>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  let title = "Something went wrong";
  let details = "An unexpected error occurred. Reloading the page usually fixes it.";

  if (isRouteErrorResponse(error)) {
    title = error.status === 404 ? "Page not found" : `Error ${error.status}`;
    details =
      error.status === 404
        ? "That page does not exist. The way back is the home page."
        : error.statusText || details;
  } else if (import.meta.env.DEV && error instanceof Error) {
    details = error.message;
  }

  return (
    <main className="mx-auto flex w-full max-w-2xl flex-1 flex-col items-start justify-center gap-5 px-6 py-24">
      <p className="mono-label text-ink-muted">Strangerly</p>
      <h1 className="font-display text-display-md text-ink">{title}</h1>
      <p className="text-body text-ink-soft">{details}</p>
      <a
        href="/"
        className="mt-2 inline-flex h-12 items-center rounded-control bg-signal px-6 text-body font-medium text-white"
      >
        Go to the home page
      </a>
    </main>
  );
}
