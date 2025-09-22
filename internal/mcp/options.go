package mcp

// Options configures the MCP server startup.
type Options struct {
	// Transport selects the server transport. Supported: "stdio", "http".
	Transport string
	// Addr is the listen address for HTTP transport.
	Addr string
	// AllowInsecureSetKey toggles a local-only tool to set API key via MCP.
	AllowInsecureSetKey bool
	// Verbose enables extra debug logging.
	Verbose bool
}

const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)
