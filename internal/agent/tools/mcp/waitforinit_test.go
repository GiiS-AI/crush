package mcp

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func swapInitGate(t *testing.T) chan struct{} {
	t.Helper()

	origDone := initDone
	origOnce := initOnce
	initDone = make(chan struct{})
	initOnce = sync.Once{}

	initMu.Lock()
	origStarted := initStarted
	origArmedAt := initArmedAt
	initStarted = true
	initArmedAt = time.Now()
	initMu.Unlock()

	t.Cleanup(func() {
		initDone = origDone
		initOnce = origOnce
		initMu.Lock()
		initStarted = origStarted
		initArmedAt = origArmedAt
		initMu.Unlock()
	})

	return initDone
}

func TestWaitForInit_BlocksUntilInitCompletes(t *testing.T) {
	gate := swapInitGate(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, WaitForInit(ctx), context.DeadlineExceeded)

	close(gate)
	require.NoError(t, WaitForInit(context.Background()))
}

func TestWaitForInit_ReturnsWhenNotArmed(t *testing.T) {
	initMu.Lock()
	origStarted := initStarted
	initStarted = false
	initMu.Unlock()
	t.Cleanup(func() {
		initMu.Lock()
		initStarted = origStarted
		initMu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, WaitForInit(ctx))
}

func TestWaitForInitBudget_ProceedsWhenInitWedged(t *testing.T) {
	swapInitGate(t)

	start := time.Now()
	require.NoError(t, WaitForInitBudget(context.Background(), 50*time.Millisecond))
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestWaitForInitBudget_ReturnsOnceInitCompletes(t *testing.T) {
	gate := swapInitGate(t)

	go func() {
		time.Sleep(20 * time.Millisecond)
		close(gate)
	}()

	start := time.Now()
	require.NoError(t, WaitForInitBudget(context.Background(), 30*time.Second))
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestWaitForInitBudget_CallerCancellationStillAborts(t *testing.T) {
	swapInitGate(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	require.ErrorIs(t, WaitForInitBudget(ctx, 30*time.Second), context.Canceled)
}

func TestWaitForInitBudget_DeadlineIsAbsolute(t *testing.T) {
	swapInitGate(t)

	initMu.Lock()
	initArmedAt = time.Now().Add(-time.Minute)
	initMu.Unlock()

	start := time.Now()
	require.NoError(t, WaitForInitBudget(context.Background(), 30*time.Second))
	require.Less(t, time.Since(start), 5*time.Second)
}
