package presenter

import (
	"context"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

type MessageQuerier interface {
	QueryByStatus(ctx context.Context, status string) ([]*repository.Message, error)
}

type MessageUpdater interface {
	Update(ctx context.Context, msg *repository.Message) error
}
