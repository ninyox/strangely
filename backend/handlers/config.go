package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"strgback/config"
	"strgback/models"
	"strgback/utils"
)

// iceServerPayload mirrors the RTCIceServer dictionary the browser expects, so
// the frontend can pass it straight to RTCPeerConnection.
type iceServerPayload struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// publicConfigPayload is everything the browser needs before it can start a
// call. It contains no secrets: TURN credentials, when used, are short lived
// and minted per request.
type publicConfigPayload struct {
	IceServers        []iceServerPayload `json:"iceServers"`
	SkipLockSeconds   float64            `json:"skipLockSeconds"`
	MaxSignalingBytes int                `json:"maxSignalingBytes"`
	ReportReasons     []string           `json:"reportReasons"`
	ServerTime        time.Time          `json:"serverTime"`
}

// PublicConfig implements GET /api/config.
func (a *API) PublicConfig(c *gin.Context) {
	utils.SuccessResponse(c, "", publicConfigPayload{
		IceServers:        buildIceServers(config.App.ICE),
		SkipLockSeconds:   a.SkipLock.Seconds(),
		MaxSignalingBytes: maxSignalingBytes,
		ReportReasons:     models.ReportReasons,
		ServerTime:        time.Now().UTC(),
	})
}

// buildIceServers turns configuration into the list the browser receives. A
// STUN-only setup is perfectly usable locally; TURN is what makes calls work
// across restrictive NATs in production.
func buildIceServers(cfg config.ICEConfig) []iceServerPayload {
	servers := make([]iceServerPayload, 0, 2)

	if len(cfg.STUNURLs) > 0 {
		servers = append(servers, iceServerPayload{URLs: cfg.STUNURLs})
	}

	if cfg.TURNURL == "" {
		return servers
	}

	// Preferred: derive credentials from a shared secret so no static
	// credential ever reaches the browser or the frontend bundle.
	if cfg.TURNSharedSecret != "" {
		username, credential := turnCredentials(cfg, time.Now().UTC())
		servers = append(servers, iceServerPayload{
			URLs:       []string{cfg.TURNURL},
			Username:   username,
			Credential: credential,
		})
		return servers
	}

	// Fallback for a TURN server that only supports static credentials.
	// Documented in README.md as a development convenience.
	if cfg.TURNUsername != "" && cfg.TURNCredential != "" {
		servers = append(servers, iceServerPayload{
			URLs:       []string{cfg.TURNURL},
			Username:   cfg.TURNUsername,
			Credential: cfg.TURNCredential,
		})
	}

	return servers
}

// turnCredentials implements the "time limited credentials" scheme TURN
// servers such as coturn expect: the username carries an expiry, and the
// credential is an HMAC of that username under a secret only the server knows.
func turnCredentials(cfg config.ICEConfig, now time.Time) (string, string) {
	expiry := now.Add(cfg.TURNCredentialTTL).Unix()

	suffix, err := utils.RandomToken(6)
	if err != nil {
		suffix = "strangerly"
	}

	username := fmt.Sprintf("%d:%s", expiry, suffix)

	mac := hmac.New(sha1.New, []byte(cfg.TURNSharedSecret))
	mac.Write([]byte(username))
	credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return username, credential
}

// NotFound responds with a JSON 404 for unknown API routes.
func (a *API) NotFound(c *gin.Context) {
	utils.ErrorResponseCode(c, http.StatusNotFound, "not_found", "Unknown endpoint.", nil)
}
