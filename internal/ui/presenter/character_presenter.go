package presenter

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
)

const (
	notifiedKeyword = "NOTIFIED"
)

// CharacterPresenter maps activity and alert events to character state
// transitions.
type CharacterPresenter struct {
	char          character.Character
	source        ActivitySource
	alertSource   AlertSource
	decayDuration time.Duration

	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu         sync.Mutex
	decayTimer *time.Timer
}

// NewCharacterPresenter creates a new CharacterPresenter. alertSource may
// be nil during transitional wiring; alert handling is a no-op when nil.
func NewCharacterPresenter(char character.Character, source ActivitySource, alertSource AlertSource, decayDuration time.Duration) (*CharacterPresenter, error) {
	return &CharacterPresenter{
		char:          char,
		source:        source,
		alertSource:   alertSource,
		decayDuration: decayDuration,
	}, nil
}

// Start begins consuming events and mapping them to character states.
func (p *CharacterPresenter) Start(ctx context.Context) {
	ctx, p.cancel = context.WithCancel(ctx)

	var alertCh <-chan AlertEvent
	if p.alertSource != nil {
		alertCh = p.alertSource.Events()
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-alertCh:
				p.char.TransitionTo(character.StateNotifying)
				p.resetDecayTimer()
			case event := <-p.source.Events():
				state := p.mapEventToState(event)
				p.char.TransitionTo(state)
				p.resetDecayTimer()
			}
		}
	}()
}

// Stop cancels the event loop and waits for it to finish.
func (p *CharacterPresenter) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	p.mu.Lock()
	if p.decayTimer != nil {
		p.decayTimer.Stop()
	}
	p.mu.Unlock()
}

func (p *CharacterPresenter) mapEventToState(event ActivityEvent) character.CharacterState {
	if event.IsError {
		return character.StateError
	}
	if strings.Contains(event.Message, notifiedKeyword) {
		return character.StateNotifying
	}
	return character.StateWorking
}

func (p *CharacterPresenter) resetDecayTimer() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.decayTimer != nil {
		p.decayTimer.Stop()
	}

	p.decayTimer = time.AfterFunc(p.decayDuration, func() {
		p.char.TransitionTo(character.StateIdle)
	})
}
