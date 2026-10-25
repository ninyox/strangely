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

// reportRequest is the body of POST /api/matches/{id}/report.
type reportRequest struct {
	Reason string `json:"reason"`
	Note   string `json:"note"`
}

// reportPayload acknowledges a report. It returns nothing that could be used
// to identify the other participant.
type reportPayload struct {
	ReportID   string    `json:"reportId"`
	MatchID    string    `json:"matchId"`
	ReceivedAt time.Time `json:"receivedAt"`
}

// ReportMatch implements POST /api/matches/{id}/report.
//
// Reporting is available at any point in a call, including inside the five
// second Skip window, and also for a moment after the call ended: the safety
// affordance must never be the thing that is locked.
func (a *API) ReportMatch(c *gin.Context) {
	sessionID := middleware.SessionIDFrom(c)
	matchID := c.Param("id")

	var req reportRequest
	if err := decodeJSON(c, &req); err != nil {
		utils.ErrorResponseCode(c, http.StatusBadRequest, "invalid_body",
			"Malformed request body.", nil)
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if !models.ValidReportReason(reason) {
		utils.ErrorResponseCode(c, http.StatusBadRequest, "invalid_reason",
			"Choose a reason from the list.", map[string]any{"allowed": models.ReportReasons})
		return
	}

	note := utils.SanitizeNote(req.Note)

	report, err := a.Store.SaveReport(sessionID, matchID, reason, note)
	switch {
	case err == nil:
	case errors.Is(err, store.ErrUnknownSession):
		utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
			"Your session expired. Reload the page.", nil)
		return
	case errors.Is(err, store.ErrReportRateLimited):
		utils.ErrorResponseCode(c, http.StatusTooManyRequests, "report_rate_limited",
			"You have already sent several reports. Please try again later.", nil)
		return
	case errors.Is(err, store.ErrNotInMatch):
		utils.ErrorResponseCode(c, http.StatusNotFound, "match_not_found",
			"That call is no longer active.", nil)
		return
	default:
		a.Logger.Error("report failed", "error", err, "session", sessionID)
		utils.ErrorResponseCode(c, http.StatusInternalServerError, "report_failed",
			"Could not send the report. Please try again.", nil)
		return
	}

	// Structured, deliberately thin: the reason and the length of any note.
	// The note itself is not logged, and no address or token is included.
	a.Logger.Warn("abuse report received",
		"match_id", report.MatchID,
		"reason", report.Reason,
		"reporter", report.ReporterID,
		"note_length", len(report.Note),
	)

	utils.SuccessResponse(c, "Thanks — your report was received.", reportPayload{
		ReportID:   report.ID,
		MatchID:    report.MatchID,
		ReceivedAt: report.CreatedAt,
	})
}
