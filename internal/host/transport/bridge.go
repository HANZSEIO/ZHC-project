package transport

import (
	"bufio"
	"net"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ZHC-project/internal/guest/shared"
)

type ToolResult struct {
	Success bool
	Stdout string
	Stderr string
	Error string
}
 type Bridge struct {
	socketPath string
	conn net.Conn
	reader *bufio.Reader
	mu sync.Mutex
	nextID int
 }

 func NewBridge(socketPath string) *Bridge {
	return &Bridge{socketPath: socketPath}
 }

 func (b *Bridge) Connect(ctx context.Context) error {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "unix", b.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to unix socket: %w", err)
	}

	b.conn = conn
	b.reader = bufio.NewReader(conn)
	return nil
 }

 func (b *Bridge) Close() error {
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
 }

 func (b *Bridge) Call(ctx context.Context, method string, params interface{}) (*shared.JSONResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	req := shared.JSONRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      b.nextID,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	if _, err := fmt.Fprintf(b.conn, "%s\n", data); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	line, err := b.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var resp shared.JSONResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return &resp, nil
}

func (b *Bridge) Initialize(ctx context.Context) error {
	_, err := b.Call(ctx, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]bool{},
		"clientInfo": map[string]string{
			"name":    "zhc-orchestrator",
			"version": "1.0.0",
		},
	})
	return err
}

func (b *Bridge) ListTools(ctx context.Context) ([]map[string]interface{}, error) {
	resp, err := b.Call(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Tools []map[string]interface{} `json:"tools"`
	}

	if err := json.Unmarshal(resultBytes, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Tools, nil

}
func (b *Bridge) ExecuteTool(ctx context.Context, toolName, target, flags string) (*shared.ToolCallResult, error) {
	arguments := map[string]interface{}{
		"tool": toolName,
		"target": target,
		"flags": flags,
	}
	resp, err := b.Call(ctx, "tools/call", map[string]interface{}{
		"name": toolName,
		"arguments": arguments,
	})

	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, err
	}

	var result shared.ToolCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, err
	}

	return &result, nil
}