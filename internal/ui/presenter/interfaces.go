package presenter

import (
	"context"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

type MessageQuerier interface {
	QueryByStatus(ctx context.Context, status string) ([]*repository.Message, error)
}

type MessageUpdater interface {
	Update(ctx context.Context, msg *repository.Message) error
}

type ActivityEvent struct {
	Source  string
	Message string
	IsError bool
}

type ActivityEntry struct {
	Source    string
	Message   string
	IsError   bool
	Timestamp time.Time
}

type ActivitySource interface {
	Events() <-chan ActivityEvent
}
