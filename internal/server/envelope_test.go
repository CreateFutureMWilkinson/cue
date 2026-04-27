package server_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

type EnvelopeSuite struct {
	suite.Suite
}

func TestEnvelope(t *testing.T) {
	suite.Run(t, new(EnvelopeSuite))
}

// ---------------------------------------------------------------------------
// Behavior 1: ActivityEnvelope marshals to stable JSON matching the contract
// ---------------------------------------------------------------------------

func (s *EnvelopeSuite) TestActivityEnvelopeMarshalJSON() {
	ts := time.Date(2026, 4, 15, 14, 30, 0, 0, time.UTC)

	s.Run("zero dropped_since_last is omitted", func() {
		env := server.ActivityEnvelope{
			Seq:       42,
			Type:      "activity",
			Timestamp: ts,
			Data: server.ActivityData{
				Source:  "slack",
				Message: "Polled 5 messages from #general",
				IsError: false,
			},
			DroppedSinceLast: 0,
		}

		raw, err := json.Marshal(env)
		s.Require().NoError(err)

		var m map[string]any
		s.Require().NoError(json.Unmarshal(raw, &m))

		// seq must be present as a number equal to 42
		s.InDelta(42, m["seq"], 0.001, "seq must be 42")

		// type must be the literal string "activity"
		s.Equal("activity", m["type"], "type must be \"activity\"")

		// timestamp must be RFC 3339 UTC
		s.Equal("2026-04-15T14:30:00Z", m["timestamp"], "timestamp must be RFC 3339 UTC")

		// data must nest ActivityData fields
		data, ok := m["data"].(map[string]any)
		s.Require().True(ok, "data must be a nested object")
		s.Equal("slack", data["source"])
		s.Equal("Polled 5 messages from #general", data["message"])
		s.Equal(false, data["is_error"])

		// dropped_since_last must be absent when zero (omitempty)
		_, present := m["dropped_since_last"]
		s.False(present, "dropped_since_last must be omitted when zero")
	})

	s.Run("non-zero dropped_since_last is present", func() {
		env := server.ActivityEnvelope{
			Seq:       43,
			Type:      "activity",
			Timestamp: ts,
			Data: server.ActivityData{
				Source:  "email",
				Message: "Fetched 3 new emails",
				IsError: false,
			},
			DroppedSinceLast: 5,
		}

		raw, err := json.Marshal(env)
		s.Require().NoError(err)

		var m map[string]any
		s.Require().NoError(json.Unmarshal(raw, &m))

		// dropped_since_last must appear and equal 5
		s.InDelta(5, m["dropped_since_last"], 0.001, "dropped_since_last must be 5")
	})
}
