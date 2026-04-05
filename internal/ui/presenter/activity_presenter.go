package presenter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ActivityPresenter struct {
	source     ActivitySource
	maxEntries int

	mu       sync.Mutex
	entries  []ActivityEntry
	onUpdate func()

	cancel context.CancelFunc
}

func NewActivityPresenter(source ActivitySource, maxEntries int) (*ActivityPresenter, error) {
	if source == nil {
		return nil, fmt.Errorf("source must not be nil")
	}
	if maxEntries <= 0 {
		return nil, fmt.Errorf("maxEntries must be greater than zero")
	}
	return &ActivityPresenter{
		source:     source,
		maxEntries: maxEntries,
	}, nil
}

func (p *ActivityPresenter) Start(ctx context.Context) {
	ctx, p.cancel = context.WithCancel(ctx)
	go p.run(ctx)
}

func (p *ActivityPresenter) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *ActivityPresenter) SetOnUpdate(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onUpdate = fn
}

func (p *ActivityPresenter) Entries() []ActivityEntry {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.entries)
	result := make([]ActivityEntry, n)
	for i := range n {
		result[n-1-i] = p.entries[i]
	}
	return result
}

func (p *ActivityPresenter) run(ctx context.Context) {
	ch := p.source.Events()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			entry := ActivityEntry{
				Source:    ev.Source,
				Message:   ev.Message,
				IsError:   ev.IsError,
				Timestamp: time.Now(),
			}

			p.mu.Lock()
			p.entries = append(p.entries, entry)
			if len(p.entries) > p.maxEntries {
				p.entries = p.entries[len(p.entries)-p.maxEntries:]
			}
			cb := p.onUpdate
			p.mu.Unlock()

			if cb != nil {
				cb()
			}
		}
	}
}
