package server

import (
	"context"
	"errors"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// ErrCompositionNotImplemented is returned by stub methods that are not yet implemented.
var ErrCompositionNotImplemented = errors.New("not implemented")

// Composition holds all long-lived components wired into cue-server.
type Composition struct {
	MessageRepo       repository.MessageRepository
	QueueRepo         repository.QueueRepository
	RuleRepo          repository.RoutingRuleRepository
	ServiceConfigRepo repository.ServiceConfigRepository
}

// NewComposition opens all repositories, constructs services, wires the
// orchestrator, and returns a ready-to-use Composition.
func NewComposition(ctx context.Context, cfg config.Config) (*Composition, error) {
	return nil, ErrCompositionNotImplemented
}

// Shutdown performs an ordered shutdown of all composition components.
func (c *Composition) Shutdown(ctx context.Context) error {
	return ErrCompositionNotImplemented
}
