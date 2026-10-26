package routes_test

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// iceServerData mirrors one entry of the public config's iceServers array.
type iceServerData struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username"`
	Credential string   `json:"credential"`
}

type publicConfigData struct {
	IceServers []iceServerData `json:"iceServers"`
}

// readPublicConfig fetches /api/config, which is what a browser asks for before
// it can start a call.
func readPublicConfig(t *testing.T, env *testEnv) publicConfigData {
	t.Helper()

	code, body := newClient(t, env).get("/api/config")
	if code != http.StatusOK {
		t.Fatalf("/api/config = %d, want 200", code)
	}
	return decode[publicConfigData](t, body.Data)
}

// turnEntry finds the TURN entry, if the config carries one.
func turnEntry(cfg publicConfigData) *iceServerData {
	for i := range cfg.IceServers {
		if len(cfg.IceServers[i].URLs) == 0 {
			continue
		}
		url := cfg.IceServers[i].URLs[0]
		if strings.HasPrefix(url, "turn:") || strings.HasPrefix(url, "turns:") {
			return &cfg.IceServers[i]
		}
	}
	return nil
}

func TestPublicConfigCarriesEveryTurnURLWithItsStaticCredential(t *testing.T) {
	env := newTestEnvWith(t, map[string]string{
		// The shape a hosted relay publishes: UDP, TCP, and TLS on 443, all
		// sharing one static credential.
		"TURN_URLS":       "turn:relay.example.test:80,turn:relay.example.test:80?transport=tcp,turns:relay.example.test:443?transport=tcp",
		"TURN_USERNAME":   "strangerly",
		"TURN_CREDENTIAL": "hosted-relay-credential",
	})

	cfg := readPublicConfig(t, env)

	turn := turnEntry(cfg)
	if turn == nil {
		t.Fatal("the config should carry a TURN entry when TURN_URLS is set")
	}
	if len(turn.URLs) != 3 {
		t.Fatalf("turn urls = %v, want all three transports", turn.URLs)
	}
	if turn.URLs[2] != "turns:relay.example.test:443?transport=tcp" {
		t.Errorf("third turn url = %q, want the TLS transport", turn.URLs[2])
	}
	if turn.Username != "strangerly" || turn.Credential != "hosted-relay-credential" {
		t.Errorf("turn credentials = %q / %q, want the configured pair", turn.Username, turn.Credential)
	}
}

func TestPublicConfigMintsShortLivedCredentialsFromASharedSecret(t *testing.T) {
	env := newTestEnvWith(t, map[string]string{
		"TURN_URLS":           "turn:coturn.example.test:3478?transport=udp,turn:coturn.example.test:3478?transport=tcp",
		"TURN_SHARED_SECRET":  "a-long-random-secret",
		"TURN_CREDENTIAL_TTL": "10m",
	})

	first := readPublicConfig(t, env)
	turn := turnEntry(first)
	if turn == nil {
		t.Fatal("the config should carry a TURN entry when TURN_URLS is set")
	}

	// The username carries its own expiry, which is the scheme coturn expects.
	expiry, suffix, ok := strings.Cut(turn.Username, ":")
	if !ok || suffix == "" {
		t.Fatalf("turn username = %q, want an expiry and a suffix", turn.Username)
	}
	expirySeconds, err := strconv.ParseInt(expiry, 10, 64)
	if err != nil {
		t.Fatalf("turn username expiry = %q, want a unix timestamp", expiry)
	}
	want := time.Now().Add(10 * time.Minute).Unix()
	if expirySeconds < want-60 || expirySeconds > want+60 {
		t.Errorf("turn username expiry = %d, want about %d", expirySeconds, want)
	}

	// The credential is an HMAC-SHA1 of that username under the shared secret.
	raw, err := base64.StdEncoding.DecodeString(turn.Credential)
	if err != nil || len(raw) != 20 {
		t.Fatalf("turn credential = %q, want a base64 encoded HMAC-SHA1", turn.Credential)
	}

	// Every request mints a fresh credential, so no two visitors share one.
	second := readPublicConfig(t, env)
	if again := turnEntry(second); again == nil || again.Credential == turn.Credential {
		t.Error("a second request should mint a different short lived credential")
	}
}

func TestPublicConfigKeepsWorkingWithTheSingleTurnURLForm(t *testing.T) {
	env := newTestEnvWith(t, map[string]string{
		// The form this service read before TURN_URLS existed.
		"TURN_URL":        "turn:legacy.example.test:3478",
		"TURN_USERNAME":   "legacy",
		"TURN_CREDENTIAL": "legacy-credential",
	})

	turn := turnEntry(readPublicConfig(t, env))
	if turn == nil {
		t.Fatal("TURN_URL should still be read")
	}
	if len(turn.URLs) != 1 || turn.URLs[0] != "turn:legacy.example.test:3478" {
		t.Errorf("turn urls = %v, want the single configured URL", turn.URLs)
	}
}
