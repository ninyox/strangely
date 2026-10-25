package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"strgback/middleware"
	"strgback/models"
	"strgback/store"
	"strgback/utils"
)

// joinRequest is the optional body of POST /api/matchmaking/join. Sending the
// username here is what makes "choose a name, then press Start matching" a
// single round trip.
type joinRequest struct {
	Username string `json:"username"`
}

// JoinMatchmaking implements POST /api/matchmaking/join.
func (a *API) JoinMatchmaking(c *gin.Context) {
	sessionID := middleware.SessionIDFrom(c)

	var req joinRequest
	if err := decodeJSON(c, &req); err != nil {
		utils.ErrorResponseCode(c, http.StatusBadRequest, "invalid_body",
			"Malformed request body.", nil)
		return
	}

	requested, hasUsername := normalizeRequestedUsername(c, req.Username)
	if c.IsAborted() {
		return
	}
	if hasUsername {
		if err := a.Store.SetUsername(sessionID, requested); err != nil {
			utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
				"Your session expired. Reload the page.", nil)
			return
		}
	}

	// A display name is required: the partner sees it, and the five second
	// rule reads better when there is a name attached to the other side.
	if strings.TrimSpace(a.Store.Username(sessionID)) == "" {
		utils.ErrorResponseCode(c, http.StatusBadRequest, "username_required",
			"Choose a name before you start matching.", nil)
		return
	}

	res, err := a.Store.Join(sessionID)
	if err != nil {
		if errors.Is(err, store.ErrUnknownSession) {
			utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
				"Your session expired. Reload the page.", nil)
			return
		}
		a.Logger.Error("join failed", "error", err, "session", sessionID)
		utils.ErrorResponseCode(c, http.StatusInternalServerError, "join_failed",
			"Could not join matchmaking. Please try again.", nil)
		return
	}

	payload := stateFromJoin(res, a.SkipLock, time.Now().UTC())

	switch {
	case res.InMatch:
		// Already in a call: the client is told the match it is in so it can
		// recover instead of opening a second one.
		utils.ErrorResponseCode(c, http.StatusConflict, "already_in_match",
			"You are already in a call.", payload)
	case res.AlreadyQueued:
		utils.ErrorResponseCode(c, http.StatusConflict, "already_queued",
			"You are already waiting for a partner.", payload)
	case res.State == models.StateQueued:
		utils.SuccessResponse(c, "Looking for someone…", payload)
	default:
		utils.SuccessResponse(c, "You are matched.", payload)
	}
}

// CancelMatchmaking implements POST /api/matchmaking/cancel.
func (a *API) CancelMatchmaking(c *gin.Context) {
	sessionID := middleware.SessionIDFrom(c)

	res, err := a.Store.Cancel(sessionID)
	if err != nil {
		if errors.Is(err, store.ErrUnknownSession) {
			utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
				"Your session expired. Reload the page.", nil)
			return
		}
		a.Logger.Error("cancel failed", "error", err, "session", sessionID)
		utils.ErrorResponseCode(c, http.StatusInternalServerError, "cancel_failed",
			"Could not leave the queue. Please try again.", nil)
		return
	}

	if res.InMatch {
		// Cancelling is a queue action. In a call the client should use
		// stop or skip, which is what the payload tells it.
		utils.ErrorResponseCode(c, http.StatusConflict, "already_in_match",
			"You are in a call. Use Stop to end it.", stateFromJoin(res, a.SkipLock, time.Now().UTC()))
		return
	}

	utils.SuccessResponse(c, "You left the queue.", stateFromJoin(res, a.SkipLock, time.Now().UTC()))
}

// MatchmakingStatus implements GET /api/matchmaking/status. It is the polling
// fallback for a client whose signaling socket is reconnecting.
func (a *API) MatchmakingStatus(c *gin.Context) {
	sessionID := middleware.SessionIDFrom(c)

	view, err := a.Store.Status(sessionID)
	if err != nil {
		if errors.Is(err, store.ErrUnknownSession) {
			utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
				"Your session expired. Reload the page.", nil)
			return
		}
		utils.ErrorResponseCode(c, http.StatusInternalServerError, "status_failed",
			"Could not read the current state.", nil)
		return
	}

	utils.SuccessResponse(c, "", stateFromStatus(view, time.Now().UTC()))
}
