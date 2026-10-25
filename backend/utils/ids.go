package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// RandomToken returns nBytes of cryptographically secure randomness, hex
// encoded. Session tokens and match ids are both drawn from here: neither is
// guessable, and neither encodes anything about the visitor.
func RandomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken is the one-way mapping between the token handed to a browser and
// the value kept in memory. Only the hash is ever stored, so a memory dump or a
// log line cannot be replayed as a session.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ShortID truncates an id or hash for logs and event payloads. It is long
// enough to correlate entries and far too short to be used as a credential.
func ShortID(value string) string {
	const length = 16
	if len(value) <= length {
		return value
	}
	return value[:length]
}
