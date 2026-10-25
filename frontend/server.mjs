import { createServer } from "node:http";
import { randomBytes } from "node:crypto";
import { readFile, stat } from "node:fs/promises";
import { extname, join, normalize, resolve } from "node:path";
import { fileURLToPath } from "node:url";

// Minimal static server for the SPA build (build/client).
//
// It serves the files as they are, falls back to index.html for any unknown
// path so client-side routes survive a hard reload, and applies the security
// headers a browser app needs. The Content Security Policy is enforced with a
// per-response nonce, because React Router's SPA shell contains inline
// bootstrap scripts: a nonce lets those run without opening script-src to
// 'unsafe-inline'.

const root = resolve(fileURLToPath(new URL(".", import.meta.url)), "build/client");
const port = Number(process.env.PORT) || 3000;

// The API origin is needed in connect-src so the browser may call the Go
// backend and open the signaling WebSocket.
const apiBaseUrl = process.env.VITE_API_BASE_URL || process.env.API_BASE_URL || "http://localhost:8080";

const MIME = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript",
  ".mjs": "text/javascript",
  ".css": "text/css",
  ".json": "application/json",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".gif": "image/gif",
  ".webp": "image/webp",
  ".ico": "image/x-icon",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
  ".map": "application/json",
};

function connectSources() {
  const sources = new Set(["'self'"]);

  try {
    const api = new URL(apiBaseUrl);
    sources.add(api.origin);
    sources.add(`${api.protocol === "https:" ? "wss:" : "ws:"}//${api.host}`);
  } catch {
    // A malformed VITE_API_BASE_URL should not stop the site from loading; the
    // browser will report the blocked request instead.
  }

  return [...sources].join(" ");
}

function contentSecurityPolicy(nonce) {
  return [
    "default-src 'self'",
    `script-src 'self' 'nonce-${nonce}'`,
    // Inline styles are allowed because Radix and the browser itself set style
    // attributes for focus management and scroll locking. Style injection is a
    // far smaller risk than script injection, which stays nonce-only.
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: blob:",
    "media-src 'self' blob:",
    "font-src 'self'",
    `connect-src ${connectSources()}`,
    "frame-ancestors 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "object-src 'none'",
  ].join("; ");
}

function securityHeaders(nonce) {
  return {
    "Content-Security-Policy": contentSecurityPolicy(nonce),
    "X-Content-Type-Options": "nosniff",
    "X-Frame-Options": "DENY",
    "Referrer-Policy": "strict-origin-when-cross-origin",
    "Cross-Origin-Opener-Policy": "same-origin",
    "Permissions-Policy": "camera=(self), microphone=(self), geolocation=(), payment=()",
    "Strict-Transport-Security": "max-age=31536000; includeSubDomains",
  };
}

// index.html is read once and re-read only if the build changes underneath us.
let indexCache = null;
let indexCachedAt = 0;

async function loadIndex() {
  const info = await stat(join(root, "index.html")).catch(() => null);
  const stamp = info ? info.mtimeMs : 0;

  if (indexCache && indexCachedAt === stamp) {
    return indexCache;
  }

  indexCache = await readFile(join(root, "index.html"), "utf8");
  indexCachedAt = stamp;
  return indexCache;
}

function withNonce(html, nonce) {
  // Every script tag in the shell gets the nonce, including the inline
  // bootstrap scripts React Router emits.
  return html.replaceAll("<script", `<script nonce="${nonce}"`);
}

createServer(async (req, res) => {
  try {
    const url = new URL(req.url ?? "/", "http://localhost");
    const pathname = decodeURIComponent(url.pathname);

    // Resolve and guard against path traversal.
    const filePath = resolve(join(root, normalize(pathname)));
    if (!filePath.startsWith(root)) {
      res.writeHead(403, { "Content-Type": "text/plain" });
      return res.end("Forbidden");
    }

    try {
      const info = await stat(filePath);
      if (!info.isFile()) {
        throw new Error("not a file");
      }

      const extension = extname(filePath);
      const data = await readFile(filePath);

      // Hashed assets can be cached forever; the shell cannot, because it
      // carries a fresh nonce on every response.
      const cacheControl =
        extension === ".html" ? "no-store" : "public, max-age=31536000, immutable";

      res.writeHead(200, {
        "Content-Type": MIME[extension] ?? "application/octet-stream",
        "Cache-Control": cacheControl,
        ...securityHeaders(randomBytes(16).toString("base64")),
      });
      return res.end(data);
    } catch {
      // SPA fallback: any unmatched path gets the shell.
      const nonce = randomBytes(16).toString("base64");
      const html = withNonce(await loadIndex(), nonce);

      res.writeHead(200, {
        "Content-Type": MIME[".html"],
        "Cache-Control": "no-store",
        ...securityHeaders(nonce),
      });
      return res.end(html);
    }
  } catch (error) {
    res.writeHead(500, { "Content-Type": "text/plain" });
    return res.end(String(error));
  }
}).listen(port, () => {
  console.log(`Serving build/client on http://localhost:${port}`);
  console.log(`Content Security Policy allows the API at ${apiBaseUrl}`);
});

