# Strangerly frontend: built with bun, served as static files by node.

FROM oven/bun:1-alpine AS build
WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .

# The API base URL is baked into the bundle at build time, so pass the real one:
#   docker build --build-arg VITE_API_BASE_URL=https://api.example.com ...
ARG VITE_API_BASE_URL=http://localhost:8080
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
RUN bun run build

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000

# server.mjs needs the built client plus the API base URL, which it uses for the
# Content Security Policy connect-src. That one is read at runtime.
COPY --from=build /app/build ./build
COPY --from=build /app/server.mjs ./server.mjs

USER node
EXPOSE 3000

CMD ["node", "server.mjs"]
