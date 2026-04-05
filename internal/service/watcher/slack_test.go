package watcher_test

import (
	"context"
	"errors"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/watcher"
	"github.com/stretchr/testify/suite"
)

// mockSlackAPI implements watcher.SlackAPI for testing.
type mockSlackAPI struct {
	channels      []watcher.SlackChannel
	channelsErr   error
	messages      map[string][]watcher.SlackMessage // channelID -> messages
	messagesErr   error
	threadReplies map[string][]watcher.SlackMessage // threadTS -> replies
	threadErr     error
}

func (m *mockSlackAPI) GetUserChannels(ctx context.Context) ([]watcher.SlackChannel, error) {
	return m.channels, m.channelsErr
}

func (m *mockSlackAPI) GetChannelMessages(ctx context.Context, channelID string, oldest string) ([]watcher.SlackMessage, error) {
	if m.messagesErr != nil {
		return nil, m.messagesErr
	}
	return m.messages[channelID], nil
}

func (m *mockSlackAPI) GetThreadReplies(ctx context.Context, channelID string, threadTS string) ([]watcher.SlackMessage, error) {
	if m.threadErr != nil {
		return nil, m.threadErr
	}
	return m.threadReplies[threadTS], nil
}

type SlackWatcherSuite struct {
	suite.Suite
}

func TestSlackWatcher(t *testing.T) {
	suite.Run(t, new(SlackWatcherSuite))
}

// --- Constructor tests ---

func (s *SlackWatcherSuite) TestNewSlackWatcher_ValidConfig() {
	api := &mockSlackAPI{}
	cfg := config.SlackConfig{
		Enabled:             true,
		BotToken:            "xoxb-test",
		WorkspaceID:         "T12345",
		PollIntervalSeconds: 600,
	}

	w, err := watcher.NewSlackWatcher(api, cfg)
	s.NoError(err)
	s.NotNil(w)
}

func (s *SlackWatcherSuite) TestNewSlackWatcher_NilAPI() {
	cfg := config.SlackConfig{
		Enabled:             true,
		BotToken:            "xoxb-test",
		WorkspaceID:         "T12345",
		PollIntervalSeconds: 600,
	}

	w, err := watcher.NewSlackWatcher(nil, cfg)
	s.Error(err)
	s.Nil(w)
	s.Contains(err.Error(), "api")
}

func (s *SlackWatcherSuite) TestNewSlackWatcher_EmptyWorkspaceID() {
	api := &mockSlackAPI{}
	cfg := config.SlackConfig{
		Enabled:             true,
		BotToken:            "xoxb-test",
		WorkspaceID:         "",
		PollIntervalSeconds: 600,
	}

	w, err := watcher.NewSlackWatcher(api, cfg)
	s.Error(err)
	s.Nil(w)
	s.Contains(err.Error(), "workspace_id")
}

// --- Poll: basic message fetching ---

func (s *SlackWatcherSuite) TestPoll_FetchesMessagesFromAllChannels() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{
			{ID: "C001", Name: "general"},
			{ID: "C002", Name: "alerts"},
		},
		messages: map[string][]watcher.SlackMessage{
			"C001": {
				{ID: "msg1", ChannelID: "C001", ChannelName: "general", Sender: "U100", Text: "hello world", Timestamp: "1711500000.000100"},
			},
			"C002": {
				{ID: "msg2", ChannelID: "C002", ChannelName: "alerts", Sender: "U200", Text: "server down", Timestamp: "1711500001.000200"},
			},
		},
	}

	w := s.mustNewWatcher(api, "T12345")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Len(msgs, 2)

	// Verify messages are converted to repository.Message format
	sources := map[string]bool{}
	for _, msg := range msgs {
		s.Equal("slack", msg.Source)
		s.Equal("T12345", msg.SourceAccount)
		s.Equal("Pending", msg.Status)
		s.Equal("message", msg.MessageType)
		sources[msg.Channel] = true
	}
	s.True(sources["general"])
	s.True(sources["alerts"])
}

func (s *SlackWatcherSuite) TestPoll_MessageFieldsPopulated() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{
			{ID: "C001", Name: "general"},
		},
		messages: map[string][]watcher.SlackMessage{
			"C001": {
				{ID: "msg1", ChannelID: "C001", ChannelName: "general", Sender: "U100", Text: "important message", Timestamp: "1711500000.000100"},
			},
		},
	}

	w := s.mustNewWatcher(api, "T12345")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(msgs, 1)

	msg := msgs[0]
	s.Equal("slack", msg.Source)
	s.Equal("T12345", msg.SourceAccount)
	s.Equal("general", msg.Channel)
	s.Equal("U100", msg.Sender)
	s.Equal("msg1", msg.MessageID)
	s.Equal("important message", msg.RawContent)
	s.Equal("message", msg.MessageType)
	s.Equal("Pending", msg.Status)
	s.False(msg.ID.String() == "00000000-0000-0000-0000-000000000000")
	s.False(msg.CreatedAt.IsZero())
}

// --- Poll: new channel detection ---

func (s *SlackWatcherSuite) TestPoll_DetectsNewChannelJoin() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{
			{ID: "C001", Name: "general"},
		},
		messages: map[string][]watcher.SlackMessage{
			"C001": {},
		},
	}

	w := s.mustNewWatcher(api, "T12345")

	// First poll: general is new, should emit channel_join
	msgs, err := w.Poll(context.Background())
	s.NoError(err)

	joinMsgs := filterByType(msgs, "channel_join")
	s.Len(joinMsgs, 1)
	s.Equal("general", joinMsgs[0].Channel)
	s.Contains(joinMsgs[0].RawContent, "general")

	// Second poll with same channels: no new joins
	msgs, err = w.Poll(context.Background())
	s.NoError(err)

	joinMsgs = filterByType(msgs, "channel_join")
	s.Len(joinMsgs, 0)
}

func (s *SlackWatcherSuite) TestPoll_DetectsMultipleNewChannels() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{
			{ID: "C001", Name: "general"},
			{ID: "C002", Name: "alerts"},
			{ID: "C003", Name: "random"},
		},
		messages: map[string][]watcher.SlackMessage{
			"C001": {},
			"C002": {},
			"C003": {},
		},
	}

	w := s.mustNewWatcher(api, "T12345")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)

	joinMsgs := filterByType(msgs, "channel_join")
	s.Len(joinMsgs, 3)
}

func (s *SlackWatcherSuite) TestPoll_IncrementalChannelDetection() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{
			{ID: "C001", Name: "general"},
		},
		messages: map[string][]watcher.SlackMessage{
			"C001": {},
			"C002": {},
		},
	}

	w := s.mustNewWatcher(api, "T12345")

	// First poll: see general
	_, err := w.Poll(context.Background())
	s.NoError(err)

	// Add a new channel
	api.channels = append(api.channels, watcher.SlackChannel{ID: "C002", Name: "incidents"})

	// Second poll: only incidents is new
	msgs, err := w.Poll(context.Background())
	s.NoError(err)

	joinMsgs := filterByType(msgs, "channel_join")
	s.Len(joinMsgs, 1)
	s.Equal("incidents", joinMsgs[0].Channel)
}

// --- Poll: thread context ---

func (s *SlackWatcherSuite) TestPoll_IncludesThreadContext() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{
			{ID: "C001", Name: "general"},
		},
		messages: map[string][]watcher.SlackMessage{
			"C001": {
				{ID: "msg1", ChannelID: "C001", ChannelName: "general", Sender: "U100", Text: "thread reply", Timestamp: "1711500001.000200", ThreadTS: "1711500000.000100"},
			},
		},
		threadReplies: map[string][]watcher.SlackMessage{
			"1711500000.000100": {
				{ID: "parent1", ChannelID: "C001", ChannelName: "general", Sender: "U200", Text: "original message", Timestamp: "1711500000.000100"},
			},
		},
	}

	w := s.mustNewWatcher(api, "T12345")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)

	regularMsgs := filterByType(msgs, "message")
	s.Require().Len(regularMsgs, 1)
	// Thread reply should include parent context in raw content
	s.Contains(regularMsgs[0].RawContent, "original message")
	s.Contains(regularMsgs[0].RawContent, "thread reply")
}

// --- Poll: error handling ---

func (s *SlackWatcherSuite) TestPoll_ChannelListError_ReturnsError() {
	api := &mockSlackAPI{
		channelsErr: errors.New("slack api rate limited"),
	}

	w := s.mustNewWatcher(api, "T12345")
	msgs, err := w.Poll(context.Background())
	s.Error(err)
	s.Nil(msgs)
	s.Contains(err.Error(), "slack api rate limited")
}

func (s *SlackWatcherSuite) TestPoll_MessageFetchError_ContinuesOtherChannels() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{
			{ID: "C001", Name: "general"},
			{ID: "C002", Name: "alerts"},
		},
		messagesErr: errors.New("fetch failed"),
	}

	w := s.mustNewWatcher(api, "T12345")
	msgs, err := w.Poll(context.Background())
	// Should not return a hard error — channels were fetched, messages failed gracefully
	s.NoError(err)
	// Channel join events should still be emitted even if message fetch fails
	joinMsgs := filterByType(msgs, "channel_join")
	s.Len(joinMsgs, 2)
}

func (s *SlackWatcherSuite) TestPoll_ThreadFetchError_MessageStillIncluded() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{
			{ID: "C001", Name: "general"},
		},
		messages: map[string][]watcher.SlackMessage{
			"C001": {
				{ID: "msg1", ChannelID: "C001", ChannelName: "general", Sender: "U100", Text: "thread reply", Timestamp: "1711500001.000200", ThreadTS: "1711500000.000100"},
			},
		},
		threadErr: errors.New("thread fetch failed"),
	}

	w := s.mustNewWatcher(api, "T12345")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)

	// Message should still be included without thread context
	regularMsgs := filterByType(msgs, "message")
	s.Len(regularMsgs, 1)
	s.Equal("thread reply", regularMsgs[0].RawContent)
}

// --- Poll: context cancellation ---

func (s *SlackWatcherSuite) TestPoll_ContextCancelled_ReturnsError() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{
			{ID: "C001", Name: "general"},
		},
	}

	w := s.mustNewWatcher(api, "T12345")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := w.Poll(ctx)
	s.Error(err)
}

// --- Poll: no channels ---

func (s *SlackWatcherSuite) TestPoll_NoChannels_ReturnsEmpty() {
	api := &mockSlackAPI{
		channels: []watcher.SlackChannel{},
	}

	w := s.mustNewWatcher(api, "T12345")
	msgs, err := w.Poll(context.Background())
	s.NoError(err)
	s.Empty(msgs)
}

// --- Poll: tracks last poll timestamp per channel ---

func (s *SlackWatcherSuite) TestPoll_PassesLastTimestampToAPI() {
	callLog := &apiCallLog{}
	api := &trackingMockAPI{
		inner: &mockSlackAPI{
			channels: []watcher.SlackChannel{
				{ID: "C001", Name: "general"},
			},
			messages: map[string][]watcher.SlackMessage{
				"C001": {
					{ID: "msg1", ChannelID: "C001", ChannelName: "general", Sender: "U100", Text: "hello", Timestamp: "1711500005.000100"},
				},
			},
		},
		log: callLog,
	}

	w := s.mustNewWatcherWithAPI(api, "T12345")

	// First poll: oldest should be empty (fetch all)
	_, err := w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(callLog.getMessageCalls, 1)
	s.Equal("", callLog.getMessageCalls[0].oldest)

	// Second poll: oldest should be the timestamp of the last message seen
	_, err = w.Poll(context.Background())
	s.NoError(err)
	s.Require().Len(callLog.getMessageCalls, 2)
	s.Equal("1711500005.000100", callLog.getMessageCalls[1].oldest)
}

// --- Helpers ---

func (s *SlackWatcherSuite) mustNewWatcher(api watcher.SlackAPI, workspaceID string) *watcher.SlackWatcher {
	cfg := config.SlackConfig{
		Enabled:             true,
		BotToken:            "xoxb-test",
		WorkspaceID:         workspaceID,
		PollIntervalSeconds: 600,
	}
	w, err := watcher.NewSlackWatcher(api, cfg)
	s.Require().NoError(err)
	return w
}

func (s *SlackWatcherSuite) mustNewWatcherWithAPI(api watcher.SlackAPI, workspaceID string) *watcher.SlackWatcher {
	cfg := config.SlackConfig{
		Enabled:             true,
		BotToken:            "xoxb-test",
		WorkspaceID:         workspaceID,
		PollIntervalSeconds: 600,
	}
	w, err := watcher.NewSlackWatcher(api, cfg)
	s.Require().NoError(err)
	return w
}

func filterByType(msgs []*repository.Message, msgType string) []*repository.Message {
	var result []*repository.Message
	for _, m := range msgs {
		if m.MessageType == msgType {
			result = append(result, m)
		}
	}
	return result
}

// trackingMockAPI wraps a SlackAPI and logs calls for verification.
type trackingMockAPI struct {
	inner watcher.SlackAPI
	log   *apiCallLog
}

type apiCallLog struct {
	getMessageCalls []getMessageCall
}

type getMessageCall struct {
	channelID string
	oldest    string
}

func (t *trackingMockAPI) GetUserChannels(ctx context.Context) ([]watcher.SlackChannel, error) {
	return t.inner.GetUserChannels(ctx)
}

func (t *trackingMockAPI) GetChannelMessages(ctx context.Context, channelID string, oldest string) ([]watcher.SlackMessage, error) {
	t.log.getMessageCalls = append(t.log.getMessageCalls, getMessageCall{channelID: channelID, oldest: oldest})
	return t.inner.GetChannelMessages(ctx, channelID, oldest)
}

func (t *trackingMockAPI) GetThreadReplies(ctx context.Context, channelID string, threadTS string) ([]watcher.SlackMessage, error) {
	return t.inner.GetThreadReplies(ctx, channelID, threadTS)
}
