package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/steipete/wacli/internal/api"
	"github.com/steipete/wacli/internal/service"
	"github.com/steipete/wacli/internal/webhook"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[Main] Starting WhatsApp API Service...")

	// Load configuration from environment
	cfg := service.LoadFromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("[Main] Invalid configuration: %v", err)
	}

	log.Printf("[Main] Data directory: %s", cfg.DataDir)
	log.Printf("[Main] Listen address: %s", cfg.Addr())

	// Create service manager
	mgr, err := service.NewManager(cfg)
	if err != nil {
		log.Fatalf("[Main] Failed to create manager: %v", err)
	}

	// Create webhook emitter if configured
	var webhookEmitter *webhook.Emitter
	if cfg.WebhookURL != "" {
		log.Printf("[Main] Webhook URL: %s", cfg.WebhookURL)
		webhookEmitter = webhook.NewEmitter(webhook.Config{
			URL:        cfg.WebhookURL,
			Secret:     cfg.WebhookSecret,
			MaxRetries: cfg.WebhookRetries,
			Timeout:    cfg.WebhookTimeout,
		})
		webhookEmitter.Start()

		// Register message handler for webhooks
		mgr.OnMessage(func(msg *service.ReceivedMessage) {
			webhookEmitter.Emit("message.received", msg)
		})
	}

	// Create HTTP API server
	server := api.NewServer(cfg, mgr)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start service manager
	if err := mgr.Start(ctx); err != nil {
		log.Fatalf("[Main] Failed to start manager: %v", err)
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("[Main] Server error: %v", err)
			sigChan <- syscall.SIGTERM
		}
	}()

	log.Println("[Main] Service started successfully")

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("[Main] Received signal %v, shutting down...", sig)

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Main] Server shutdown error: %v", err)
	}

	// Stop webhook emitter
	if webhookEmitter != nil {
		webhookEmitter.Stop()
	}

	// Stop service manager
	if err := mgr.Stop(); err != nil {
		log.Printf("[Main] Manager stop error: %v", err)
	}

	log.Println("[Main] Shutdown complete")
}
