package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// wsMessage is the envelope the signaling socket sends.
type wsMessage struct {
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	ServerTime time.Time       `json:"serverTime"`
}

// dialSignaling opens the signaling socket as the given client, with the
// session cookie and the configured Origin.
func dialSignaling(t *testing.T, env *testEnv, c *client) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(env.server.URL, "http") + "/api/signaling"

	header := http.Header{}
	header.Set("Origin", testOrigin)

	if cookie := c.sessionCookie(); cookie != nil {
		header.Set("Cookie", cookie.Name+"="+cookie.Value)
	}

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("could not open the signaling socket: %v (status %d)", err, status)
	}

	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readMessage(t *testing.T, conn *websocket.Conn) wsMessage {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(4 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("reading a signaling message failed: %v", err)
	}

	var msg wsMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("could not decode signaling message %s: %v", raw, err)
	}
	return msg
}

// readUntil skips the messages a test does not care about (notably the initial
// hello) and returns the first message of the wanted type.
func readUntil(t *testing.T, conn *websocket.Conn, msgType string) wsMessage {
	t.Helper()

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if msg := readMessage(t, conn); msg.Type == msgType {
			return msg
		}
	}

	t.Fatalf("never received a %q message", msgType)
	return wsMessage{}
}

// matchedClients creates two clients that are already in a call, opens their
// signaling sockets, and drains the greeting messages.
func matchedClients(t *testing.T, env *testEnv) (*client, *client, *websocket.Conn, *websocket.Conn, string) {
	t.Helper()

	ada := newClient(t, env)
	grace := newClient(t, env)

	ada.createSession("Ada")
	grace.createSession("Grace")

	ada.join("Ada")
	_, body := grace.join("Grace")
	matchID := decode[stateData](t, body.Data).MatchID

	adaConn := dialSignaling(t, env, ada)
	graceConn := dialSignaling(t, env, grace)

	// Both sockets greet their client before anything else.
	if hello := readUntil(t, adaConn, "hello"); len(hello.Data) == 0 {
		t.Fatal("the hello message should describe the session")
	}
	readUntil(t, graceConn, "hello")

	return ada, grace, adaConn, graceConn, matchID
}

// sendJSON writes a signaling message.
func sendJSON(t *testing.T, conn *websocket.Conn, payload any) {
	t.Helper()

	if err := conn.WriteJSON(payload); err != nil {
		t.Fatalf("could not send a signaling message: %v", err)
	}
}

func TestSignalingRefusesUnknownOriginsAndMissingSessions(t *testing.T) {
	env := newTestEnv(t)
	plain := &http.Client{Timeout: 10 * time.Second}

	// A hostile page: the Origin is not in the allowlist, so the upgrade is
	// refused before anything else happens.
	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/api/signaling", nil)
	if err != nil {
		t.Fatalf("could not build request: %v", err)
	}
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	resp, err := plain.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unknown origin status = %d, want 403", resp.StatusCode)
	}

	// The right origin but no session cookie is still refused.
	req, err = http.NewRequest(http.MethodGet, env.server.URL+"/api/signaling", nil)
	if err != nil {
		t.Fatalf("could not build request: %v", err)
	}
	req.Header.Set("Origin", testOrigin)

	resp, err = plain.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing session status = %d, want 401", resp.StatusCode)
	}
}

func TestSignalingRelaysOnlyBetweenTheMatchedPair(t *testing.T) {
	env := newTestEnv(t)

	ada, grace, adaConn, graceConn, matchID := matchedClients(t, env)

	// Readiness. Whichever side the server processes second triggers the
	// notification to the other one, so Ada is made definitively first: wait
	// until the server has recorded her readiness before Grace reports in.
	sendJSON(t, adaConn, map[string]any{"type": "ready", "matchId": matchID})

	deadline := time.Now().Add(3 * time.Second)
	for {
		if ada.status().Ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the server never recorded Ada's readiness")
		}
		time.Sleep(25 * time.Millisecond)
	}

	sendJSON(t, graceConn, map[string]any{"type": "ready", "matchId": matchID})

	// Grace marked ready second, while Ada was already ready, so Ada is told.
	readUntil(t, adaConn, "partner_ready")

	if state := ada.status(); state.State != "matched" || !state.PeerReady {
		t.Fatalf("ada state = %+v, want matched with a ready peer", state)
	}
	if state := grace.status(); state.State != "matched" {
		t.Fatalf("grace state = %+v, want matched", state)
	}

	// An offer travels to the partner untouched.
	offerSDP := "v=0\r\no=- 1 1 IN IP4 127.0.0.1\r\n"
	sendJSON(t, adaConn, map[string]any{
		"type":    "offer",
		"matchId": matchID,
		"payload": map[string]any{"type": "offer", "sdp": offerSDP},
	})

	relayed := readUntil(t, graceConn, "offer")
	var data struct {
		MatchID string          `json:"matchId"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(relayed.Data, &data); err != nil {
		t.Fatalf("could not decode the relayed offer: %v", err)
	}
	if data.MatchID != matchID {
		t.Errorf("relayed match = %q, want %q", data.MatchID, matchID)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data.Payload, &decoded); err != nil {
		t.Fatalf("could not decode the relayed payload: %v", err)
	}
	if decoded["sdp"] != offerSDP {
		t.Errorf("relayed sdp = %v, want %v", decoded["sdp"], offerSDP)
	}

	// An answer travels back.
	sendJSON(t, graceConn, map[string]any{
		"type":    "answer",
		"matchId": matchID,
		"payload": map[string]any{"type": "answer", "sdp": "v=0\r\nanswer"},
	})
	readUntil(t, adaConn, "answer")

	// ICE candidates travel both ways.
	sendJSON(t, adaConn, map[string]any{
		"type":    "ice",
		"matchId": matchID,
		"payload": map[string]any{"candidate": "candidate:1 1 udp 1 127.0.0.1 1 typ host"},
	})
	readUntil(t, graceConn, "ice")

	// A third session cannot reach into the call.
	outsider := newClient(t, env)
	outsider.createSession("Outsider")
	outsiderConn := dialSignaling(t, env, outsider)
	readUntil(t, outsiderConn, "hello")

	sendJSON(t, outsiderConn, map[string]any{
		"type":    "offer",
		"matchId": matchID,
		"payload": map[string]any{"type": "offer", "sdp": "v=0\r\nspoofed"},
	})

	refusal := readUntil(t, outsiderConn, "error")
	var refusalData struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(refusal.Data, &refusalData); err != nil {
		t.Fatalf("could not decode the error payload: %v", err)
	}
	if refusalData.Code != "not_in_match" {
		t.Errorf("refusal code = %q, want not_in_match", refusalData.Code)
	}
}
