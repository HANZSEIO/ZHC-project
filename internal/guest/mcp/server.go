package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"ZHC-project/internal/guest/shared"
)

type GuestServer struct {
	socketPath string
}

func NewGuestServer(socketPath string) *GuestServer {
	return &GuestServer{socketPath: socketPath}
}

func (g *GuestServer) Start() error {
	return nil
}

func (g *GuestServer) Stop() error {
	return nil
}

type StdioBridge struct {
	reader io.Reader
	writer io.Writer
}

func NewStdioBridge(r io.Reader, w io.Writer) *StdioBridge {
	return &StdioBridge{reader: r, writer: w}
}

func (b *StdioBridge) Serve(s *server.MCPServer) error {
	stdioSrv := server.NewStdioServer(s)
	ctx := context.Background()
	return stdioSrv.Listen(ctx, b.reader, b.writer)
}

func BuildToolResult(result shared.ExecResult) *mcp.CallToolResult {
	if result.Success {
		return mcp.NewToolResultText(fmt.Sprintf("=== Output ===\n%s", result.Stdout))
	}
	return mcp.NewToolResultError(fmt.Sprintf("Error:  %s\nOutput:\n%s", result.ErrorMsg, result.Stdout))
}

func MashalToolparams(params string, args []string) map[string]interface{} {
	arguments := map[string]interface{}{
		"tool":   params,
		"target": "",
		"flags":  "",
	}
	if len(args) > 0 {
		arguments["target"] = args[len(args)-1]
		if len(args) > 1 {
			arguments["flags"] = joinArgs(args[:len(args)-1])
		}
	}
	return arguments
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}

func UnmarshalToolResult(resultBytes []byte) (*shared.ToolCallResult, error) {
	var result shared.ToolCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

