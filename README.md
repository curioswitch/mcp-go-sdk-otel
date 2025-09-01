# OpenTelemetry instrumentation for modelcontextprotocol/go-sdk

This library provides middleware to provide observability of MCP servers and clients
using OpenTelemetry. Currently only trace propagation is implemented, full traces
between when making tool calls that themselves generate traces such as for Gen AI
SDKs.

## Installation

Add `github.com/curioswitch/mcp-go-sdk-otel` to your go.mod.

```bash
go get github.com/curioswitch/mcp-go-sdk-otel@latest
```

## Usage

Add middleware to clients and servers to enable instrumentation. While typical usage
requires adding sending middleware to clients and receiving middleware to servers, it
is possible to have invocations in the other direction and can make sense to add both.

```go
server := mcp.NewServer(&mcp.Implementation{Name: "server", Version: "v0.0.1"}, nil)
server.AddReceivingMiddleware(mcpotel.ReceivingMiddleware())
server.AddSendingMiddleware(mcpotel.SendingMiddleware())

client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
client.AddSendingMiddleware(mcpotel.SendingMiddleware())
server.AddReceivingMiddleware(mcpotel.ReceivingMiddleware())
```

See the [package docs](https://pkg.go.dev/mod/github.com/curioswitch/mcp-go-sdk-otel) for more information.
