// Hand-written (not generated). Exposes MCP sampling support on both the
// server side (ToolWithSampling, ServerSession) and the client side
// (NewClientWithSampling, NewClientWithSamplingTools — defined in mcp.kuki).
package mcp

import (
	"context"

	"github.com/kukichalang/kukicha/stdlib/json"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolSamplingHandler is a tool handler that receives a *ServerSession so it
// can invoke sampling (createMessage) on the connected AI client.
// Register it with ToolWithSampling.
type ToolSamplingHandler func(ctx context.Context, session *ServerSession, args map[string]any) (any, error)

// ServerSession wraps *gomcp.ServerSession so tool handlers can call sampling
// without a direct dependency on the go-sdk.
type ServerSession struct {
	inner *gomcp.ServerSession
}

// CreateMessage sends a sampling/createMessage request to the connected client
// and returns the AI-generated response.  Use this inside a ToolWithSampling
// handler to get AI completions during tool execution.
func (ss *ServerSession) CreateMessage(ctx context.Context, params *CreateMessageParams) (*CreateMessageResult, error) {
	return ss.inner.CreateMessage(ctx, params)
}

// CreateMessageWithTools sends a sampling request that includes tools and
// supports array content in the response (parallel tool_use blocks).
func (ss *ServerSession) CreateMessageWithTools(ctx context.Context, params *CreateMessageWithToolsParams) (*CreateMessageWithToolsResult, error) {
	return ss.inner.CreateMessageWithTools(ctx, params)
}

// ToolWithSampling registers an MCP tool whose handler receives a *ServerSession
// for sampling.  This is the sampling-capable analogue of Tool.
//
// Example:
//
//	mcp.ToolWithSampling(server, "chat", "Chat with the AI", schema,
//	    func(ctx context.Context, session *mcp.ServerSession, args map[string]any) (any, error) {
//	        msgs := []*mcp.SamplingMessage{{Role: "user", Content: &mcp.TextContent{Text: args["message"].(string)}}}
//	        result, err := session.CreateMessage(ctx, &mcp.CreateMessageParams{Messages: msgs, MaxTokens: 1024})
//	        if err != nil { return nil, err }
//	        return result.Content.(*mcp.TextContent).Text, nil
//	    })
func ToolWithSampling(server *gomcp.Server, name, description string, schema any, handler ToolSamplingHandler) {
	server.AddTool(&gomcp.Tool{Name: name, Description: description, InputSchema: schema},
		func(ctx context.Context, req *gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
			args := make(map[string]any)
			if len(req.Params.Arguments) > 0 {
				if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
					return nil, err
				}
			}
			ss := &ServerSession{inner: req.Session}
			res, handlerErr := handler(ctx, ss, args)
			if handlerErr != nil {
				return &gomcp.CallToolResult{
					Content: []gomcp.Content{&gomcp.TextContent{Text: handlerErr.Error()}},
					IsError: true,
				}, nil
			}
			switch r := res.(type) {
			case *gomcp.CallToolResult:
				return r, nil
			case string:
				return &gomcp.CallToolResult{Content: []gomcp.Content{&gomcp.TextContent{Text: r}}}, nil
			}
			data, _ := json.Marshal(res)
			return &gomcp.CallToolResult{Content: []gomcp.Content{&gomcp.TextContent{Text: string(data)}}}, nil
		})
}
