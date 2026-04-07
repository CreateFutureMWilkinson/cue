package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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

// RunCleanup runs cleanup functions sequentially with a timeout. If the total
// elapsed time exceeds timeout, it returns immediately with a timeout error.
// Otherwise it returns the first error from any cleanup function.
func RunCleanup(timeout time.Duration, fns ...func() error) error {
	done := make(chan error, 1)
	go func() {
		var firstErr error
		for _, fn := range fns {
			if err := fn(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		done <- firstErr
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("shutdown cleanup timeout after %s", timeout)
	}
}
