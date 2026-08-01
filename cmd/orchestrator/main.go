package main

import (
	"context"
	"bufio"
	"fmt"
	"log"
	"os"

	"ZHC-project/internal/host/config"
	"ZHC-project/internal/host/llm"
	"ZHC-project/internal/host/transport"
	"ZHC-project/internal/host/vm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config-path>\n", os.Args[0])
		os.Exit(1)
	}

	cfg, err := config.Load(os.Args[1])
	if err != nil {
		log.Fatalf("failed to load config: %w", err)
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

		messages := []llm.Message{
			{Role: "user", Content: prompt},
		}

		resp, err := provider.Chat(ctx, messages)
		if err != nil {
			log.Printf("llm error: %v", err)
			continue
		}

		if len(resp.Choices) > 0 {
			fmt.Printf("\nLLM> %s\n", resp.Choices[0].Message.Content)
		}
	}
}