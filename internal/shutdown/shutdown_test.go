package shutdown_test

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/shutdown"
	"github.com/stretchr/testify/suite"
)

type SignalHandlerSuite struct {
	suite.Suite
}

func TestSignalHandler(t *testing.T) {
	suite.Run(t, new(SignalHandlerSuite))
}

func (s *SignalHandlerSuite) TestCallsQuitOnInterrupt() {
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
