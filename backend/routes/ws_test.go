package routes_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSignalingRejectsMalformedAndOversizedMessages(t *testing.T) {
	env := newTestEnv(t)

	_, ada, adaConn, _, matchID := matchedClients(t, env)

	// Malformed JSON gets an error, not a dropped socket.
	if err := adaConn.WriteMessage(websocket.TextMessage, []byte("this is not json")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	refusal := readUntil(t, adaConn, "error")
	var refusalData struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(refusal.Data, &refusalData); err != nil {
		t.Fatalf("could not decode the error payload: %v", err)
	}
	if refusalData.Code != "invalid_message" {
		t.Errorf("code = %q, want invalid_message", refusalData.Code)
	}

	// A message over the read limit closes the connection.
	huge := strings.Repeat("a", 70*1024)
	if err := adaConn.WriteMessage(websocket.TextMessage, []byte(huge)); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_ = adaConn.SetReadDeadline(time.Now().Add(4 * time.Second))
	for {
		if _, _, err := adaConn.ReadMessage(); err != nil {
			// The server closed the socket, which is the expected outcome.
			break
		}
	}

	// The call itself survives: the client reconnects and resyncs.
	if state := ada.status(); state.MatchID != matchID {
		t.Errorf("state after the oversized message = %+v, want the call to still exist", state)
	}
}

func TestSignalingGoodbyeEndsTheCallForThePartner(t *testing.T) {
	env := newTestEnv(t)

	_, grace, adaConn, graceConn, matchID := matchedClients(t, env)

	// Ada closes the tab and says goodbye on the way out.
	sendJSON(t, adaConn, map[string]any{"type": "bye", "matchId": matchID})

	left := readUntil(t, graceConn, "partner_left")
	var data struct {
		Reason  string `json:"reason"`
		MatchID string `json:"matchId"`
	}
	if err := json.Unmarshal(left.Data, &data); err != nil {
		t.Fatalf("could not decode the partner_left payload: %v", err)
	}
	if data.Reason != "left" {
		t.Errorf("reason = %q, want left", data.Reason)
	}
	if data.MatchID != matchID {
		t.Errorf("match = %q, want %q", data.MatchID, matchID)
	}

	state := grace.status()
	if state.State != "ended" || state.MatchID != "" {
		t.Fatalf("grace state = %+v, want ended and free to match again", state)
	}
}

func TestSignalingUnknownMessageTypeIsRejected(t *testing.T) {
	env := newTestEnv(t)

	_, _, adaConn, _, matchID := matchedClients(t, env)

	sendJSON(t, adaConn, map[string]any{"type": "nonsense", "matchId": matchID})

	refusal := readUntil(t, adaConn, "error")
	var data struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(refusal.Data, &data); err != nil {
		t.Fatalf("could not decode the error payload: %v", err)
	}
	if data.Code != "unknown_type" {
		t.Errorf("code = %q, want unknown_type", data.Code)
	}
}
