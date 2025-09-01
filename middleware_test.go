package mcpotel

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestBidi(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	tracer := tp.Tracer("test")

	hello := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, serverSpan := tracer.Start(ctx, "02-hello")
		defer serverSpan.End()

		if sess, ok := req.GetSession().(*mcp.ServerSession); ok {
			_, err := sess.CreateMessage(ctx, &mcp.CreateMessageParams{})
			if err != nil {
				return nil, fmt.Errorf("sending create message request: %w", err)
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Hello"},
			},
		}, nil
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
	server.AddReceivingMiddleware(ReceivingMiddleware())
	server.AddSendingMiddleware(SendingMiddleware())
	server.AddTool(
		&mcp.Tool{
			Name:        "hello",
			Description: "Says hello",
			InputSchema: &jsonschema.Schema{
				Type: "object",
			},
		},
		hello,
	)

	ctx := context.Background()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Wait()

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, &mcp.ClientOptions{
		CreateMessageHandler: func(ctx context.Context, _ *mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
			_, clientSpan := tracer.Start(ctx, "03-create_message")
			defer clientSpan.End()

			return &mcp.CreateMessageResult{
				Content: &mcp.TextContent{Text: "Hello"},
			}, nil
		},
	})
	client.AddReceivingMiddleware(ReceivingMiddleware())
	client.AddSendingMiddleware(SendingMiddleware())
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	func() {
		ctx, rootSpan := tracer.Start(ctx, "01-root")
		defer rootSpan.End()

		res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
			Name: "hello",
		})
		require.NoError(t, err)
		require.Len(t, res.Content, 1)
		require.Equal(t, "Hello", res.Content[0].(*mcp.TextContent).Text)
	}()

	spans := exp.GetSpans()
	// We sort by span name since it is reliable even if machine time precision
	// is low such as on Windows.
	slices.SortFunc(spans, func(a, b tracetest.SpanStub) int {
		return strings.Compare(a.Name, b.Name)
	})
	require.Len(t, spans, 3)
	require.Equal(t, "01-root", spans[0].Name)
	require.False(t, spans[0].Parent.IsValid())
	require.Equal(t, "02-hello", spans[1].Name)
	require.Equal(t, spans[0].SpanContext.SpanID().String(), spans[1].Parent.SpanID().String())
	require.Equal(t, "03-create_message", spans[2].Name)
	require.Equal(t, spans[1].SpanContext.SpanID().String(), spans[2].Parent.SpanID().String())
}
