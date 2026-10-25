package routes

import (
	"log/slog"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"strgback/config"
	"strgback/handlers"
	"strgback/middleware"
)

// NewRouter builds the router the server runs. main.go and the route tests
// both build their engine here, so the tests exercise the real middleware
// chain instead of a bare engine.
//
// It reads configuration, so it must be called after config.LoadEnv().
func NewRouter(api *handlers.API) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.SecurityHeaders())

	origins := config.App.AllowedOrigins
	if len(origins) == 0 {
		// cors.New panics on an empty origin list, and an empty list would
		// mean "any origin" for a credentialed API, which is never intended.
		origins = []string{"http://localhost:5173"}
		slog.Warn("ALLOWED_ORIGINS was empty, falling back to the local development origin")
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Empty TRUSTED_PROXIES keeps gin on the socket peer address, so a
	// spoofed X-Forwarded-For cannot move a rate limit key.
	if err := r.SetTrustedProxies(config.App.TrustedProxies); err != nil {
		slog.Error("could not configure trusted proxies", "error", err)
	}

	SetupRoutes(r, api)

	return r
}

// SetupRoutes registers every route on the engine.
func SetupRoutes(r *gin.Engine, api *handlers.API) {
	r.GET("/", func(c *gin.Context) {
		c.String(200, "Strangerly API. See /healthz, /readyz and /api/config.")
	})

	r.GET("/healthz", api.Health)
	r.GET("/readyz", api.Ready)

	// The signaling socket authenticates inside its handler: a WebSocket
	// handshake failure has to be a plain HTTP status, and the Origin must be
	// validated before the upgrade.
	r.GET("/api/signaling", api.Hub.ServeWS)

	apiGroup := r.Group("/api")
	{
		// Public: ICE servers, the Skip window, and the report reasons. No
		// session is needed to read the rules of the room.
		apiGroup.GET("/config", api.PublicConfig)

		// The only endpoint that mints an identity, so it is limited by peer
		// address rather than by session.
		apiGroup.POST("/session",
			middleware.RateLimit(api.SessionLimiter, middleware.ByIP),
			api.CreateSession)

		// Everything below needs the anonymous session cookie.
		authed := apiGroup.Group("", middleware.SessionMiddleware(api.Store))
		{
			authed.GET("/session", api.SessionInfo)

			authed.POST("/matchmaking/join",
				middleware.RateLimit(api.JoinLimiter, middleware.BySession),
				api.JoinMatchmaking)
			authed.POST("/matchmaking/cancel", api.CancelMatchmaking)
			authed.GET("/matchmaking/status", api.MatchmakingStatus)

			// Skip is gated by the store's five second window; stop and
			// report deliberately are not.
			authed.POST("/matches/:id/skip", api.SkipMatch)
			authed.POST("/matches/:id/stop", api.StopMatch)
			authed.POST("/matches/:id/report", api.ReportMatch)
		}
	}

	r.NoRoute(api.NotFound)
}
