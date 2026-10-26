package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Environment is "development" or "production". It drives cookie security
	// and log verbosity, never behaviour the user can see.
	Environment string

	// AllowedOrigins is the exact set of browser origins allowed to call the
	// API and to open the signaling WebSocket. Requests are credentialed, so
	// wildcards are refused rather than silently downgraded.
	AllowedOrigins []string

	ListenAddr string

	CookieName   string
	CookieSecure bool
	CookieDomain string

	// SessionTTL is how long an idle anonymous session survives. It slides
	// forward on every authenticated request.
	SessionTTL time.Duration
	// QueueTTL drops a waiting session that never came back.
	QueueTTL time.Duration
	// MatchIdleTTL is how long a match survives after a participant stops
	// talking to the signaling socket, which is what a closed tab looks like.
	MatchIdleTTL time.Duration
	// MaxMatchDuration caps a single call so a forgotten tab cannot hold a
	// peer connection open forever.
	MaxMatchDuration time.Duration
	// SkipLock is the server enforced window during which Skip is refused.
	SkipLock time.Duration

	ICE ICEConfig

	RateLimit RateLimitConfig

	LogLevel string

	// TrustedProxies is empty by default: gin then reads the socket peer
	// address instead of a spoofable X-Forwarded-For header.
	TrustedProxies []string
}

// ICEConfig describes the STUN/TURN servers handed to browsers. A STUN server
// alone does not guarantee connectivity across every NAT, so TURN is supported
// and optional; when a shared secret is configured the backend mints
// short-lived credentials instead of shipping static ones to the client.
//
// TURNURLs is a list on purpose: a hosted relay publishes the same relay over
// several transports (UDP, TCP, and TLS on 443 for networks that allow nothing
// else), and a browser needs all of them to stand a chance on a hostile
// network.
type ICEConfig struct {
	STUNURLs          []string
	TURNURLs          []string
	TURNUsername      string
	TURNCredential    string
	TURNSharedSecret  string
	TURNCredentialTTL time.Duration
}

// RateLimitConfig holds the request budgets per client. They are deliberately
// generous for real people and tight enough to make scripted abuse boring.
type RateLimitConfig struct {
	SessionPerMinute    int
	JoinPerMinute       int
	SignalPerSecond     int
	ReportPerTenMinutes int
}

// App is the loaded configuration. LoadEnv must run before any other package
// reads it; main.go is the only caller.
var App Config

// LoadEnv reads .env (when present) and the process environment into App.
//
// .env.local is read first and wins: it is how a development machine overrides
// a .env that is written for the deployed service, without editing the file
// that gets deployed. A deployed host must not have one.
func LoadEnv() {
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load()

	App = Config{
		Environment: getEnvWithDefault("ENVIRONMENT", "development"),
		AllowedOrigins: splitList(getEnvWithDefault(
			"ALLOWED_ORIGINS",
			getEnvWithDefault("FRONTEND_ORIGIN", "http://localhost:5173"),
		)),
		ListenAddr:   getEnvWithDefault("LISTEN_ADDR", ":8080"),
		CookieName:   getEnvWithDefault("COOKIE_NAME", "strg_session"),
		CookieSecure: getEnvBool("COOKIE_SECURE", false),
		CookieDomain: os.Getenv("COOKIE_DOMAIN"),

		SessionTTL:       getEnvDuration("SESSION_TTL", 2*time.Hour),
		QueueTTL:         getEnvDuration("QUEUE_TTL", 2*time.Minute),
		MatchIdleTTL:     getEnvDuration("MATCH_IDLE_TTL", 20*time.Second),
		MaxMatchDuration: getEnvDuration("MAX_MATCH_DURATION", 60*time.Minute),
		SkipLock:         getEnvDuration("SKIP_LOCK_SECONDS", 5*time.Second),

		ICE: ICEConfig{
			STUNURLs: splitList(getEnvWithDefault(
				"STUN_URLS",
				"stun:stun.l.google.com:19302",
			)),
			// TURN_URLS takes a comma separated list; TURN_URL is the single
			// URL form this service used to read, kept so an existing .env
			// keeps working.
			TURNURLs:          splitList(getEnvWithDefault("TURN_URLS", os.Getenv("TURN_URL"))),
			TURNUsername:      os.Getenv("TURN_USERNAME"),
			TURNCredential:    os.Getenv("TURN_CREDENTIAL"),
			TURNSharedSecret:  os.Getenv("TURN_SHARED_SECRET"),
			TURNCredentialTTL: getEnvDuration("TURN_CREDENTIAL_TTL", time.Hour),
		},

		RateLimit: RateLimitConfig{
			SessionPerMinute:    getEnvInt("RATE_LIMIT_SESSION_PER_MINUTE", 20),
			JoinPerMinute:       getEnvInt("RATE_LIMIT_JOIN_PER_MINUTE", 60),
			SignalPerSecond:     getEnvInt("RATE_LIMIT_SIGNAL_PER_SECOND", 60),
			ReportPerTenMinutes: getEnvInt("RATE_LIMIT_REPORT_PER_10MIN", 5),
		},

		LogLevel:       getEnvWithDefault("LOG_LEVEL", "info"),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	}
}

// IsProduction reports whether the app believes it is running for real users.
func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}

// PrimaryOrigin is the origin the browser is expected to be served from. It is
// the first entry of AllowedOrigins, which is what CORS and the WebSocket
// upgrade check use.
func (c Config) PrimaryOrigin() string {
	if len(c.AllowedOrigins) == 0 {
		return ""
	}
	return c.AllowedOrigins[0]
}

// SlogLevel maps LOG_LEVEL onto a slog level, defaulting to info.
func (c Config) SlogLevel() slog.Level {
	switch strings.ToLower(strings.TrimSpace(c.LogLevel)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return defaultValue
	}
	return n
}

func getEnvBool(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	// A bare number is read as seconds so SKIP_LOCK_SECONDS=5 and
	// SESSION_TTL=2h both work.
	if n, err := strconv.Atoi(value); err == nil {
		if n <= 0 {
			return defaultValue
		}
		return time.Duration(n) * time.Second
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
