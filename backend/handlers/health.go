package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"strgback/utils"
)

// healthPayload is the body of /healthz and /readyz.
type healthPayload struct {
	Status        string      `json:"status"`
	UptimeSeconds int64       `json:"uptimeSeconds"`
	ServerTime    time.Time   `json:"serverTime"`
	Stats         *storeStats `json:"stats,omitempty"`
}

// storeStats is the in-memory counters worth watching on a deployment.
type storeStats struct {
	Sessions          int    `json:"sessions"`
	Queued            int    `json:"queued"`
	ActiveMatches     int    `json:"activeMatches"`
	Reports           int    `json:"reports"`
	SessionsCreated   uint64 `json:"sessionsCreated"`
	MatchesCreated    uint64 `json:"matchesCreated"`
	MatchesEnded      uint64 `json:"matchesEnded"`
	StaleQueueDropped uint64 `json:"staleQueueDropped"`
}

// Health implements GET /healthz: the process is up and serving.
func (a *API) Health(c *gin.Context) {
	utils.SuccessResponse(c, "", healthPayload{
		Status:        "ok",
		UptimeSeconds: a.Store.Stats().UptimeSeconds,
		ServerTime:    time.Now().UTC(),
	})
}

// Ready implements GET /readyz. There is no database to check in the first
// release, so readiness is "the store exists and the process is not draining".
func (a *API) Ready(c *gin.Context) {
	stats := a.Store.Stats()

	payload := healthPayload{
		Status:        "ready",
		UptimeSeconds: stats.UptimeSeconds,
		ServerTime:    time.Now().UTC(),
		Stats: &storeStats{
			Sessions:          stats.Sessions,
			Queued:            stats.Queued,
			ActiveMatches:     stats.ActiveMatches,
			Reports:           stats.Reports,
			SessionsCreated:   stats.Counters.SessionsCreated,
			MatchesCreated:    stats.Counters.MatchesCreated,
			MatchesEnded:      stats.Counters.MatchesEnded,
			StaleQueueDropped: stats.Counters.StaleQueueDropped,
		},
	}

	if !a.IsReady() {
		payload.Status = "draining"
		c.JSON(http.StatusServiceUnavailable, utils.ApiResponse{
			Success: false,
			Code:    "not_ready",
			Message: "Shutting down.",
			Data:    payload,
		})
		return
	}

	utils.SuccessResponse(c, "", payload)
}
