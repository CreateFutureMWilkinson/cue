package handler_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/stretchr/testify/suite"
)

// stubHistoryProvider implements handler.HistoryProvider with canned JSON.
type stubHistoryProvider struct {
	data []byte
	err  error
}

func (s *stubHistoryProvider) HistoryJSON(sinceSeq uint64) ([]byte, error) {
	return s.data, s.err
}

// ---------- suite ----------

type EventsHandlerSuite struct {
	suite.Suite
}

func TestEventsHandler(t *testing.T) {
	suite.Run(t, new(EventsHandlerSuite))
}

// ---------- tests ----------

func (s *EventsHandlerSuite) TestMissingSinceReturns400() {
	provider := &stubHistoryProvider{}
	srv := httptest.NewServer(handler.EventsHandler(provider))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/events")
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var errResp map[string]string
	s.Require().NoError(json.Unmarshal(body, &errResp))
	s.Equal("invalid since parameter", errResp["error"])
}

func (s *EventsHandlerSuite) TestNonNumericSinceReturns400() {
	provider := &stubHistoryProvider{}
	srv := httptest.NewServer(handler.EventsHandler(provider))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/events?since=abc")
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var errResp map[string]string
	s.Require().NoError(json.Unmarshal(body, &errResp))
	s.Equal("invalid since parameter", errResp["error"])
}

func (s *EventsHandlerSuite) TestNegativeSinceReturns400() {
	provider := &stubHistoryProvider{}
	srv := httptest.NewServer(handler.EventsHandler(provider))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/events?since=-1")
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var errResp map[string]string
	s.Require().NoError(json.Unmarshal(body, &errResp))
	s.Equal("invalid since parameter", errResp["error"])
}

func (s *EventsHandlerSuite) TestValidSinceEmptyHistory() {
	canned := `{"events":[],"truncated":false,"oldest_seq":0,"latest_seq":0}`
	provider := &stubHistoryProvider{data: []byte(canned)}
	srv := httptest.NewServer(handler.EventsHandler(provider))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/events?since=0")
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(resp.Header.Get("Content-Type"), "application/json")

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var result struct {
		Events    []json.RawMessage `json:"events"`
		Truncated bool              `json:"truncated"`
		OldestSeq uint64            `json:"oldest_seq"`
		LatestSeq uint64            `json:"latest_seq"`
	}
	s.Require().NoError(json.Unmarshal(body, &result))
	s.Empty(result.Events)
	s.False(result.Truncated)
	s.Equal(uint64(0), result.OldestSeq)
	s.Equal(uint64(0), result.LatestSeq)
}

func (s *EventsHandlerSuite) TestValidSinceWithEvents() {
	canned := `{"events":[{"seq":1},{"seq":2},{"seq":3}],"truncated":false,"oldest_seq":1,"latest_seq":3}`
	provider := &stubHistoryProvider{data: []byte(canned)}
	srv := httptest.NewServer(handler.EventsHandler(provider))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/events?since=0")
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)
	s.Contains(resp.Header.Get("Content-Type"), "application/json")

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var result struct {
		Events    []json.RawMessage `json:"events"`
		Truncated bool              `json:"truncated"`
		OldestSeq uint64            `json:"oldest_seq"`
		LatestSeq uint64            `json:"latest_seq"`
	}
	s.Require().NoError(json.Unmarshal(body, &result))
	s.Len(result.Events, 3)
	s.False(result.Truncated)
	s.Equal(uint64(1), result.OldestSeq)
	s.Equal(uint64(3), result.LatestSeq)
}
