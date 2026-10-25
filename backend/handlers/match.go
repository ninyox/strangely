package handlers

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"strgback/middleware"
	"strgback/models"
	"strgback/store"
	"strgback/utils"
)

// endPayload describes the result of ending a call.
type endPayload struct {
	State        string    `json:"state"`
	MatchID      string    `json:"matchId"`
	Reason       string    `json:"reason"`
	AlreadyEnded bool      `json:"alreadyEnded"`
	ServerTime   time.Time `json:"serverTime"`
}

// skipLockedPayload is the conflict body for an early Skip. It carries the
// authoritative remaining time so the UI can count down from the server's
// clock rather than from a local timer.
type skipLockedPayload struct {
	MatchID     string    `json:"matchId"`
	RemainingMs int64     `json:"remainingMs"`
	SkipLockMs  int64     `json:"skipLockMs"`
	MatchedAt   time.Time `json:"matchedAt"`
	ServerTime  time.Time `json:"serverTime"`
}

// SkipMatch implements POST /api/matches/{id}/skip. The five second rule is
// enforced here, from the server's own match timestamp; a client cannot bypass
// it by calling the API directly or by rewriting its local clock.
func (a *API) SkipMatch(c *gin.Context) {
	sessionID := middleware.SessionIDFrom(c)
	matchID := c.Param("id")
	now := time.Now().UTC()

	res, err := a.Store.Skip(sessionID, matchID)
	switch {
	case err == nil:
	case errors.Is(err, store.ErrUnknownSession):
		utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
			"Your session expired. Reload the page.", nil)
		return
	case errors.Is(err, store.ErrNotInMatch):
		utils.ErrorResponseCode(c, http.StatusNotFound, "match_not_found",
			"That call is no longer active.", nil)
		return
	default:
		a.Logger.Error("skip failed", "error", err, "session", sessionID)
		utils.ErrorResponseCode(c, http.StatusInternalServerError, "skip_failed",
			"Could not end the call. Please try again.", nil)
		return
	}

	if res.TooEarly {
		seconds := int(math.Ceil(float64(res.RemainingMs) / 1000))
		if seconds < 1 {
			seconds = 1
		}
		c.Header("Retry-After", fmt.Sprintf("%d", seconds))
		utils.ErrorResponseCode(c, http.StatusConflict, "skip_locked",
			fmt.Sprintf("Skip is available in %d second%s.", seconds, plural(seconds)),
			skipLockedPayload{
				MatchID:     matchID,
				RemainingMs: res.RemainingMs,
				SkipLockMs:  a.SkipLock.Milliseconds(),
				ServerTime:  now,
			})
		return
	}

	utils.SuccessResponse(c, "Call ended.", endPayload{
		State:        string(models.StateEnded),
		MatchID:      res.MatchID,
		Reason:       models.ReasonSkipped,
		AlreadyEnded: res.AlreadyEnded,
		ServerTime:   now,
	})
}

// StopMatch implements POST /api/matches/{id}/stop. Stop is never gated: it
// works during the five second window, during connecting, and after the call
// already ended (idempotently).
func (a *API) StopMatch(c *gin.Context) {
	sessionID := middleware.SessionIDFrom(c)
	matchID := c.Param("id")

	res, err := a.Store.Stop(sessionID, matchID)
	switch {
	case err == nil:
	case errors.Is(err, store.ErrUnknownSession):
		utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
			"Your session expired. Reload the page.", nil)
		return
	case errors.Is(err, store.ErrNotInMatch):
		utils.ErrorResponseCode(c, http.StatusNotFound, "match_not_found",
			"That call is no longer active.", nil)
		return
	default:
		a.Logger.Error("stop failed", "error", err, "session", sessionID)
		utils.ErrorResponseCode(c, http.StatusInternalServerError, "stop_failed",
			"Could not end the call. Please try again.", nil)
		return
	}

	utils.SuccessResponse(c, "Call ended.", endPayload{
		State:        string(models.StateEnded),
		MatchID:      res.MatchID,
		Reason:       models.ReasonStopped,
		AlreadyEnded: res.AlreadyEnded,
		ServerTime:   time.Now().UTC(),
	})
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
