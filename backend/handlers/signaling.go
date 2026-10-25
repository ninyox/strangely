package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"strgback/config"
	"strgback/middleware"
	"strgback/models"
	"strgback/store"
	"strgback/utils"
)

// Signaling limits. A browser message is a small JSON envelope around an SDP
// blob or an ICE candidate, so 64 KiB is generous and anything larger is
// refused before it is parsed.
const (
	maxSignalingBytes = 64 << 10
	writeWait         = 10 * time.Second
	pongWait          = 35 * time.Second
	pingPeriod        = 20 * time.Second
	sendBuffer        = 32
)

// Message types a browser may send.
const (
	msgOffer  = "offer"
	msgAnswer = "answer"
	msgICE    = "ice"
	msgReady  = "ready"
	msgBye    = "bye"
	msgPing   = "ping"
)

// clientMessage is the envelope every inbound signaling message uses.
type clientMessage struct {
	Type    string          `json:"type"`
	MatchID string          `json:"matchId"`
	Payload json.RawMessage `json:"payload"`
}

// serverMessage is the envelope every outbound signaling message uses.
type serverMessage struct {
	Type       string    `json:"type"`
	Data       any       `json:"data,omitempty"`
	ServerTime time.Time `json:"serverTime"`
}

// Hub owns every live signaling socket. It is the store's Sink, so the store
// can push state changes (matched, partner_left) without knowing that
// WebSockets exist.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*wsClient

	st      *store.Store
	limiter *middleware.Limiter
	logger  *slog.Logger

	upgrader websocket.Upgrader
}

// wsClient is one browser connection.
type wsClient struct {
	hub       *Hub
	sessionID string
	conn      *websocket.Conn

	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

// NewHub builds the hub and its upgrader.
func NewHub(st *store.Store, limiter *middleware.Limiter, logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}

	return &Hub{
		clients: make(map[string]*wsClient),
		st:      st,
		limiter: limiter,
		logger:  logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			// The Origin header is checked in ServeWS before the upgrade,
			// against the configured allowlist, so the upgrader itself does
			// not need to repeat the check.
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

// Send implements store.Sink. It never blocks: a browser that cannot keep up
// with its own signaling traffic is disconnected instead of being buffered
// forever, and it will resync from /matchmaking/status when it reconnects.
func (h *Hub) Send(sessionID string, ev store.Event) {
	h.mu.RLock()
	client := h.clients[sessionID]
	h.mu.RUnlock()

	if client == nil {
		return
	}
	client.enqueue(ev)
}

// register adds a client, replacing (and closing) any socket the same session
// already had. One session, one live socket keeps delivery unambiguous.
func (h *Hub) register(client *wsClient) {
	h.mu.Lock()
	previous := h.clients[client.sessionID]
	h.clients[client.sessionID] = client
	h.mu.Unlock()

	if previous != nil && previous != client {
		previous.close()
	}

	h.st.MarkConnected(client.sessionID, true)
}

// unregister removes a client. When the socket was already replaced by a
// reconnect the session stays marked as connected, so a blip does not look
// like a closed tab.
func (h *Hub) unregister(client *wsClient) {
	h.mu.Lock()
	current := h.clients[client.sessionID]
	if current == client {
		delete(h.clients, client.sessionID)
	}
	h.mu.Unlock()

	if current != client {
		return
	}

	h.st.MarkConnected(client.sessionID, false)

	// Tell the partner that the other side dropped so the UI can show a
	// reconnecting state; the janitor ends the call if the socket never
	// comes back within MATCH_IDLE_TTL.
	if view, err := h.st.Status(client.sessionID); err == nil && view.MatchID != "" {
		h.st.NotifyPeerUnreachable(client.sessionID, view.MatchID)
	}
}

// Close shuts every socket down. main.go calls it during graceful shutdown.
func (h *Hub) Close() {
	h.mu.RLock()
	clients := make([]*wsClient, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		client.close()
	}
}

// enqueue marshals and hands a message to the client's writer. If the writer
// is gone, or its buffer is full, the socket is closed.
func (c *wsClient) enqueue(ev store.Event) {
	payload, err := json.Marshal(serverMessage{
		Type:       ev.Type,
		Data:       ev.Data,
		ServerTime: time.Now().UTC(),
	})
	if err != nil {
		c.hub.logger.Error("could not encode signaling message", "error", err, "type", ev.Type)
		return
	}

	select {
	case <-c.done:
	case c.send <- payload:
	default:
		c.hub.logger.Warn("signaling client too slow, closing", "session", c.sessionID)
		c.close()
	}
}

// close tears the connection down exactly once. The send channel is never
// closed, so a concurrent enqueue can never panic on a closed channel.
func (c *wsClient) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.hub.unregister(c)
		_ = c.conn.Close()
	})
}

// errorEvent is a small helper for user-safe error pushes.
func errorEvent(code, message string) store.Event {
	return store.Event{Type: "error", Data: map[string]any{"code": code, "message": message}}
}

// eventFor relays a peer's signaling message.
func relayEvent(msgType, matchID string, payload json.RawMessage) store.Event {
	return store.Event{Type: msgType, Data: map[string]any{
		"matchId": matchID,
		"payload": payload,
	}}
}

// ensure the hub satisfies the store's sink contract.
var _ store.Sink = (*Hub)(nil)

// ServeWS implements GET /api/signaling.
//
// Order matters here: the Origin is checked first, then the session cookie,
// and only then is the socket upgraded. Both checks happen before any traffic
// is accepted, and neither can be satisfied by a header a hostile page can
// forge on behalf of a visitor.
func (h *Hub) ServeWS(c *gin.Context) {
	if origin := c.GetHeader("Origin"); origin != "" {
		if _, ok := middleware.CORSHeaders(config.App.AllowedOrigins, origin); !ok {
			utils.ErrorResponseCode(c, http.StatusForbidden, "origin_not_allowed",
				"This origin may not open a signaling connection.", nil)
			return
		}
	}
	// A missing Origin is only possible for a non-browser client (the e2e
	// script, a load test). Browsers always send one, and a hostile page
	// sends its own, which the check above rejects.

	token, err := c.Cookie(config.App.CookieName)
	if err != nil || token == "" {
		utils.ErrorResponseCode(c, http.StatusUnauthorized, "no_session",
			"No session yet. Reload the page to start one.", nil)
		return
	}

	view, ok := h.st.Authenticate(token)
	if !ok {
		utils.ErrorResponseCode(c, http.StatusUnauthorized, "session_expired",
			"Your session expired. Reload the page.", nil)
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade has already written a response.
		h.logger.Debug("signaling upgrade failed", "error", err, "session", view.ID)
		return
	}

	client := &wsClient{
		hub:       h,
		sessionID: view.ID,
		conn:      conn,
		send:      make(chan []byte, sendBuffer),
		done:      make(chan struct{}),
	}

	h.register(client)
	defer client.close()

	go client.writeLoop()

	// Greet the client with who it is and where it stands, so a reconnect
	// resyncs without a separate round trip.
	status, _ := h.st.Status(view.ID)
	client.enqueue(store.Event{Type: "hello", Data: map[string]any{
		"sessionId":       view.ID,
		"username":        view.Username,
		"skipLockMs":      h.st.SkipLock().Milliseconds(),
		"maxMessageBytes": maxSignalingBytes,
		"state":           stateFromStatus(status, time.Now().UTC()),
	}})

	client.readLoop()
}

// readLoop consumes inbound messages until the socket closes or misbehaves.
func (c *wsClient) readLoop() {
	c.conn.SetReadLimit(maxSignalingBytes)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

	c.conn.SetPongHandler(func(string) error {
		c.hub.st.TouchSignal(c.sessionID)
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		c.hub.st.TouchSignal(c.sessionID)

		// Per session message budget: signaling is chatty by nature (ICE
		// candidates), so the budget is high, but not unlimited.
		if c.hub.limiter != nil && !c.hub.limiter.Allow(c.sessionID) {
			c.enqueue(errorEvent("rate_limited", "Too many signaling messages."))
			continue
		}

		var msg clientMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.enqueue(errorEvent("invalid_message", "Malformed signaling message."))
			continue
		}

		c.handle(msg)
	}
}

// handle dispatches one validated message.
func (c *wsClient) handle(msg clientMessage) {
	switch msg.Type {
	case msgPing:
		c.enqueue(store.Event{Type: "pong", Data: map[string]any{}})

	case msgReady:
		if _, err := c.hub.st.MarkReady(c.sessionID, msg.MatchID); err != nil {
			c.enqueue(errorEvent("not_in_match", "That call is no longer active."))
		}

	case msgOffer, msgAnswer, msgICE:
		c.relay(msg)

	case msgBye:
		// An explicit goodbye, sent when the tab is closed or the page is
		// hidden, so the partner sees "Your partner left" straight away
		// instead of waiting for the janitor.
		if _, err := c.hub.st.Leave(c.sessionID, msg.MatchID, models.ReasonLeft); err != nil {
			c.enqueue(errorEvent("not_in_match", "That call is no longer active."))
		}

	default:
		c.enqueue(errorEvent("unknown_type", "Unsupported signaling message type."))
	}
}

// relay forwards an offer, answer or ICE candidate to the partner.
//
// The store decides whether that is allowed: the sender must be a participant
// of the match the message names, and the match must still be running. This is
// the single place where one browser can reach another, which is what makes
// cross-session signaling impossible.
func (c *wsClient) relay(msg clientMessage) {
	if len(msg.Payload) == 0 {
		c.enqueue(errorEvent("empty_payload", "Signaling messages need a payload."))
		return
	}

	target, err := c.hub.st.RelayTarget(c.sessionID, msg.MatchID)
	if err != nil {
		c.enqueue(errorEvent("not_in_match", "That call is no longer active."))
		return
	}

	c.hub.Send(target, relayEvent(msg.Type, msg.MatchID, msg.Payload))
}

// writeLoop owns every write on the socket, which gorilla/websocket requires.
func (c *wsClient) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				c.close()
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close()
				return
			}
		}
	}
}
