# Strangerly backend: a single static Go binary with no runtime dependencies.

FROM golang:1.26-alpine AS build
WORKDIR /src

# Dependencies first, so a code change does not re-download the module graph.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/strgback .

FROM alpine:3.21
RUN adduser -D -u 10001 strangerly
COPY --from=build /out/strgback /usr/local/bin/strgback

USER strangerly
EXPOSE 8080

# /healthz answers as soon as the process is up; /readyz flips to 503 while the
# process is draining.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/strgback"]
