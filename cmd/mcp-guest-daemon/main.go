package main

import (
	"context"
	"fmt"
	"net"
	"os"
	osexec "os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	guestexec "ZHC-project/internal/guest/exec"
	"ZHC-project/internal/guest/osdetect"
	"ZHC-project/internal/guest/shared"
)

func main() {
	sysInfo, err := osdetect.DetectSystemInfo()
	if err != nil {
		fmt.Printf("[GUEST Daemon] Warn: failed to detect OS info: %v\n", err)
	} else {
		fmt.Printf("[GUEST Daemon] Running on: %s %s (Pentest Distro: %v)\n",
			sysInfo.OSName, sysInfo.OSVersion, sysInfo.IsPentestDistro)
		fmt.Printf("[GUEST Daemon] Available tools: %v\n", sysInfo.AvailableTools)
	}

	s := server.NewMCPServer("mcp-guest-daemon", "1.0.0")

	runCliTool := mcp.NewTool("execute_cli_tool",
		mcp.WithDescription("Execute a CLI security tool safely with arguments and target inside the isolated VM"),
		mcp.WithString("tool", mcp.Required(), mcp.Description("Tool binary name (e.g., 'nmap', 'ffuf', 'gobuster')")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target IP / Hostname / URL")),
		mcp.WithString("flags", mcp.Description("Additional CLI flags (e.g., '-sV -p 80,443')")),
	)

	s.AddTool(runCliTool, handleExecuteCLITool)

	socketPath := "/tmp/mcp-guest-agent.sock"
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Printf("[GUEST Daemon] Failed to listen on unix socket: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()
	_ = os.Chmod(socketPath, 0777)

	fmt.Printf("[GUEST Daemon] Listening on %s\n", socketPath)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n[GUEST Daemon] Shutting down...")
		_ = listener.Close()
		_ = os.Remove(socketPath)
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("[GUEST Daemon] Accept error: %v\n", err)
			continue
		}
		go handleConnection(conn, s)
	}
}

func handleConnection(conn net.Conn, s *server.MCPServer) {
	defer conn.Close()
	fmt.Printf("[GUEST Daemon] Client connected\n")

	stdioServer := server.NewStdioServer(s)
	ctx := context.Background()
	if err := stdioServer.Listen(ctx, conn, conn); err != nil {
		fmt.Printf("[GUEST Daemon] Connection closed with error: %v\n", err)
	}
}

func handleExecuteCLITool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Invalid arguments format"), nil
	}
	
	toolName, _ := arguments["tool"].(string)
	target, _ := arguments["target"].(string)
	flags, _ := arguments["flags"].(string)

	if toolName == "" {
		return mcp.NewToolResultError("Missing required parameter 'tool'"), nil
	}

	if _, err := osexec.LookPath(toolName); err != nil {
		sysInfo, _ := osdetect.DetectSystemInfo()
		var avail []string
		if sysInfo != nil {
			avail = sysInfo.AvailableTools
		}
		return mcp.NewToolResultError(fmt.Sprintf(
			"Tool '%s' is not installed in Guest VM. Available tools: %v",
			toolName, avail,
		)), nil
	}

	var args []string
	if flags != "" {
		args = strings.Fields(flags)
	}
	switch toolName {
	case "nmap", "masscan":
		if target != "" {
			args = append(args, target)
		}
	case "sqlmap":
		if target != "" {
			args = append([]string{"-u", target}, args...)
		}
	case "ffuf", "gobuster", "dirb", "dirbuster":
		if target != "" {
			args = append([]string{"-u", target}, args...)
		}
	case "nikto":
		if target != "" {
			args = append([]string{"-h", target}, args...)
		}
	default:
		if target != "" {
			args = append(args, target)
		}
	}

	result := guestexec.ExecuteTool(shared.ExecRequest{
		ToolName:   toolName,
		Args:       args,
		TimeoutSec: 300,
	})

	if !result.Success {
		return mcp.NewToolResultError(fmt.Sprintf(
			"Tool execution failed for '%s': %s\nStdout:\n%s\nStderr:\n%s",
			toolName, result.ErrorMsg, result.Stdout, result.Stderr,
		)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("=== Output of %s ===\n%s", toolName, result.Stdout)), nil
}