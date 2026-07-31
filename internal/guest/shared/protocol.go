package shared

type JSONRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type JSONResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      int           `json:"id"`
}

type ToolCallParams struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolCallResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError"`
}

type TextContent struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type ExecRequest struct {
	ToolName   string   `json:"tool_name"`
	Args       []string `json:"args"`
	TimeoutSec int      `json:"timeout_sec"`
}

type ExecResult struct {
	Success  bool   `json:"success"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	ErrorMsg string `json:"error_msg"`
}

type SystemInfo struct {
	OSName          string   `json:"os_name"`
	OSVersion       string   `json:"os_version"`
	IsPentestDistro bool     `json:"is_pentest_distro"`
	AvailableTools  []string `json:"available_tools"`
}