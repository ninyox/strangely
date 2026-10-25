package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"strgback/config"
	"strgback/middleware"
	"strgback/utils"
)

// createSessionRequest is the optional body of POST /api/session.
type createSessionRequest struct {
	Username string `json:"username"`
}

// sessionPayload describes a temporary visit. SessionID is a short,
// non-secret handle for correlating logs and client state; it is not a
// credential and cannot be used to reach another session.
type sessionPayload struct {
	SessionID        string    `json:"sessionId"`
	Username         string    `json:"username,omitempty"`
	State            string    `json:"state"`
	SkipLockMs       int64     `json:"skipLockMs"`
	ExpiresInSeconds int64     `json:"expiresInSeconds"`
	ServerTime       time.Time `json:"serverTime"`
}

// CreateSession implements POST /api/session. It creates a temporary anonymous
// session and sets the opaque HttpOnly cookie, or resumes the session the
// browser already has. No account, email or password is involved anywhere.
func (a *API) CreateSession(c *gin.Context) {
	var req createSessionRequest
	if err := decodeJSON(c, &req); err != nil {
		utils.ErrorResponseCode(c, http.StatusBadRequest, "invalid_body",
			"Malformed request body.", nil)
		return
	}

	requested, hasUsername := normalizeRequestedUsername(c, req.Username)
	if c.IsAborted() {
		return
	}

	// Resume: a valid cookie means the visit is already identified.
	if token, err := c.Cookie(config.App.CookieName); err == nil && token != "" {
		if view, ok := a.Store.Authenticate(token); ok {
			username := view.Username
			if hasUsername {
				if err := a.Store.SetUsername(view.ID, requested); err != nil {
					utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
						"Your session expired. Reload the page.", nil)
					return
				}
				username = requested
			}
			utils.SuccessResponse(c, "Session resumed.", a.sessionPayload(view.ID, username))
			return
		}
	}

	username := ""
	if hasUsername {
		username = requested
	}

	view, token, err := a.Store.CreateSession(username)
	if err != nil {
		a.Logger.Error("could not create session", "error", err)
		utils.ErrorResponseCode(c, http.StatusInternalServerError, "session_failed",
			"Could not start a session. Please try again.", nil)
		return
	}

	middleware.SetSessionCookie(c, token, int(config.App.SessionTTL.Seconds()))
	utils.SuccessResponse(c, "Session created.", a.sessionPayload(view.ID, username))
}

// SessionInfo implements GET /api/session: it lets a reloading tab find out
// whether its cookie is still good without creating anything.
func (a *API) SessionInfo(c *gin.Context) {
	sessionID := middleware.SessionIDFrom(c)
	utils.SuccessResponse(c, "", a.sessionPayload(sessionID, middleware.UsernameFrom(c)))
}

func (a *API) sessionPayload(sessionID, username string) sessionPayload {
	return sessionPayload{
		SessionID:        sessionID,
		Username:         username,
		State:            "idle",
		SkipLockMs:       a.SkipLock.Milliseconds(),
		ExpiresInSeconds: int64(config.App.SessionTTL.Seconds()),
		ServerTime:       time.Now().UTC(),
	}
}

// normalizeRequestedUsername validates an optional username from a request
// body. When the body carries nothing usable it returns hasUsername=false and
// leaves the decision to the caller.
func normalizeRequestedUsername(c *gin.Context, raw string) (string, bool) {
	if strings.TrimSpace(raw) == "" {
		return "", false
	}

	name, err := utils.NormalizeUsername(raw)
	if err != nil {
		utils.ErrorResponseCode(c, http.StatusBadRequest, "invalid_username", err.Error(), nil)
		c.Abort()
		return "", false
	}
	return name, true
}
