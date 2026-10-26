package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"strgback/models"
)

func TestByeReasonIsLimitedToWhatABrowserMayClaim(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "no payload at all", payload: "", want: models.ReasonLeft},
		{name: "the tab was closed", payload: `{"reason":"left"}`, want: models.ReasonLeft},
		{name: "the peer connection failed", payload: `{"reason":"disconnected"}`, want: models.ReasonDisconnected},
		{
			name:    "with diagnostics attached",
			payload: `{"reason":"disconnected","diagnostics":{"ice":"failed","connection":"failed","localCandidates":6,"remoteCandidates":0,"remoteDescription":true,"elapsedMs":28450}}`,
			want:    models.ReasonDisconnected,
		},
		{name: "a reason the client does not get to choose", payload: `{"reason":"too_long"}`, want: models.ReasonLeft},
		{name: "a body that is not JSON", payload: `not json`, want: models.ReasonLeft},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			details := parseBye(json.RawMessage(tc.payload))
			if got := details.reason(); got != tc.want {
				t.Errorf("reason = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestByeDiagnosticsCannotWriteAnythingIntoTheLog(t *testing.T) {
	details := parseBye(json.RawMessage(`{
		"reason":"disconnected",
		"diagnostics":{
			"ice":"<script>alert(1)</script>",
			"connection":"totally-made-up",
			"localCandidates":-40,
			"remoteCandidates":900000,
			"remoteDescription":true,
			"elapsedMs":-1
		}
	}`))

	if details.Diagnostics == nil {
		t.Fatal("the diagnostics should have been read")
	}
	if got := iceState(details.Diagnostics.ICE); got != "unknown" {
		t.Errorf("ice = %q, want unknown", got)
	}
	if got := connectionState(details.Diagnostics.Connection); got != "unknown" {
		t.Errorf("connection = %q, want unknown", got)
	}
	if got := clampCount(details.Diagnostics.LocalCandidates); got != 0 {
		t.Errorf("local candidates = %d, want 0", got)
	}
	if got := clampCount(details.Diagnostics.RemoteCandidates); got != 10_000 {
		t.Errorf("remote candidates = %d, want the cap", got)
	}
	if got := clampElapsedMs(details.Diagnostics.ElapsedMs); got != 0 {
		t.Errorf("elapsed = %d, want 0", got)
	}
	if got := clampElapsedMs(int64(72 * time.Hour / time.Millisecond)); got != int64(24*time.Hour/time.Millisecond) {
		t.Errorf("elapsed = %d, want the cap", got)
	}

	// The states a browser really reports are kept as they are.
	for _, state := range []string{"new", "checking", "connected", "completed", "disconnected", "failed", "closed"} {
		if got := iceState(state); got != state {
			t.Errorf("iceState(%q) = %q", state, got)
		}
	}
	for _, state := range []string{"new", "connecting", "connected", "disconnected", "failed", "closed"} {
		if got := connectionState(state); got != state {
			t.Errorf("connectionState(%q) = %q", state, got)
		}
	}
}
