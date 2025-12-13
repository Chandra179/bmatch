package main

import (
	"context"
	"log"

	"bmatch/cfg"
	"bmatch/internal/app"
)

func main() {
	config, err := cfg.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	server, err := app.NewServer(ctx, config)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}
	defer server.Shutdown(ctx)

	if err := server.Run(":8080"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
