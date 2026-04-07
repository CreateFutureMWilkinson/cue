package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// SignalHandler listens for OS interrupt and SIGTERM signals and calls a quit
// function when one is received. It guarantees quitFn is called at most once.
type SignalHandler struct {
	quitFn func()
	once   sync.Once
}

// NewSignalHandler creates a SignalHandler that will call quitFn when an
// os.Interrupt or syscall.SIGTERM signal is received.
func NewSignalHandler(quitFn func()) *SignalHandler {
	return &SignalHandler{quitFn: quitFn}
}

// Start begins listening for signals in a background goroutine. When a signal
// arrives it calls quitFn. When ctx is cancelled it stops listening.
func (h *SignalHandler) Start(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			h.once.Do(h.quitFn)
		case <-ctx.Done():
			signal.Stop(sigCh)
		}
	}()
}
