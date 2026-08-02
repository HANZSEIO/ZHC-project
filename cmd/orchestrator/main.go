package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"ZHC-project/internal/host/config"
	"ZHC-project/internal/host/llm"
	"ZHC-project/internal/host/transport"
	"ZHC-project/internal/host/vm"
)

const systemPrompt = `You are an autonomous security research agent operating within an isolated lab VM via a host orchestrator.

CRITICAL INSTRUCTIONS:
1. DO NOT give manual instructions, terminal commands to copy-paste, or conversational advice to the user.
2. ALL command executions MUST be performed directly on the guest VM using the TOOL_CALL syntax.
3. NEVER tell the user to run commands themselves. YOU are the one executing them.

TOOL CALL SYNTAX:
To execute a tool/command on the guest VM, you MUST output exact string format:
TOOL_CALL(tool_name, target, flags)

EXAMPLES:
- Scan ports on localhost:
  TOOL_CALL(netcat, 127.0.0.1, -zv 1-100)
- Run tcpdump interface check:
  TOOL_CALL(tcpdump, lo, -c 5)
- General command execution (if tool is exec/bash):
  TOOL_CALL(exec, 127.0.0.1, nmap -sV)

EXECUTION RULES:
- If action is requested, respond ONLY with the TOOL_CALL(...) invocation.
- When receiving TOOL_RESULT, analyze the output and issue the next TOOL_CALL(...) or provide a brief final summary.`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config-path>\n", os.Args[0])
		os.Exit(1)
	}

	cfg, err := config.Load(os.Args[1])
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	vmMgr, err := vm.NewVMManager()
	if err != nil {
		log.Fatalf("failed to create VM manager: %v", err)
	}

	if err := vmMgr.CreateOverlay(cfg.QEMU.BaseImage, cfg.QEMU.OverlayPath); err != nil {
		log.Fatalf("failed to create overlay: %v", err)
	}
	defer func() {
		if err := vmMgr.StopVM(); err != nil {
			log.Printf("failed to stop VM: %v", err)
		}
	}()

	bridge := transport.NewBridge(cfg.Transport.SocketPath)
	ctx := context.Background()
	messages := []llm.Message{{Role: "system", Content: systemPrompt}}
	if err := bridge.Connect(ctx); err != nil {
		log.Fatalf("failed to connect to guest: %v", err)
	}
	defer bridge.Close()

	if err := bridge.Initialize(ctx); err != nil {
		log.Fatalf("failed to initialize guest: %v", err)
	}

	tools, err := bridge.ListTools(ctx)
	if err != nil {
		log.Fatalf("failed to list tools: %v", err)
	}
	fmt.Printf("[Host] Connected. available tools: %d\n", len(tools))

	llmCfg := llm.Config{
		Provider: cfg.LLM.Provider,
		Ollama: llm.OllamaConfig{
			BaseURL: cfg.LLM.Ollama.BaseURL,
			Model:   cfg.LLM.Ollama.Model,
		},
		Cloud: llm.CloudConfig{
			APIKey: cfg.LLM.Cloud.APIKey,
			Model:  cfg.LLM.Cloud.Model,
		},
	}

	provider, err := llm.NewProvider(llmCfg)
	if err != nil {
		log.Fatalf("failed to init LLM: %v", err)
	}
	fmt.Println("\n[Host] ready.. type your security prompt (Ctrl+C to exit):")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nYou>")
		if !scanner.Scan() {
			break
		}
		prompt := scanner.Text()
		if prompt == "" {
			continue
		}

		messages = append(messages, llm.Message{Role: "user", Content: prompt})

		resp, err := runAgenticLoop(ctx, provider, bridge, &messages)
		if err != nil {
			log.Printf("agent err: %v", err)
			continue
		}

		if resp != nil && len(resp.Choices) > 0 {
			fmt.Printf("\nLLM> %s\n", resp.Choices[0].Message.Content)
			messages = append(messages, llm.Message{Role: "assistant", Content: resp.Choices[0].Message.Content})
		}
	}
}

func runAgenticLoop(ctx context.Context, provider llm.Provider, bridge *transport.Bridge, messages *[]llm.Message) (*llm.Response, error) {
	maxTurns := 10
	for i := 0; i < maxTurns; i++ {
		resp, err := provider.Chat(ctx, *messages)
		if err != nil {
			return nil, err
		}

		if len(resp.Choices) == 0 {
			return resp, nil
		}

		content := resp.Choices[0].Message.Content
		toolCall := parseToolCall(content)
		if toolCall == nil {
			return resp, nil
		}

		fmt.Printf("[Host] Executing: %s %s %s\n", toolCall.Tool, toolCall.Target, toolCall.Flags)
		result, err := bridge.ExecuteTool(ctx, toolCall.Tool, toolCall.Target, toolCall.Flags)
		if err != nil {
			return nil, err
		}

		resultText := "Tool execution failed"
		if !result.IsError && len(result.Content) > 0 {
			resultText = result.Content[0].Content
		}

		*messages = append(*messages, llm.Message{Role: "assistant", Content: content})
		*messages = append(*messages, llm.Message{Role: "user", Content: "TOOL_RESULT:\n" + resultText})
	}
	return nil, fmt.Errorf("max agentic turns reached")
}

type toolCall struct {
	Tool   string
	Target string
	Flags  string
}

func parseToolCall(s string) *toolCall {
	if !strings.Contains(s, "TOOL_CALL(") {
		return nil
	}

	start := strings.Index(s, "TOOL_CALL(")
	end := strings.Index(s[start:], ")")
	if end == -1 {
		return nil
	}

	inner := s[start+len("TOOL_CALL(") : start+end]
	parts := strings.SplitN(inner, ",", 3)
	if len(parts) < 2 {
		return nil
	}

	tc := &toolCall{
		Tool:   strings.TrimSpace(parts[0]),
		Target: strings.TrimSpace(parts[1]),
	}
	if len(parts) >= 3 {
		tc.Flags = strings.TrimSpace(parts[2])
	}
	return tc
}
