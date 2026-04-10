// Type aliases exposing go-sdk content and parameter types from stdlib/mcp so
// callers can type-assert CallToolResult.Content elements without a direct
// dependency on github.com/modelcontextprotocol/go-sdk/mcp.
//
// Usage:
//
//	for _, c := range result.Content {
//	    switch v := c.(type) {
//	    case *mcp.TextContent:
//	        fmt.Println(v.Text)
//	    case *mcp.ImageContent:
//	        fmt.Println(v.MIMEType)
//	    }
//	}
package mcp

import gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

// TextContent is a content item containing plain text.
type TextContent = gomcp.TextContent

// ImageContent is a content item containing base64-encoded image data.
type ImageContent = gomcp.ImageContent

// CallToolParams holds the parameters for a CallTool request.
// Use with (*ClientSession).CallTool:
//
//	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "tool", Arguments: args})
type CallToolParams = gomcp.CallToolParams

// Sampling types — exposed so callers can construct and inspect sampling
// requests and responses without a direct go-sdk dependency.

// SamplingMessage is a single-content sampling conversation turn (pre-2025-11-25 wire format).
type SamplingMessage = gomcp.SamplingMessage

// SamplingMessageV2 is a multi-content sampling conversation turn (2025-11-25 wire format).
// Use this with CreateMessageWithToolsParams to support parallel tool calls.
type SamplingMessageV2 = gomcp.SamplingMessageV2

// CreateMessageParams is the request sent by the server to the client for sampling.
type CreateMessageParams = gomcp.CreateMessageParams

// CreateMessageWithToolsParams is a sampling request that includes tools and tool-choice.
// Use with (*ServerSession).CreateMessageWithTools and SamplingWithToolsHandler.
type CreateMessageWithToolsParams = gomcp.CreateMessageWithToolsParams

// CreateMessageResult is the client's response to a basic sampling request.
type CreateMessageResult = gomcp.CreateMessageResult

// CreateMessageWithToolsResult is the client's response to a tool-enabled sampling request.
// Content is a slice to support multiple parallel tool-use blocks.
type CreateMessageWithToolsResult = gomcp.CreateMessageWithToolsResult

// ToolUseContent represents a request from the assistant to invoke a tool.
// Only valid in sampling messages.
type ToolUseContent = gomcp.ToolUseContent

// ToolResultContent represents the result of a tool invocation.
// Only valid in sampling messages with role "user".
type ToolResultContent = gomcp.ToolResultContent

// ToolChoice controls how the model uses tools during sampling.
type ToolChoice = gomcp.ToolChoice

// SamplingCapabilities describes the client's support for sampling.
type SamplingCapabilities = gomcp.SamplingCapabilities

// SamplingToolsCapabilities indicates the client supports tool use in sampling.
type SamplingToolsCapabilities = gomcp.SamplingToolsCapabilities
