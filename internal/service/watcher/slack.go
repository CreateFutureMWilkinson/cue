package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// Constants for Slack message types and status values
const (
	SourceSlack     = "slack"
	StatusPending   = "Pending"
	MessageTypeMsg  = "message"
	MessageTypeJoin = "channel_join"
	ThreadSeparator = "\n"
)

// SlackChannel represents a Slack channel.
type SlackChannel struct {
	ID   string
	Name string
}

// SlackMessage represents a message from Slack.
type SlackMessage struct {
	ID          string
	ChannelID   string
	ChannelName string
	Sender      string
	Text        string
	Timestamp   string
	ThreadTS    string
}

// SlackAPI defines the interface for interacting with the Slack API.
type SlackAPI interface {
	GetUserChannels(ctx context.Context) ([]SlackChannel, error)
	GetChannelMessages(ctx context.Context, channelID string, oldest string) ([]SlackMessage, error)
	GetThreadReplies(ctx context.Context, channelID string, threadTS string) ([]SlackMessage, error)
}

// SlackWatcher polls Slack for new messages and converts them to repository messages.
type SlackWatcher struct {
	api           SlackAPI
	workspaceID   string
	knownChannels map[string]bool
	lastTimestamp map[string]string
}

// NewSlackWatcher creates a new SlackWatcher with the given API client and configuration.
// Returns an error if api is nil or if the workspace ID is empty.
func NewSlackWatcher(api SlackAPI, cfg config.SlackConfig) (*SlackWatcher, error) {
	if api == nil {
		return nil, fmt.Errorf("api must not be nil")
	}
	if cfg.WorkspaceID == "" {
		return nil, fmt.Errorf("workspace_id must not be empty")
	}
	return &SlackWatcher{
		api:           api,
		workspaceID:   cfg.WorkspaceID,
		knownChannels: make(map[string]bool),
		lastTimestamp: make(map[string]string),
	}, nil
}

// Poll fetches new messages from all Slack channels and returns them as repository messages.
// It detects new channel joins and includes thread context for threaded messages.
func (w *SlackWatcher) Poll(ctx context.Context) ([]*repository.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	channels, err := w.api.GetUserChannels(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting user channels: %w", err)
	}

	var result []*repository.Message

	for _, channel := range channels {
		channelMessages := w.processChannel(ctx, channel)
		result = append(result, channelMessages...)
	}

	return result, nil
}

// processChannel handles message fetching and processing for a single channel
func (w *SlackWatcher) processChannel(ctx context.Context, channel SlackChannel) []*repository.Message {
	var result []*repository.Message

	isNewChannel := !w.knownChannels[channel.ID]
	if isNewChannel {
		w.knownChannels[channel.ID] = true
	}

	messages, err := w.api.GetChannelMessages(ctx, channel.ID, w.lastTimestamp[channel.ID])
	if err != nil {
		// If we can't fetch messages but this is a new channel, still emit the join event
		if isNewChannel {
			result = append(result, w.createChannelJoinMessage(channel))
		}
		return result
	}

	// Emit channel join event for new channels with no messages
	if isNewChannel && len(messages) == 0 {
		result = append(result, w.createChannelJoinMessage(channel))
	}

	// Process each message
	for _, slackMsg := range messages {
		repoMsg := w.convertSlackMessage(ctx, slackMsg)
		result = append(result, repoMsg)

		// Update last seen timestamp for this channel
		w.updateLastTimestamp(channel.ID, slackMsg.Timestamp)
	}

	return result
}

// convertSlackMessage converts a SlackMessage to a repository.Message with thread context
func (w *SlackWatcher) convertSlackMessage(ctx context.Context, slackMsg SlackMessage) *repository.Message {
	content := w.buildMessageContent(ctx, slackMsg)

	return &repository.Message{
		ID:            uuid.New(),
		Source:        SourceSlack,
		SourceAccount: w.workspaceID,
		Channel:       slackMsg.ChannelName,
		Sender:        slackMsg.Sender,
		MessageID:     slackMsg.ID,
		MessageType:   MessageTypeMsg,
		RawContent:    content,
		Status:        StatusPending,
		CreatedAt:     time.Now(),
	}
}

// buildMessageContent constructs the message content, including thread context if available
func (w *SlackWatcher) buildMessageContent(ctx context.Context, slackMsg SlackMessage) string {
	content := slackMsg.Text

	if slackMsg.ThreadTS != "" {
		if threadContext := w.getThreadContext(ctx, slackMsg); threadContext != "" {
			content = threadContext + ThreadSeparator + content
		}
	}

	return content
}

// getThreadContext retrieves the parent message content for thread replies
func (w *SlackWatcher) getThreadContext(ctx context.Context, slackMsg SlackMessage) string {
	replies, err := w.api.GetThreadReplies(ctx, slackMsg.ChannelID, slackMsg.ThreadTS)
	if err != nil || len(replies) == 0 {
		return ""
	}
	return replies[0].Text
}

// updateLastTimestamp updates the last seen timestamp for a channel if the new timestamp is newer
func (w *SlackWatcher) updateLastTimestamp(channelID, timestamp string) {
	if timestamp > w.lastTimestamp[channelID] {
		w.lastTimestamp[channelID] = timestamp
	}
}

// createChannelJoinMessage creates a repository message for channel join events
func (w *SlackWatcher) createChannelJoinMessage(channel SlackChannel) *repository.Message {
	return &repository.Message{
		ID:            uuid.New(),
		Source:        SourceSlack,
		SourceAccount: w.workspaceID,
		Channel:       channel.Name,
		MessageID:     channel.ID,
		MessageType:   MessageTypeJoin,
		RawContent:    fmt.Sprintf("Joined channel %s", channel.Name),
		Status:        StatusPending,
		CreatedAt:     time.Now(),
	}
}
