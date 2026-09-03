// Package mcp provides functionality for managing Model Context Protocol (MCP)
// clients within the GiiS-Code application.
package mcp

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/GiiS-AI/GiiS-Code/internal/csync"
	"github.com/GiiS-AI/GiiS-Code/internal/home"
	"github.com/GiiS-AI/GiiS-Code/internal/permission"
	"github.com/GiiS-AI/GiiS-Code/internal/pubsub"
	"github.com/GiiS-AI/GiiS-Code/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// parseLevel converts an MCP logging level string to a slog.Level.
func parseLevel(level mcp.LoggingLevel) slog.Level {
	switch level {
	case "info":
		return slog.LevelInfo
	case "notice":
		return slog.LevelInfo
	case "warning":
		return slog.LevelWarn
	default:
		return slog.LevelDebug
	}
}

// ClientSession wraps an mcp.ClientSession with a context cancel function so
// that the context created during session establishment is properly cleaned up
// on close.
type ClientSession struct {
	*mcp.ClientSession
	cancel context.CancelFunc
}

// Close cancels the session context and then closes the underlying session.
func (s *ClientSession) Close() error {
	s.cancel()
	return s.ClientSession.Close()
}

var (
	sessions = csync.NewMap[string, *ClientSession]()
	states   = csync.NewMap[string, ClientInfo]()
	broker   = pubsub.NewBroker[Event]()
	initOnce sync.Once
	initDone = make(chan struct{})

	// initStarted records whether initialization was armed. WaitForInit only
	// blocks once startup expects MCP initialization to run.
	initMu      sync.Mutex
	initStarted bool
	initArmedAt time.Time

	// renewMus serializes lazy session renewals per server so concurrent tool
	// calls do not race to rebuild the same session.
	renewMusMu sync.Mutex
	renewMus   = map[string]*sync.Mutex{}

	// gens tracks a per-server generation. teardown bumps it so in-flight
	// initializations can detect they are stale before publishing results.
	gens = csync.NewMap[string, uint64]()

	// newSession is a test seam for renewal paths.
	newSession = createSession
)

// ArmInit marks that MCP initialization is expected, so WaitForInit blocks
// until it completes.
func ArmInit() {
	initMu.Lock()
	initStarted = true
	initArmedAt = time.Now()
	initMu.Unlock()
}

// DisarmInit exists for tests that need to restore the unarmed state.
func DisarmInit() {
	initMu.Lock()
	initStarted = false
	initMu.Unlock()
}

func renewLock(name string) *sync.Mutex {
	renewMusMu.Lock()
	defer renewMusMu.Unlock()
	if mu, ok := renewMus[name]; ok {
		return mu
	}
	mu := &sync.Mutex{}
	renewMus[name] = mu
	return mu
}

// State represents the current state of an MCP client
type State int

const (
	StateDisabled State = iota
	StateStarting
	StateConnected
	StateError
)

func (s State) String() string {
	switch s {
	case StateDisabled:
		return "disabled"
	case StateStarting:
		return "starting"
	case StateConnected:
		return "connected"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// EventType represents the type of MCP event
type EventType uint

const (
	EventStateChanged EventType = iota
	EventToolsListChanged
	EventPromptsListChanged
	EventResourcesListChanged
	EventChannelMessage
)

// Event represents an event in the MCP system
type Event struct {
	Type           EventType
	Name           string
	State          State
	Error          error
	Counts         Counts
	ChannelMessage string
}

// Counts number of available tools, prompts, etc.
type Counts struct {
	Tools     int
	Prompts   int
	Resources int
}

// ClientInfo holds information about an MCP client's state
type ClientInfo struct {
	Name          string
	State         State
	Error         error
	Client        *ClientSession
	Counts        Counts
	ConnectedAt   time.Time
	Config        config.MCPConfig
	PendingConfig *config.MCPConfig
}

// SubscribeEvents returns a channel for MCP events.
//
// Channel message events are filtered out here. The MCP broker is process
// global and those payloads do not yet carry workspace identity, so forwarding
// them through the shared app event stream would create a cross-workspace
// injection path.
func SubscribeEvents(ctx context.Context) <-chan pubsub.Event[Event] {
	raw := broker.Subscribe(ctx)
	filtered := make(chan pubsub.Event[Event], 64)
	go func() {
		defer close(filtered)
		for ev := range raw {
			if ev.Payload.Type == EventChannelMessage {
				continue
			}
			select {
			case filtered <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return filtered
}

// GetStates returns the current state of all MCP clients
func GetStates() map[string]ClientInfo {
	return states.Copy()
}

// GetState returns the state of a specific MCP client
func GetState(name string) (ClientInfo, bool) {
	return states.Get(name)
}

// Close closes all MCP clients. This should be called during application shutdown.
func Close(ctx context.Context) error {
	var wg sync.WaitGroup
	for name, session := range sessions.Seq2() {
		wg.Go(func() {
			done := make(chan error, 1)
			go func() {
				done <- session.Close()
			}()
			select {
			case err := <-done:
				if err != nil &&
					!errors.Is(err, io.EOF) &&
					!errors.Is(err, context.Canceled) &&
					err.Error() != "signal: killed" {
					slog.Warn("Failed to shutdown MCP client", "name", name, "error", err)
				}
			case <-ctx.Done():
			}
		})
	}
	wg.Wait()
	broker.Shutdown()
	return nil
}

// Initialize initializes MCP clients based on the provided configuration.
func Initialize(ctx context.Context, permissions permission.Service, cfg *config.ConfigStore) {
	_ = permissions
	ArmInit()
	slog.Info("Initializing MCP clients")
	var wg sync.WaitGroup
	// Initialize states for all configured MCPs
	for name, m := range cfg.Config().MCP {
		if m.Disabled {
			updateState(name, StateDisabled, nil, nil, Counts{})
			slog.Debug("Skipping disabled MCP", "name", name)
			continue
		}

		// Set initial starting state
		wg.Add(1)
		goInitClient(ctx, cfg, name, m, &wg)
	}
	wg.Wait()
	initOnce.Do(func() { close(initDone) })
}

// WaitForInit blocks until MCP initialization is complete.
func WaitForInit(ctx context.Context) error {
	initMu.Lock()
	started := initStarted
	initMu.Unlock()
	if !started {
		return nil
	}
	select {
	case <-initDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// InitWaitBudget bounds how long a caller waits for MCP startup before
// proceeding with whichever servers have registered so far.
const InitWaitBudget = 10 * time.Second

// WaitForInitBudget behaves like WaitForInit, but only until the budget
// measured from ArmInit elapses.
func WaitForInitBudget(ctx context.Context, budget time.Duration) error {
	initMu.Lock()
	started := initStarted
	armedAt := initArmedAt
	initMu.Unlock()
	if !started {
		return nil
	}
	waitCtx, cancel := context.WithDeadline(ctx, armedAt.Add(budget))
	defer cancel()
	if err := WaitForInit(waitCtx); err == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	slog.Warn("MCP initialization still pending after wait budget; continuing without unfinished servers", "budget", budget)
	return nil
}

// InitializeSingle initializes a single MCP client by name.
func InitializeSingle(ctx context.Context, name string, cfg *config.ConfigStore) error {
	m, exists := cfg.Config().MCP[name]
	if !exists {
		return fmt.Errorf("mcp '%s' not found in configuration", name)
	}

	if m.Disabled {
		updateState(name, StateDisabled, nil, nil, Counts{})
		slog.Debug("Skipping disabled MCP", "name", name)
		return nil
	}

	return initClient(ctx, cfg, name, m, currentGen(name), cfg.Resolver())
}

// initClient initializes a single MCP client with the given configuration.
func initClient(ctx context.Context, cfg *config.ConfigStore, name string, m config.MCPConfig, gen uint64, resolver config.VariableResolver) error {
	// Set initial starting state.
	updateState(name, StateStarting, nil, nil, Counts{}, withPending(m))

	// createSession handles its own timeout internally.
	session, err := createSession(ctx, name, m, resolver)
	if err != nil {
		return err
	}

	if currentGen(name) != gen {
		closeSession(name, session)
		return context.Canceled
	}

	var counts Counts
	counts.Tools, err = registerSessionTools(ctx, cfg, name, session)
	if err != nil {
		slog.Error("Error listing tools", "error", err)
		updateState(name, StateError, err, session, Counts{})
		return err
	}

	prompts, err := getPrompts(ctx, session)
	if err != nil {
		slog.Error("Error listing prompts", "error", err)
		updateState(name, StateError, err, session, Counts{})
		return err
	}
	updatePrompts(name, prompts)
	counts.Prompts = len(prompts)

	resources, err := getResources(ctx, session)
	if err != nil {
		slog.Error("Error listing resources", "error", err)
		updateState(name, StateError, err, session, Counts{})
		return err
	}
	counts.Resources = updateResources(name, resources)

	if currentGen(name) != gen {
		closeSession(name, session)
		return context.Canceled
	}

	sessions.Set(name, session)
	updateState(name, StateConnected, nil, session, counts, withConfig(m))

	return nil
}

// DisableSingle disables and closes a single MCP client by name.
func DisableSingle(cfg *config.ConfigStore, name string) error {
	_ = cfg
	teardown(name)
	updateState(name, StateDisabled, nil, nil, Counts{})
	slog.Info("Disabled mcp client", "name", name)
	return nil
}

func goInitClient(ctx context.Context, cfg *config.ConfigStore, name string, m config.MCPConfig, wg *sync.WaitGroup) {
	gen := currentGen(name)
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch v := r.(type) {
				case error:
					err = v
				case string:
					err = fmt.Errorf("panic: %s", v)
				default:
					err = fmt.Errorf("panic: %v", v)
				}
				updateState(name, StateError, err, nil, Counts{})
				slog.Error("Panic in MCP client initialization", "error", err, "name", name)
			}
		}()
		if err := initClient(ctx, cfg, name, m, gen, cfg.Resolver()); err != nil && !errors.Is(err, context.Canceled) {
			slog.Debug("Failed to initialize MCP client", "name", name, "error", err)
		}
	}()
}

func currentGen(name string) uint64 {
	gen, _ := gens.Get(name)
	return gen
}

func teardown(name string) {
	gen := currentGen(name)
	gens.Set(name, gen+1)
	if session, ok := sessions.Take(name); ok {
		closeSession(name, session)
	}
	clearMCPData(name)
}

func getOrRenewClient(ctx context.Context, cfg *config.ConfigStore, name string) (*ClientSession, error) {
	m := cfg.Config().MCP[name]
	timeout := mcpTimeout(m)

	if sess, ok := sessions.Get(name); ok {
		if err := pingSession(ctx, sess, timeout); err == nil {
			return sess, nil
		}
	}

	mu := renewLock(name)
	mu.Lock()
	defer mu.Unlock()

	sess, ok := sessions.Get(name)
	if !ok {
		return nil, fmt.Errorf("mcp '%s' not available", name)
	}

	pingErr := pingSession(ctx, sess, timeout)
	if pingErr == nil {
		return sess, nil
	}

	state, _ := states.Get(name)
	updateState(name, StateError, maybeTimeoutErr(pingErr, timeout), sess, state.Counts)

	gen := currentGen(name)
	newSess, err := newSession(ctx, name, m, cfg.Resolver())
	if err != nil {
		clearMCPData(name)
		return nil, err
	}

	if currentGen(name) != gen {
		closeSession(name, newSess)
		return nil, context.Canceled
	}

	var counts Counts
	counts.Tools, err = registerSessionTools(ctx, cfg, name, newSess)
	if err != nil {
		updateState(name, StateError, err, newSess, Counts{})
		return nil, err
	}

	prompts, err := getPrompts(ctx, newSess)
	if err != nil {
		updateState(name, StateError, err, newSess, Counts{})
		return nil, err
	}
	updatePrompts(name, prompts)
	counts.Prompts = len(prompts)

	resources, err := getResources(ctx, newSess)
	if err != nil {
		updateState(name, StateError, err, newSess, Counts{})
		return nil, err
	}
	counts.Resources = updateResources(name, resources)

	if currentGen(name) != gen {
		closeSession(name, newSess)
		return nil, context.Canceled
	}

	sessions.Set(name, newSess)
	updateState(name, StateConnected, nil, newSess, counts, withConfig(m))
	return newSess, nil
}

func pingSession(ctx context.Context, session *ClientSession, timeout time.Duration) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return session.Ping(pingCtx, nil)
}

func closeSession(name string, session *ClientSession) {
	if err := session.Close(); err != nil &&
		!errors.Is(err, io.EOF) &&
		!errors.Is(err, context.Canceled) &&
		err.Error() != "signal: killed" {
		slog.Warn("Error closing MCP session", "name", name, "error", err)
	}
}

type stateOpt func(*ClientInfo)

func withConfig(m config.MCPConfig) stateOpt {
	return func(info *ClientInfo) {
		info.Config = m
		info.PendingConfig = nil
	}
}

func withPending(m config.MCPConfig) stateOpt {
	return func(info *ClientInfo) {
		mc := m
		info.PendingConfig = &mc
	}
}

// updateState updates the state of an MCP client and publishes an event.
func updateState(name string, state State, err error, client *ClientSession, counts Counts, opts ...stateOpt) {
	prev, _ := states.Get(name)
	info := prev
	info.Name = name
	info.State = state
	info.Error = err
	info.Client = client
	info.Counts = counts
	for _, opt := range opts {
		opt(&info)
	}
	switch state {
	case StateConnected:
		info.ConnectedAt = time.Now()
	case StateDisabled:
		info.Config = config.MCPConfig{}
		info.PendingConfig = nil
	case StateError:
		switch {
		case client != nil:
			if cur, ok := sessions.Get(name); ok && cur == client {
				sessions.Del(name)
				clearMCPData(name)
			}
			closeSession(name, client)
		default:
			if old, ok := sessions.Take(name); ok {
				closeSession(name, old)
			}
			clearMCPData(name)
		}
		info.Client = nil
	}
	states.Set(name, info)

	// Publish state change event
	broker.Publish(pubsub.UpdatedEvent, Event{
		Type:   EventStateChanged,
		Name:   name,
		State:  state,
		Error:  err,
		Counts: counts,
	})
}

func createSession(ctx context.Context, name string, m config.MCPConfig, resolver config.VariableResolver) (*ClientSession, error) {
	timeout := mcpTimeout(m)
	mcpCtx, cancel := context.WithCancel(ctx)
	cancelTimer := time.AfterFunc(timeout, cancel)

	transport, err := createTransport(mcpCtx, m, resolver)
	if err != nil {
		updateState(name, StateError, err, nil, Counts{})
		slog.Error("Error creating MCP client", "error", err, "name", name)
		cancel()
		cancelTimer.Stop()
		return nil, err
	}

	channelGate := newChannelGate()
	transport = &channelTransport{inner: transport, name: name, gate: channelGate}

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "giis-code",
			Version: version.Version,
			Title:   "GiiS-Code",
		},
		&mcp.ClientOptions{
			ToolListChangedHandler: func(context.Context, *mcp.ToolListChangedRequest) {
				broker.Publish(pubsub.UpdatedEvent, Event{
					Type: EventToolsListChanged,
					Name: name,
				})
			},
			PromptListChangedHandler: func(context.Context, *mcp.PromptListChangedRequest) {
				broker.Publish(pubsub.UpdatedEvent, Event{
					Type: EventPromptsListChanged,
					Name: name,
				})
			},
			ResourceListChangedHandler: func(context.Context, *mcp.ResourceListChangedRequest) {
				broker.Publish(pubsub.UpdatedEvent, Event{
					Type: EventResourcesListChanged,
					Name: name,
				})
			},
			LoggingMessageHandler: func(ctx context.Context, req *mcp.LoggingMessageRequest) {
				level := parseLevel(req.Params.Level)
				slog.Log(ctx, level, "MCP log", "name", name, "logger", req.Params.Logger, "data", req.Params.Data)
			},
		},
	)

	session, err := client.Connect(mcpCtx, transport, nil)
	if err != nil {
		err = maybeStdioErr(err, transport)
		updateState(name, StateError, maybeTimeoutErr(err, timeout), nil, Counts{})
		slog.Error("MCP client failed to initialize", "error", err, "name", name)
		cancel()
		cancelTimer.Stop()
		return nil, err
	}

	cancelTimer.Stop()
	slog.Debug("MCP client initialized", "name", name)
	channelGate.resolve(false)
	return &ClientSession{session, cancel}, nil
}

type transportWrapper interface {
	unwrapTransport() mcp.Transport
}

func unwrapTransport(transport mcp.Transport) mcp.Transport {
	for {
		wrapped, ok := transport.(transportWrapper)
		if !ok {
			return transport
		}
		transport = wrapped.unwrapTransport()
	}
}

// maybeStdioErr if a stdio mcp prints an error in non-json format, it'll fail
// to parse, and the cli will then close it, causing the EOF error.
// so, if we got an EOF err, and the transport is STDIO, we try to exec it
// again with a timeout and collect the output so we can add details to the
// error.
// this happens particularly when starting things with npx, e.g. if node can't
// be found or some other error like that.
func maybeStdioErr(err error, transport mcp.Transport) error {
	if !errors.Is(err, io.EOF) {
		return err
	}
	transport = unwrapTransport(transport)
	ct, ok := transport.(*mcp.CommandTransport)
	if !ok {
		return err
	}
	if err2 := stdioCheck(ct.Command); err2 != nil {
		err = errors.Join(err, err2)
	}
	return err
}

func maybeTimeoutErr(err error, timeout time.Duration) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("timed out after %s", timeout)
	}
	return err
}

func createTransport(ctx context.Context, m config.MCPConfig, resolver config.VariableResolver) (mcp.Transport, error) {
	switch m.Type {
	case config.MCPStdio:
		command, err := resolver.ResolveValue(m.Command)
		if err != nil {
			return nil, fmt.Errorf("invalid mcp command: %w", err)
		}
		if strings.TrimSpace(command) == "" {
			return nil, fmt.Errorf("mcp stdio config requires a non-empty 'command' field")
		}
		args, err := m.ResolvedArgs(resolver)
		if err != nil {
			return nil, err
		}
		envs, err := m.ResolvedEnv(resolver)
		if err != nil {
			return nil, err
		}
		cmd := exec.CommandContext(ctx, home.Long(command), args...)
		configureStdioProcess(cmd)
		cmd.Env = append(os.Environ(), envs...)
		return &mcp.CommandTransport{
			Command: cmd,
		}, nil
	case config.MCPHttp:
		url, err := m.ResolvedURL(resolver)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(url) == "" {
			return nil, fmt.Errorf("mcp http config requires a non-empty 'url' field")
		}
		headers, err := m.ResolvedHeaders(resolver)
		if err != nil {
			return nil, err
		}
		client := &http.Client{
			Transport: &headerRoundTripper{
				headers: headers,
			},
		}
		return &mcp.StreamableClientTransport{
			Endpoint:   url,
			HTTPClient: client,
		}, nil
	case config.MCPSSE:
		url, err := m.ResolvedURL(resolver)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(url) == "" {
			return nil, fmt.Errorf("mcp sse config requires a non-empty 'url' field")
		}
		headers, err := m.ResolvedHeaders(resolver)
		if err != nil {
			return nil, err
		}
		client := &http.Client{
			Transport: &headerRoundTripper{
				headers: headers,
			},
		}
		return &mcp.SSEClientTransport{
			Endpoint:   url,
			HTTPClient: client,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported mcp type: %s", m.Type)
	}
}

type headerRoundTripper struct {
	headers map[string]string
}

func (rt headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range rt.headers {
		req.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(req)
}

func mcpTimeout(m config.MCPConfig) time.Duration {
	return time.Duration(cmp.Or(m.Timeout, 15)) * time.Second
}

func stdioCheck(old *exec.Cmd) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	args := old.Args
	if len(args) > 0 {
		args = args[1:]
	}
	cmd := exec.CommandContext(ctx, old.Path, args...)
	cmd.Env = old.Env
	out, err := cmd.CombinedOutput()
	if err == nil || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil
	}
	return fmt.Errorf("%w: %s", err, string(out))
}

func clearMCPData(name string) {
	allTools.Del(name)
	allPrompts.Del(name)
	allResources.Del(name)
}
