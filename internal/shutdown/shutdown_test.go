package shutdown_test

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/shutdown"
	"github.com/stretchr/testify/suite"
)

type ShutdownSuite struct {
	suite.Suite
}

func TestShutdown(t *testing.T) {
	suite.Run(t, new(ShutdownSuite))
}

func (s *ShutdownSuite) TestCallsQuitOnInterrupt() {
	// Prevent SIGINT from killing the test process by registering our own
	// handler. This ensures the signal is captured even if the stub does
	// not register its own signal.Notify call.
	guard := make(chan os.Signal, 1)
	signal.Notify(guard, os.Interrupt)
	defer signal.Stop(guard)

	var called atomic.Bool

	quitFn := func() {
		called.Store(true)
	}

	handler := shutdown.NewSignalHandler(quitFn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler.Start(ctx)

	// Give the goroutine a moment to register its signal listener (when
	// implemented). With the noop stub this is irrelevant but harmless.
	time.Sleep(50 * time.Millisecond)

	// Send SIGINT to our own process.
	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	s.Require().NoError(err)

	// Drain the guard channel so it does not block.
	select {
	case <-guard:
	case <-time.After(1 * time.Second):
	}

	// Wait up to 1 second for quitFn to be called.
	deadline := time.After(1 * time.Second)
	for {
		if called.Load() {
			break
		}
		select {
		case <-deadline:
			s.Fail("quitFn was not called within 1 second after sending SIGINT")
			return
		case <-time.After(10 * time.Millisecond):
			// poll again
		}
	}

	s.True(called.Load(), "expected quitFn to have been called")
}

func (s *ShutdownSuite) TestRunCleanupCompletesWithinTimeout() {
	fastFn := func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	start := time.Now()
	err := shutdown.RunCleanup(1*time.Second, fastFn, fastFn)
	elapsed := time.Since(start)

	s.NoError(err, "expected no error when cleanup fns complete within timeout")
	s.Less(elapsed, 100*time.Millisecond, "expected cleanup to complete well within timeout")
}

func (s *ShutdownSuite) TestRunCleanupReturnsTimeoutError() {
	slowFn := func() error {
		time.Sleep(2 * time.Second)
		return nil
	}

	start := time.Now()
	err := shutdown.RunCleanup(100*time.Millisecond, slowFn)
	elapsed := time.Since(start)

	s.Require().Error(err, "expected a timeout error when cleanup fn exceeds timeout")
	s.True(errors.Is(err, context.DeadlineExceeded) || containsTimeout(err),
		"expected error to indicate timeout, got: %v", err)
	s.Less(elapsed, 200*time.Millisecond, "expected RunCleanup to return promptly after timeout")
}

// containsTimeout checks if the error message contains "timeout".
func containsTimeout(err error) bool {
	if err == nil {
		return false
	}
	return len(err.Error()) > 0 && contains(err.Error(), "timeout")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
