import type { Config } from "@react-router/dev/config";

export default {
  // This is a client-side rendered (SPA) app, copied from Lovable.
  // Disable server-side rendering so hard reloads serve the SPA shell and
  // render entirely on the client (no blank page on reload).
  ssr: false,
} satisfies Config;
