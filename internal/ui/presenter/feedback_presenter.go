package presenter

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// FeedbackItem represents a buffered message for user review.
type FeedbackItem struct {
	ID              uuid.UUID
	Source          string
	Sender          string
	Channel         string
	Content         string
	ImportanceScore float64
	ConfidenceScore float64
}

// FeedbackPresenter manages the feedback buffer review workflow.
type FeedbackPresenter struct {
	reviewer BufferReviewer
	items    []*FeedbackItem
	index    int
}

// NewFeedbackPresenter creates a new FeedbackPresenter with the given BufferReviewer.
func NewFeedbackPresenter(reviewer BufferReviewer) (*FeedbackPresenter, error) {
	if reviewer == nil {
		return nil, fmt.Errorf("reviewer must not be nil")
	}
	return &FeedbackPresenter{
		reviewer: reviewer,
	}, nil
}

// Load fetches buffered messages from the reviewer and resets the position.
func (p *FeedbackPresenter) Load(ctx context.Context) error {
	messages, err := p.reviewer.GetBufferedMessages(ctx)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}

	p.items = make([]*FeedbackItem, len(messages))
	for i, msg := range messages {
		p.items[i] = messageToFeedbackItem(msg)
	}
	p.index = 0
	return nil
}

// Current returns the current FeedbackItem, or nil if there is none.
func (p *FeedbackPresenter) Current() *FeedbackItem {
	if !p.HasCurrent() {
		return nil
	}
	return p.items[p.index]
}

// Counter returns a string in the format "X of Y" (1-indexed).
func (p *FeedbackPresenter) Counter() string {
	return fmt.Sprintf("%d of %d", p.index+1, len(p.items))
}

// HasCurrent returns true if there is a current item to review.
func (p *FeedbackPresenter) HasCurrent() bool {
	return p.index < len(p.items)
}

// SaveRating saves the rating for the current message and advances to the next.
func (p *FeedbackPresenter) SaveRating(ctx context.Context, rating int, feedback *string) error {
	if !p.HasCurrent() {
		return fmt.Errorf("save rating: no current message")
	}

	current := p.items[p.index]
	if err := p.reviewer.SaveRating(ctx, current.ID, rating, feedback); err != nil {
		return fmt.Errorf("save rating: %w", err)
	}

	p.index++
	return nil
}

// Skip advances to the next message without saving a rating.
func (p *FeedbackPresenter) Skip() {
	if p.HasCurrent() {
		p.index++
	}
}

// Delete removes the current message from the list and the reviewer.
func (p *FeedbackPresenter) Delete(ctx context.Context) error {
	if !p.HasCurrent() {
		return fmt.Errorf("delete: no current message")
	}

	current := p.items[p.index]
	if err := p.reviewer.DeleteMessage(ctx, current.ID); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	// Remove from slice; next item slides into current index.
	p.items = slices.Delete(p.items, p.index, p.index+1)
	return nil
}

func messageToFeedbackItem(msg *repository.Message) *FeedbackItem {
	return &FeedbackItem{
		ID:              msg.ID,
		Source:          msg.Source,
		Sender:          msg.Sender,
		Channel:         msg.Channel,
		Content:         msg.RawContent,
		ImportanceScore: msg.ImportanceScore,
		ConfidenceScore: msg.ConfidenceScore,
	}
}
