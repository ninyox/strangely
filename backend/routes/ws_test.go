package routes_test

import (
	"encoding/json"
	"net/http"
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

func TestSignalingTellsTheSenderWhenThePartnerHasNoSocket(t *testing.T) {
	env := newTestEnv(t)

	ada := newClient(t, env)
	grace := newClient(t, env)
	ada.createSession("Ada")
	grace.createSession("Grace")

	// Only Ada opens a signaling socket.
	adaConn := dialSignaling(t, env, ada)
	readUntil(t, adaConn, "hello")

	if code, body := ada.join("Ada"); code != http.StatusOK {
		t.Fatalf("ada join = %d, %+v", code, body)
	}

	// Grace joins over HTTP and is matched without ever opening a socket, which
	// is exactly what a phone whose handshake was refused looks like.
	_, body := grace.join("Grace")
	matchID := decode[stateData](t, body.Data).MatchID
	if matchID == "" {
		t.Fatal("the two sessions should have matched")
	}
	readUntil(t, adaConn, "matched")

	// Ada's offer has nowhere to go. She is told immediately, instead of
	// staring at "Connecting video…" until her own ICE gives up on a partner
	// that was never reachable.
	sendJSON(t, adaConn, map[string]any{
		"type":    "offer",
		"matchId": matchID,
		"payload": map[string]any{"type": "offer", "sdp": "v=0"},
	})

	refusal := readUntil(t, adaConn, "error")
	var data struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(refusal.Data, &data); err != nil {
		t.Fatalf("could not decode the error payload: %v", err)
	}
	if data.Code != "partner_offline" {
		t.Errorf("code = %q, want partner_offline", data.Code)
	}

	// The call is not dead: once the partner's socket is up, the same relay
	// arrives, which is what a reconnect depends on.
	graceConn := dialSignaling(t, env, grace)
	readUntil(t, graceConn, "hello")

	sendJSON(t, adaConn, map[string]any{
		"type":    "offer",
		"matchId": matchID,
		"payload": map[string]any{"type": "offer", "sdp": "v=0"},
	})

	if relayed := readUntil(t, graceConn, "offer"); relayed.Type != "offer" {
		t.Errorf("relayed message = %q, want offer", relayed.Type)
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

func TestSignalingGoodbyeReasonsAreLimitedToWhatABrowserMayClaim(t *testing.T) {
	cases := []struct {
		sent string
		want string
	}{
		// A browser that gave up on a peer connection that never came up.
		{sent: "disconnected", want: "disconnected"},
		{sent: "left", want: "left"},
		// Anything else is not the client's story to tell.
		{sent: "too_long", want: "left"},
	}

	for _, tc := range cases {
		t.Run(tc.sent, func(t *testing.T) {
			env := newTestEnv(t)

			_, _, adaConn, graceConn, matchID := matchedClients(t, env)

			sendJSON(t, adaConn, map[string]any{
				"type":    "bye",
				"matchId": matchID,
				"payload": map[string]any{"reason": tc.sent},
			})

			left := readUntil(t, graceConn, "partner_left")
			var data struct {
				Reason string `json:"reason"`
			}
			if err := json.Unmarshal(left.Data, &data); err != nil {
				t.Fatalf("could not decode the partner_left payload: %v", err)
			}
			if data.Reason != tc.want {
				t.Errorf("partner_left reason = %q, want %q", data.Reason, tc.want)
			}
		})
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
