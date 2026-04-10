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
