package server

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrNotImplemented is returned by stub methods that have not yet been implemented.
var ErrNotImplemented = errors.New("not implemented")

// ErrPairingNotFound is returned when a pairing request ID is not recognized.
var ErrPairingNotFound = errors.New("pairing request not found")

// ErrPairingExpired is returned when an action is attempted on an expired pairing request.
var ErrPairingExpired = errors.New("pairing request expired")

// ErrPairingResolved is returned when an action is attempted on an already-resolved pairing request.
var ErrPairingResolved = errors.New("pairing request already resolved")

// PairingRequest represents a short-lived TOFU pairing request.
type PairingRequest struct {
	ID        uuid.UUID
	Label     string
	Code      string // 6-digit numeric (e.g., "472859")
	ExpiresAt time.Time
	Status    string // "pending", "approved", "denied", "expired"
	Token     string // plaintext bearer token, set only on approval
}

// PairingStore manages short-lived pairing requests in memory.
type PairingStore struct {
	mu       sync.Mutex
	requests map[uuid.UUID]*PairingRequest
}

// NewPairingStore creates a PairingStore ready to accept pairing requests.
func NewPairingStore() *PairingStore {
	return &PairingStore{
		requests: make(map[uuid.UUID]*PairingRequest),
	}
}

// Create generates a new pairing request with a 6-digit code, 60-second expiry, and status "pending".
func (ps *PairingStore) Create(label string) *PairingRequest {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		n = big.NewInt(0)
	}

	req := &PairingRequest{
		ID:        uuid.New(),
		Label:     label,
		Code:      fmt.Sprintf("%06d", n.Int64()),
		ExpiresAt: time.Now().Add(60 * time.Second),
		Status:    "pending",
	}
	ps.requests[req.ID] = req

	return &PairingRequest{
		ID:        req.ID,
		Label:     req.Label,
		Code:      req.Code,
		ExpiresAt: req.ExpiresAt,
		Status:    req.Status,
	}
}

// Get retrieves a pairing request by ID. It lazily marks expired requests.
// Returns a copy of the request and true if found, or nil and false if not.
func (ps *PairingStore) Get(id uuid.UUID) (*PairingRequest, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	req, ok := ps.requests[id]
	if !ok {
		return nil, false
	}

	if req.Status == "pending" && time.Now().After(req.ExpiresAt) {
		req.Status = "expired"
	}

	return &PairingRequest{
		ID:        req.ID,
		Label:     req.Label,
		Code:      req.Code,
		ExpiresAt: req.ExpiresAt,
		Status:    req.Status,
		Token:     req.Token,
	}, true
}

// Approve marks a pairing request as approved and stores the bearer token.
func (ps *PairingStore) Approve(id uuid.UUID, token string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	req, ok := ps.requests[id]
	if !ok {
		return ErrPairingNotFound
	}

	if req.Status != "pending" {
		return ErrPairingResolved
	}

	if time.Now().After(req.ExpiresAt) {
		req.Status = "expired"
		return ErrPairingExpired
	}

	req.Status = "approved"
	req.Token = token
	return nil
}

// Deny marks a pairing request as denied.
func (ps *PairingStore) Deny(id uuid.UUID) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	req, ok := ps.requests[id]
	if !ok {
		return ErrPairingNotFound
	}

	if req.Status != "pending" {
		return ErrPairingResolved
	}

	if time.Now().After(req.ExpiresAt) {
		req.Status = "expired"
		return ErrPairingExpired
	}

	req.Status = "denied"
	return nil
}
