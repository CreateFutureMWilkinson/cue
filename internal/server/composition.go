package server

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
	"github.com/CreateFutureMWilkinson/cue/internal/secret"
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
func NewComposition(_ context.Context, cfg config.Config) (*Composition, error) {
	msgRepo, err := sqlite.NewSQLiteMessageRepository(cfg.Database.Path, cfg.Orchestrator.Router.BufferSizePerSource)
	if err != nil {
		return nil, fmt.Errorf("opening message repository: %w", err)
	}

	queueRepo, err := sqlite.NewSQLiteQueueRepository(msgRepo.DB())
	if err != nil {
		return nil, fmt.Errorf("opening queue repository: %w", err)
	}

	ruleRepo, err := sqlite.NewSQLiteRoutingRuleRepository(msgRepo.DB())
	if err != nil {
		return nil, fmt.Errorf("opening routing rule repository: %w", err)
	}

	keyPath := filepath.Join(filepath.Dir(cfg.Database.Path), "secret.key")
	enc, err := secret.NewKeyFileEncryptor(keyPath)
	if err != nil {
		return nil, fmt.Errorf("opening encryptor: %w", err)
	}

	serviceConfigRepo, err := sqlite.NewSQLiteServiceConfigRepository(msgRepo.DB(), enc)
	if err != nil {
		return nil, fmt.Errorf("opening service config repository: %w", err)
	}

	return &Composition{
		MessageRepo:       msgRepo,
		QueueRepo:         queueRepo,
		RuleRepo:          ruleRepo,
		ServiceConfigRepo: serviceConfigRepo,
	}, nil
}

// Shutdown performs an ordered shutdown of all composition components.
func (c *Composition) Shutdown(ctx context.Context) error {
	return ErrCompositionNotImplemented
}
