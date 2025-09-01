package mcpotel

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReceivingMiddleware returns middleware to extract OpenTelemetry context from received messages.
//
// While receiving middleware is most commonly utilized on the server, it can also be useful on the client
// in case the server calls functions exposed by the client.
//
//	func main() {
//		// Initialize your server and/or client
//		// ...
//
//		server.AddReceivingMiddleware(mcpotel.ReceivingMiddleware())
//		client.AddReceivingMiddleware(mcpotel.ReceivingMiddleware())
//	}
func ReceivingMiddleware(opts ...Option) mcp.Middleware {
	conf := newConfig(opts)
	return func(mh mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			ctx = conf.propagator.Extract(ctx, stringAnyMapCarrier{req.GetParams().GetMeta()})
			return mh(ctx, method, req)
		}
	}
}

// SendingMiddleware returns middleware to inject OpenTelemetry context into sent messages.
//
// While sending middleware is most commonly utilized on the client, it can also be useful on the server
// in case the server calls functions exposed by the client.
//
//	func main() {
//		// Initialize your server and/or client
//		// ...
//
//		client.AddSendingMiddleware(mcpotel.SendingMiddleware())
//		server.AddSendingMiddleware(mcpotel.SendingMiddleware())
//	}
func SendingMiddleware(opts ...Option) mcp.Middleware {
	conf := newConfig(opts)
	return func(mh mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			params := req.GetParams()
			meta := params.GetMeta()
			if meta == nil {
				meta = make(map[string]any)
				params.SetMeta(meta)
			}
			conf.propagator.Inject(ctx, stringAnyMapCarrier{meta})
			return mh(ctx, method, req)
		}
	}
}

type stringAnyMapCarrier struct {
	m map[string]any
}

// Get implements propagation.TextMapCarrier.
func (a stringAnyMapCarrier) Get(key string) string {
	if a.m == nil {
		return ""
	}
	if s, ok := a.m[key].(string); ok {
		return s
	}
	return ""
}

// Keys implements propagation.TextMapCarrier.
func (a stringAnyMapCarrier) Keys() []string {
	if len(a.m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(a.m))
	for k := range a.m {
		keys = append(keys, k)
	}
	return keys
}

// Set implements propagation.TextMapCarrier.
func (a stringAnyMapCarrier) Set(key string, value string) {
	a.m[key] = value
}
