package models

// MCPRequest represents an MCP protocol request
type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      string                 `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP protocol response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorObj   `json:"error,omitempty"`
}

// ErrorObj represents an error in MCP response
type ErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ToolName string                 `json:"toolName"`
	Args     map[string]interface{} `json:"args"`
}

// PDFOperation represents parameters for PDF operations
type PDFOperation struct {
	Operation string   `json:"operation"` // merge, split, optimize, export, toc
	Files     []string `json:"files"`
	Pages     []int    `json:"pages,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// ImageOperation represents parameters for image operations
type ImageOperation struct {
	Operation string                 `json:"operation"` // to_pdf, merge, convert
	Files     []string               `json:"files"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// OperationResult represents the result of an operation
type OperationResult struct {
	Success   bool        `json:"success"`
	OutputURL string      `json:"output_url,omitempty"`
	FileSize  int64       `json:"file_size,omitempty"`
	Pages     int         `json:"pages,omitempty"`
	Message   string      `json:"message"`
	Error     string      `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// HealthCheck represents server health status
type HealthCheck struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  int64  `json:"uptime_seconds"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Tools       []Tool        `json:"tools"`
}
