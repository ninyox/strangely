package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"strgback/config"
	"strgback/store"
	"strgback/utils"
)

// Context keys set by SessionMiddleware.
const (
	ContextSessionID = "strg.sessionID"
	ContextUsername  = "strg.username"
)

// SessionMiddleware authenticates a request from the anonymous session cookie.
// The cookie is opaque and HttpOnly; the store only ever sees its hash, and a
// client-supplied session id in a body or query string is ignored entirely.
func SessionMiddleware(st *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(config.App.CookieName)
		if err != nil || token == "" {
			utils.ErrorResponseCode(c, http.StatusUnauthorized, "no_session",
				"No session yet. Reload the page to start one.", nil)
			c.Abort()
			return
		}

		view, ok := st.Authenticate(token)
		if !ok {
			ClearSessionCookie(c)
			utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
				"Your session expired. Reload the page to start a new one.", nil)
			c.Abort()
			return
		}

		c.Set(ContextSessionID, view.ID)
		c.Set(ContextUsername, view.Username)
		c.Next()
	}
}

// SessionIDFrom returns the authenticated session id, or "" when the route was
// not behind SessionMiddleware.
func SessionIDFrom(c *gin.Context) string {
	if value, ok := c.Get(ContextSessionID); ok {
		if id, ok := value.(string); ok {
			return id
		}
	}
	return ""
}

// UsernameFrom returns the display name stored on the session.
func UsernameFrom(c *gin.Context) string {
	if value, ok := c.Get(ContextUsername); ok {
		if name, ok := value.(string); ok {
			return name
		}
	}
	return ""
}

// SetSessionCookie writes the anonymous session cookie. SameSite=Lax is enough
// for the API's own requests, and Secure follows the environment so local
// development over plain HTTP still works.
func SetSessionCookie(c *gin.Context, token string, maxAgeSeconds int) {
	cfg := config.App
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		cfg.CookieName,
		token,
		maxAgeSeconds,
		"/",
		cfg.CookieDomain,
		cfg.CookieSecure,
		true, // HttpOnly: no script may read the session token
	)
}

// ClearSessionCookie expires the cookie in the browser.
func ClearSessionCookie(c *gin.Context) {
	SetSessionCookie(c, "", -1)
}
