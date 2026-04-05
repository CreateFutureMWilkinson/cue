package presenter

import (
	"context"
	"time"

	"github.com/google/uuid"

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

// VolumeController abstracts audio volume control.
type VolumeController interface {
	SetVolume(volume int)
}

type BufferReviewer interface {
	GetBufferedMessages(ctx context.Context) ([]*repository.Message, error)
	CountBuffered(ctx context.Context) (int, error)
	SaveRating(ctx context.Context, messageID uuid.UUID, rating int, feedback *string) error
	DeleteMessage(ctx context.Context, messageID uuid.UUID) error
}
