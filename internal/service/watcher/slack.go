package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
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
func (w *SlackWatcher) Poll(ctx context.Context) ([]*repository.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	channels, err := w.api.GetUserChannels(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting user channels: %w", err)
	}

	var result []*repository.Message

	for _, ch := range channels {
		isNew := !w.knownChannels[ch.ID]
		if isNew {
			w.knownChannels[ch.ID] = true
		}

		messages, msgErr := w.api.GetChannelMessages(ctx, ch.ID, w.lastTimestamp[ch.ID])
		if msgErr != nil {
			if isNew {
				result = append(result, w.newChannelJoinMessage(ch))
			}
			continue
		}

		if isNew && len(messages) == 0 {
			result = append(result, w.newChannelJoinMessage(ch))
		}

		for _, sm := range messages {
			rawContent := sm.Text

			if sm.ThreadTS != "" {
				replies, threadErr := w.api.GetThreadReplies(ctx, sm.ChannelID, sm.ThreadTS)
				if threadErr == nil && len(replies) > 0 {
					rawContent = replies[0].Text + "\n" + sm.Text
				}
			}

			msg := &repository.Message{
				ID:            uuid.New(),
				Source:        "slack",
				SourceAccount: w.workspaceID,
				Channel:       sm.ChannelName,
				Sender:        sm.Sender,
				MessageID:     sm.ID,
				MessageType:   "message",
				RawContent:    rawContent,
				Status:        "Pending",
				CreatedAt:     time.Now(),
			}
			result = append(result, msg)

			if sm.Timestamp > w.lastTimestamp[ch.ID] {
				w.lastTimestamp[ch.ID] = sm.Timestamp
			}
		}
	}

	return result, nil
}

func (w *SlackWatcher) newChannelJoinMessage(ch SlackChannel) *repository.Message {
	return &repository.Message{
		ID:            uuid.New(),
		Source:        "slack",
		SourceAccount: w.workspaceID,
		Channel:       ch.Name,
		MessageID:     ch.ID,
		MessageType:   "channel_join",
		RawContent:    fmt.Sprintf("Joined channel %s", ch.Name),
		Status:        "Pending",
		CreatedAt:     time.Now(),
	}
}
