package store

import (
	"time"

	"strgback/models"
)

// MatchPayload is what a matched client receives: who it is talking to, which
// side creates the WebRTC offer, and how long Skip stays locked.
type MatchPayload struct {
	MatchID         string    `json:"matchId"`
	PartnerUsername string    `json:"partnerUsername"`
	Initiator       bool      `json:"initiator"`
	MatchedAt       time.Time `json:"matchedAt"`
	SkipLockMs      int64     `json:"skipLockMs"`
	ServerTime      time.Time `json:"serverTime"`
}

// JoinResult is the answer to "put me in the queue".
type JoinResult struct {
	State           models.State
	AlreadyQueued   bool
	InMatch         bool
	QueuePosition   int
	MatchID         string
	PartnerUsername string
	MatchedAt       time.Time
	Initiator       bool
	SkipLockMs      int64
}

// StatusView is the full state of one session, used by /matchmaking/status.
type StatusView struct {
	State            models.State
	MatchID          string
	PartnerUsername  string
	MatchedAt        time.Time
	Initiator        bool
	Ready            bool
	PeerReady        bool
	SkipLockMs       int64
	RemainingMs      int64
	QueuedAt         time.Time
	QueuePosition    int
	LastEndedReason  string
	LastEndedMatchID string
}
