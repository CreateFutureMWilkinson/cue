package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
)

// QueueProcessor processes Ollama scoring queue entries one at a time.
type QueueProcessor struct {
	queueRepo           repository.QueueRepository
	msgRepo             repository.MessageRepository
	scorer              decisionengine.Scorer
	fewShot             decisionengine.FewShotProvider
	alerter             Alerter
	eventCh             chan<- ActivityEvent
	importanceThreshold float64
	confidenceThreshold float64
	cooldown            time.Duration
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
}

// NewQueueProcessor creates a new QueueProcessor, validating all required dependencies.
func NewQueueProcessor(
	queueRepo repository.QueueRepository,
	msgRepo repository.MessageRepository,
	scorer decisionengine.Scorer,
	alerter Alerter,
	eventCh chan<- ActivityEvent,
	importanceThreshold float64,
	confidenceThreshold float64,
	cooldown time.Duration,
) (*QueueProcessor, error) {
	if queueRepo == nil {
		return nil, errors.New("queueRepo is required")
	}
	if msgRepo == nil {
		return nil, errors.New("msgRepo is required")
	}
	if scorer == nil {
		return nil, errors.New("scorer is required")
	}
	if alerter == nil {
		return nil, errors.New("alerter is required")
	}

	return &QueueProcessor{
		queueRepo:           queueRepo,
		msgRepo:             msgRepo,
		scorer:              scorer,
		alerter:             alerter,
		eventCh:             eventCh,
		importanceThreshold: importanceThreshold,
		confidenceThreshold: confidenceThreshold,
		cooldown:            cooldown,
	}, nil
}

// ProcessOne dequeues and processes a single queue entry.
// Returns (true, nil) if an entry was processed, (false, nil) if the queue is empty.
func (p *QueueProcessor) ProcessOne(ctx context.Context) (bool, error) {
	entry, err := p.queueRepo.DequeueOldest(ctx)
	if err != nil {
		return false, fmt.Errorf("dequeue: %w", err)
	}
	if entry == nil {
		return false, nil
	}

	msg, err := p.msgRepo.QueryByID(ctx, entry.MessageID)
	if err != nil {
		_ = p.queueRepo.MarkFailed(ctx, entry.ID)
		return false, fmt.Errorf("query message %s: %w", entry.MessageID, err)
	}

	result, scorerErr := p.scorer.ScoreWithContext(ctx, msg, nil)
	if scorerErr != nil {
		msg.ImportanceScore = 7
		msg.ConfidenceScore = 0
		msg.Status = "Buffered"
		msg.Reasoning = "Ollama scoring failed: " + scorerErr.Error()
		_ = p.msgRepo.Update(ctx, msg)
		_ = p.queueRepo.MarkFailed(ctx, entry.ID)
		p.emitEvent(msg.Source, fmt.Sprintf("scoring failed for %s: %v", entry.MessageID, scorerErr))
		return true, nil
	}

	msg.ImportanceScore = result.ImportanceScore
	msg.ConfidenceScore = result.ConfidenceScore
	msg.Reasoning = result.Reasoning
	msg.Status = p.determineStatus(result.ImportanceScore, result.ConfidenceScore)

	if err := p.msgRepo.Update(ctx, msg); err != nil {
		_ = p.queueRepo.MarkFailed(ctx, entry.ID)
		return false, fmt.Errorf("update message %s: %w", entry.MessageID, err)
	}

	if err := p.queueRepo.MarkDone(ctx, entry.ID); err != nil {
		return false, fmt.Errorf("mark done %s: %w", entry.ID, err)
	}

	if msg.Status == "Notified" {
		_ = p.alerter.PlayNotification(ctx)
	}

	p.emitEvent(msg.Source, fmt.Sprintf("scored %s → %s (IS=%.1f, CS=%.2f)", entry.MessageID, msg.Status, msg.ImportanceScore, msg.ConfidenceScore))

	return true, nil
}

// determineStatus returns the routing status based on importance and confidence thresholds.
func (p *QueueProcessor) determineStatus(importance, confidence float64) string {
	if importance >= p.importanceThreshold && confidence >= p.confidenceThreshold {
		return "Notified"
	}
	if importance >= p.importanceThreshold {
		return "Buffered"
	}
	return "Ignored"
}

// Start launches a background goroutine that continuously processes queue entries.
func (p *QueueProcessor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			processed, err := p.ProcessOne(ctx)
			if err != nil {
				p.emitEvent("queue", fmt.Sprintf("process error: %v", err))
			}
			_ = processed

			select {
			case <-ctx.Done():
				return
			case <-time.After(p.cooldown):
			}
		}
	}()
}

// Stop signals the background goroutine to stop and waits for it to finish.
func (p *QueueProcessor) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

// SetFewShotProvider sets the few-shot provider for calibration. Optional; nil = no calibration.
func (p *QueueProcessor) SetFewShotProvider(fsp decisionengine.FewShotProvider) {
	p.fewShot = fsp
}

// emitEvent sends an activity event if the event channel is configured.
func (p *QueueProcessor) emitEvent(source, message string) {
	if p.eventCh != nil {
		select {
		case p.eventCh <- ActivityEvent{Source: source, Message: message}:
		default:
		}
	}
}
