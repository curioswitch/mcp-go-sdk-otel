package mcpotel

import (
	"context"
	"fmt"
	"log"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type SayHiParams struct {
	Name string `json:"name"`
}

func ExampleReceivingMiddleware() {
	ctx := context.Background()

	otel.SetTextMapPropagator(propagation.TraceContext{})
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	echoTraceIDs := func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		spanCtx := trace.SpanContextFromContext(ctx)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("%s/%s", spanCtx.TraceID().String(), spanCtx.SpanID().String())},
			},
		}, nil
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "receiver", Version: "v0.0.1"}, nil)
	server.AddReceivingMiddleware(ReceivingMiddleware())
	server.AddTool(
		&mcp.Tool{
			Name:        "echo",
			Description: "Echo trace IDs",
			InputSchema: &jsonschema.Schema{
				Type: "object",
			},
		},
		echoTraceIDs,
	)

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		log.Fatal(err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "echo",
		Meta: map[string]any{
			"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res.Content[0].(*mcp.TextContent).Text)
	// Output: 0af7651916cd43dd8448eb211c80319c/b7ad6b7169203331

	clientSession.Close()
	serverSession.Wait()
}

func ExampleSendingMiddleware() {
	ctx := context.Background()

	otel.SetTextMapPropagator(propagation.TraceContext{})
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	echoTraceIDs := func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		traceparent := ""
		if meta := req.GetParams().GetMeta(); meta != nil {
			if s, ok := meta["traceparent"].(string); ok {
				traceparent = s
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: traceparent},
			},
		}, nil
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "receiver", Version: "v0.0.1"}, nil)
	server.AddTool(
		&mcp.Tool{
			Name:        "echo",
			Description: "Echo traceparent",
			InputSchema: &jsonschema.Schema{
				Type: "object",
			},
		},
		echoTraceIDs,
	)

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		log.Fatal(err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	client.AddSendingMiddleware(SendingMiddleware())
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatal(err)
	}

	traceID, _ := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
	spanID, _ := trace.SpanIDFromHex("b7ad6b7169203331")
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx = trace.ContextWithSpanContext(ctx, spanCtx)

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "echo",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res.Content[0].(*mcp.TextContent).Text)
	// Output: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01

	clientSession.Close()
	serverSession.Wait()
}
