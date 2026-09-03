package mcp

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/GiiS-AI/GiiS-Code/internal/pubsub"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	channelCapability         = "claude/channel"
	channelNotificationMethod = "notifications/claude/channel"
	maxChannelContentBytes    = 64 * 1024
	maxChannelMetaEntries     = 32
	maxChannelMetaValueBytes  = 1024
)

var metaKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var reservedMetaKeys = map[string]struct{}{
	"source": {},
	"xmlns":  {},
	"xml":    {},
}

type channelParams struct {
	Content string            `json:"content"`
	Meta    map[string]string `json:"meta"`
}

func parseChannelParams(raw json.RawMessage) (channelParams, bool) {
	if len(raw) == 0 {
		return channelParams{}, false
	}

	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()

	var p channelParams
	if err := dec.Decode(&p); err != nil {
		return channelParams{}, false
	}
	if p.Content == "" || len(p.Content) > maxChannelContentBytes {
		return channelParams{}, false
	}

	clean := channelParams{Content: p.Content}
	if len(p.Meta) > 0 {
		clean.Meta = make(map[string]string, len(p.Meta))
		for k, v := range p.Meta {
			if len(clean.Meta) >= maxChannelMetaEntries {
				break
			}
			if !metaKeyPattern.MatchString(k) {
				continue
			}
			if _, reserved := reservedMetaKeys[k]; reserved {
				continue
			}
			if len(v) > maxChannelMetaValueBytes {
				continue
			}
			clean.Meta[k] = v
		}
	}
	return clean, true
}

func renderChannel(source string, p channelParams) string {
	start := xml.StartElement{
		Name: xml.Name{Local: "channel"},
		Attr: make([]xml.Attr, 0, 1+len(p.Meta)),
	}
	start.Attr = append(start.Attr, xml.Attr{
		Name:  xml.Name{Local: "source"},
		Value: source,
	})

	keys := make([]string, 0, len(p.Meta))
	for k := range p.Meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: k},
			Value: p.Meta[k],
		})
	}

	var b strings.Builder
	enc := xml.NewEncoder(&b)
	_ = enc.EncodeToken(start)
	_ = enc.EncodeToken(xml.CharData(p.Content))
	_ = enc.EncodeToken(start.End())
	_ = enc.Flush()
	return b.String()
}

func hasChannelCapability(res *mcp.InitializeResult) bool {
	if res == nil || res.Capabilities == nil {
		return false
	}
	_, ok := res.Capabilities.Experimental[channelCapability]
	return ok
}

func publishChannelMessage(ctx context.Context, name string, raw json.RawMessage) {
	p, ok := parseChannelParams(raw)
	if !ok {
		slog.Warn("Dropping malformed channel notification", "server", name)
		return
	}
	broker.PublishMustDeliver(ctx, pubsub.CreatedEvent, Event{
		Type:           EventChannelMessage,
		Name:           name,
		ChannelMessage: renderChannel(name, p),
	})
}

type channelGateState int32

const (
	stateGateUndecided channelGateState = iota
	stateGateOpen
	stateGateClosed
)

type channelGate struct {
	state   atomic.Int32
	mu      sync.Mutex
	pending []json.RawMessage
}

func newChannelGate() *channelGate {
	g := &channelGate{}
	g.state.Store(int32(stateGateUndecided))
	return g
}

func (g *channelGate) resolve(open bool) []json.RawMessage {
	g.mu.Lock()
	defer g.mu.Unlock()
	if channelGateState(g.state.Load()) != stateGateUndecided {
		return nil
	}
	buffered := g.pending
	g.pending = nil
	if open {
		g.state.Store(int32(stateGateOpen))
		return buffered
	}
	g.state.Store(int32(stateGateClosed))
	return nil
}

func (g *channelGate) accept(raw json.RawMessage) json.RawMessage {
	switch channelGateState(g.state.Load()) {
	case stateGateOpen:
		return raw
	case stateGateClosed:
		return nil
	default:
		g.mu.Lock()
		defer g.mu.Unlock()
		switch channelGateState(g.state.Load()) {
		case stateGateOpen:
			return raw
		case stateGateClosed:
			return nil
		}
		g.pending = append(g.pending, raw)
		return nil
	}
}

type channelTransport struct {
	inner mcp.Transport
	name  string
	gate  *channelGate
}

func (t *channelTransport) unwrapTransport() mcp.Transport { return t.inner }

func (t *channelTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &channelConn{Connection: conn, name: t.name, gate: t.gate}, nil
}

type channelConn struct {
	mcp.Connection
	name string
	gate *channelGate
}

func (c *channelConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	for {
		msg, err := c.Connection.Read(ctx)
		if err != nil {
			return msg, err
		}
		req, ok := msg.(*jsonrpc.Request)
		if !ok || req.IsCall() || req.Method != channelNotificationMethod {
			return msg, nil
		}
		if raw := c.gate.accept(req.Params); raw != nil {
			publishChannelMessage(ctx, c.name, raw)
		}
	}
}
