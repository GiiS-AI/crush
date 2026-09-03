package herdr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// recordingSender captures state transitions without connecting to a real
// Unix socket.
type recordingSender struct {
	states []string
}

func (r *recordingSender) send(req reportRequest) error {
	r.states = append(r.states, req.Params.State)
	return nil
}

func (r *recordingSender) close() {}

// newTestClient creates a Client that records state transitions without
// connecting to a real Unix socket.
func newTestClient() *Client {
	rec := &recordingSender{states: make([]string, 0, 16)}
	return &Client{
		state: stateIdle,
		snd:   rec,
	}
}

// reportedStates returns the states recorded by the test sender.
func reportedStates(c *Client) []string {
	return c.snd.(*recordingSender).states
}

func TestBasicLifecycle(t *testing.T) {
	t.Parallel()
	c := newTestClient()

	c.HandleEvent(AssistantMessage{SessionID: "sess-1"})
	assert.Equal(t, []string{stateWorking}, reportedStates(c))

	c.HandleEvent(RunComplete{SessionID: "sess-1"})
	assert.Equal(t, []string{stateWorking, stateIdle}, reportedStates(c))
}

func TestPermissionBlockAndUnblock(t *testing.T) {
	t.Parallel()
	c := newTestClient()

	c.HandleEvent(AssistantMessage{SessionID: "sess-1"})
	c.HandleEvent(PermissionRequested{})
	assert.Equal(t, []string{stateWorking, stateBlocked}, reportedStates(c))

	c.HandleEvent(PermissionResolved{})
	assert.Equal(t, []string{stateWorking, stateBlocked, stateWorking}, reportedStates(c))

	c.HandleEvent(RunComplete{SessionID: "sess-1"})
	assert.Equal(t, []string{stateWorking, stateBlocked, stateWorking, stateIdle}, reportedStates(c))
}

func TestPermissionBeforeAssistantMessage(t *testing.T) {
	t.Parallel()
	c := newTestClient()

	c.HandleEvent(PermissionRequested{})
	assert.Equal(t, []string{stateBlocked}, reportedStates(c))

	c.HandleEvent(PermissionResolved{})
	assert.Equal(t, []string{stateBlocked, stateWorking}, reportedStates(c))
}

func TestSessionIDPropagation(t *testing.T) {
	t.Parallel()
	c := newTestClient()

	c.SetSessionID("early-session")
	assert.Equal(t, "early-session", c.sessionID)

	c.HandleEvent(RunComplete{SessionID: "final-session"})
	assert.Equal(t, "final-session", c.sessionID)
}

func TestDedupSkipsRedundantState(t *testing.T) {
	t.Parallel()
	c := newTestClient()

	c.HandleEvent(AssistantMessage{SessionID: "s1"})
	c.HandleEvent(AssistantMessage{SessionID: "s1"})
	assert.Equal(t, []string{stateWorking}, reportedStates(c))
}

func TestSummarizingTriggersWorking(t *testing.T) {
	t.Parallel()
	c := newTestClient()

	c.HandleEvent(Summarizing{})
	assert.Equal(t, []string{stateWorking}, reportedStates(c))

	c.HandleEvent(Summarizing{})
	assert.Equal(t, []string{stateWorking}, reportedStates(c))
}

func TestNilClientSafe(t *testing.T) {
	t.Parallel()
	var c *Client
	c.SetSessionID("s1")
	c.HandleEvent(AssistantMessage{SessionID: "s1"})
	c.HandleEvent(RunComplete{SessionID: "s1"})
	c.HandleEvent(PermissionRequested{})
	c.HandleEvent(PermissionResolved{})
	c.HandleEvent(Summarizing{})
}

func TestRegisterInitial(t *testing.T) {
	t.Parallel()
	rec := &recordingSender{states: make([]string, 0, 16)}
	c := &Client{
		state: stateIdle,
		seq:   100,
		snd:   rec,
	}
	c.registerInitial()
	assert.Equal(t, []string{stateIdle}, rec.states)
	assert.Equal(t, uint64(101), c.seq)
}

// TestInitDisabledUnderTest guards the critical safety property that herdr
// never attaches to a real pane from a test binary. Test processes inherit
// the developer's HERDR_* environment, so a missing guard would release the
// live pane's agent on teardown.
func TestInitDisabledUnderTest(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	t.Setenv("HERDR_SOCKET_PATH", "/tmp/does-not-matter.sock")
	t.Setenv("HERDR_PANE_ID", "test:pane")
	assert.Nil(t, newFromEnv())
}

func TestInitDisabledOnUnsupportedPlatform(t *testing.T) {
	prev := platformSupported
	platformSupported = false
	t.Cleanup(func() { platformSupported = prev })

	t.Setenv("HERDR_ENV", "1")
	t.Setenv("HERDR_SOCKET_PATH", "/tmp/does-not-matter.sock")
	t.Setenv("HERDR_PANE_ID", "test:pane")

	assert.Nil(t, newFromEnv())
}
