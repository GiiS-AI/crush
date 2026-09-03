package mcp

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/GiiS-AI/GiiS-Code/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func liveSession(t *testing.T, toolName string) (*ClientSession, context.Context) {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := mcp.NewServer(&mcp.Implementation{Name: "srv"}, nil)
	mcp.AddTool(
		server,
		&mcp.Tool{Name: toolName, Description: "test tool"},
		func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
		},
	)
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	client := mcp.NewClient(&mcp.Implementation{Name: "c0d3r-test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)

	return &ClientSession{ClientSession: clientSession, cancel: cancel}, ctx
}

func liveSessionWithCapabilities(t *testing.T, toolName, promptName, resourceURI string) *ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := mcp.NewServer(&mcp.Implementation{Name: "srv"}, nil)
	mcp.AddTool(
		server,
		&mcp.Tool{Name: toolName, Description: "test tool"},
		func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
		},
	)
	server.AddPrompt(
		&mcp.Prompt{Name: promptName},
		func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{}, nil
		},
	)
	server.AddResource(
		&mcp.Resource{Name: "res", URI: resourceURI},
		func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{}, nil
		},
	)
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	client := mcp.NewClient(&mcp.Implementation{Name: "c0d3r-test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)

	return &ClientSession{ClientSession: clientSession, cancel: cancel}
}

func TestReconcile(t *testing.T) {
	base := config.MCPConfig{Type: config.MCPHttp, URL: "https://example.com/mcp"}
	changed := base
	changed.URL = "https://other.example.com/mcp"
	disabled := base
	disabled.Disabled = true

	ptr := func(m config.MCPConfig) *config.MCPConfig { return &m }

	tests := []struct {
		name    string
		servers map[string]ClientInfo
		current config.MCPs
		want    map[string]reinitAction
	}{
		{
			name:    "new server starts",
			current: config.MCPs{"a": base},
			want:    map[string]reinitAction{"a": reinitStart},
		},
		{
			name: "removed server is cleaned up",
			servers: map[string]ClientInfo{
				"gone": {State: StateConnected, Config: base},
			},
			current: config.MCPs{},
			want:    map[string]reinitAction{"gone": reinitRemove},
		},
		{
			name: "unchanged connected server is skipped",
			servers: map[string]ClientInfo{
				"a": {State: StateConnected, Config: base},
			},
			current: config.MCPs{"a": base},
			want:    map[string]reinitAction{},
		},
		{
			name: "changed connected server restarts",
			servers: map[string]ClientInfo{
				"a": {State: StateConnected, Config: base},
			},
			current: config.MCPs{"a": changed},
			want:    map[string]reinitAction{"a": reinitStart},
		},
		{
			name: "disabled server is disabled",
			servers: map[string]ClientInfo{
				"a": {State: StateConnected, Config: base},
			},
			current: config.MCPs{"a": disabled},
			want:    map[string]reinitAction{"a": reinitDisable},
		},
		{
			name: "starting server with matching pending config is left alone",
			servers: map[string]ClientInfo{
				"a": {State: StateStarting, PendingConfig: ptr(base)},
			},
			current: config.MCPs{"a": base},
			want:    map[string]reinitAction{},
		},
		{
			name: "starting server with changed config restarts",
			servers: map[string]ClientInfo{
				"a": {State: StateStarting, PendingConfig: ptr(base)},
			},
			current: config.MCPs{"a": changed},
			want:    map[string]reinitAction{"a": reinitStart},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, reconcile(tc.current, tc.servers))
		})
	}
}

func TestMCPConfigEqualExhaustive(t *testing.T) {
	excluded := map[string]bool{}
	typ := reflect.TypeOf(config.MCPConfig{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if excluded[name] {
			continue
		}
		a := config.MCPConfig{}
		b := config.MCPConfig{}
		setDistinct(typ.Field(i).Type, reflect.ValueOf(&a).Elem().Field(i))
		require.Falsef(t, mcpConfigEqual(a, b), "mcpConfigEqual ignores field %q", name)
	}
}

func setDistinct(typ reflect.Type, field reflect.Value) {
	switch typ.Kind() {
	case reflect.String:
		field.SetString("x")
	case reflect.Int:
		field.SetInt(1)
	case reflect.Bool:
		field.SetBool(true)
	case reflect.Slice:
		field.Set(reflect.MakeSlice(typ, 1, 1))
		field.Index(0).Set(reflect.Zero(typ.Elem()))
		if typ.Elem().Kind() == reflect.String {
			field.Index(0).SetString("x")
		}
	case reflect.Map:
		field.Set(reflect.MakeMapWithSize(typ, 1))
		k := reflect.New(typ.Key()).Elem()
		v := reflect.New(typ.Elem()).Elem()
		if typ.Key().Kind() == reflect.String {
			k.SetString("x")
		}
		if typ.Elem().Kind() == reflect.String {
			v.SetString("x")
		}
		field.SetMapIndex(k, v)
	default:
		panic("unsupported kind " + typ.Kind().String())
	}
}

func TestUpdateState_ErrorFromStaleSessionPreservesHealthyReplacement(t *testing.T) {
	const name = "stale-error"
	t.Cleanup(func() {
		sessions.Del(name)
		allTools.Del(name)
		allPrompts.Del(name)
		allResources.Del(name)
		states.Del(name)
	})

	stale, staleCtx := liveSession(t, "old_tool")
	fresh, freshCtx := liveSession(t, "new_tool")

	sessions.Set(name, fresh)
	allTools.Set(name, []*Tool{{Name: "new_tool"}})
	allPrompts.Set(name, []*Prompt{{Name: "new_prompt"}})

	updateState(name, StateError, errors.New("ping timeout"), stale, Counts{})

	got, ok := sessions.Get(name)
	require.True(t, ok)
	require.Same(t, fresh, got)
	require.NoError(t, freshCtx.Err())
	_, ok = allTools.Get(name)
	require.True(t, ok)
	_, ok = allPrompts.Get(name)
	require.True(t, ok)
	require.ErrorIs(t, staleCtx.Err(), context.Canceled)
}

func TestUpdateState_ErrorFromCurrentSessionClearsEverything(t *testing.T) {
	const name = "current-error"
	t.Cleanup(func() {
		sessions.Del(name)
		allTools.Del(name)
		allPrompts.Del(name)
		allResources.Del(name)
		states.Del(name)
	})

	sess, sessCtx := liveSession(t, "do_thing")
	sessions.Set(name, sess)
	allTools.Set(name, []*Tool{{Name: "do_thing"}})
	allPrompts.Set(name, []*Prompt{{Name: "a_prompt"}})
	allResources.Set(name, []*Resource{{Name: "a_resource"}})

	updateState(name, StateError, errors.New("pipe broke"), sess, Counts{})

	_, ok := sessions.Get(name)
	require.False(t, ok)
	require.ErrorIs(t, sessCtx.Err(), context.Canceled)
	_, ok = allTools.Get(name)
	require.False(t, ok)
	_, ok = allPrompts.Get(name)
	require.False(t, ok)
	_, ok = allResources.Get(name)
	require.False(t, ok)
}

func TestGetOrRenewClient_SerializesConcurrentRenewals(t *testing.T) {
	const name = "renew-concurrency"
	const workers = 8

	t.Cleanup(func() {
		if s, ok := sessions.Take(name); ok {
			_ = s.Close()
		}
		allTools.Del(name)
		allPrompts.Del(name)
		allResources.Del(name)
		states.Del(name)
	})

	cfg := config.NewTestStore(&config.Config{
		MCP: config.MCPs{name: {Type: config.MCPStdio}},
	})

	dead, _ := liveSession(t, "send_message")
	require.NoError(t, dead.Close())
	sessions.Set(name, dead)

	replacements := make(chan *ClientSession, workers)
	for i := 0; i < workers; i++ {
		replacements <- liveSessionWithCapabilities(t, "send_message", "a_prompt", "res://thing")
	}
	close(replacements)

	var created atomic.Int32
	origNewSession := newSession
	newSession = func(context.Context, string, config.MCPConfig, config.VariableResolver) (*ClientSession, error) {
		created.Add(1)
		return <-replacements, nil
	}
	t.Cleanup(func() { newSession = origNewSession })

	var wg sync.WaitGroup
	results := make([]*ClientSession, workers)
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = getOrRenewClient(context.Background(), cfg, name)
		}(i)
	}
	wg.Wait()

	require.Equal(t, int32(1), created.Load())
	final, ok := sessions.Get(name)
	require.True(t, ok)
	for i := 0; i < workers; i++ {
		require.NoError(t, errs[i])
		require.Same(t, final, results[i])
	}

	info, ok := GetState(name)
	require.True(t, ok)
	require.Equal(t, Counts{Tools: 1, Prompts: 1, Resources: 1}, info.Counts)
}

type testTransportWrapper struct {
	mcp.Transport
	inner mcp.Transport
}

func (t *testTransportWrapper) unwrapTransport() mcp.Transport { return t.inner }

func TestMaybeStdioErr_UnwrapsEveryWrapper(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sh", "-c", "echo boom-diagnostic >&2; exit 3")
	var transport mcp.Transport = &mcp.CommandTransport{Command: cmd}
	transport = &channelTransport{inner: transport, name: "t", gate: newChannelGate()}
	transport = &testTransportWrapper{inner: transport}

	got := maybeStdioErr(io.EOF, transport)
	require.ErrorContains(t, got, "boom-diagnostic")
}

func TestStdioCheck_DoesNotDuplicateArgv0(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sh", "-c", "echo 'real startup error'; exit 3")

	err := stdioCheck(cmd)
	require.Error(t, err)
	require.ErrorContains(t, err, "real startup error")
	require.NotContains(t, err.Error(), "cannot execute binary file")
}
