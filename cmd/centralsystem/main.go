package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	httpserver "github.com/JohnMaddison/ocpp-tester/internal/http-server"
	ocppserver "github.com/JohnMaddison/ocpp-tester/internal/ocpp-server"
	"github.com/JohnMaddison/ocpp-tester/internal/ocppclients"
)

func main() {
	log.Print("Starting...")

	ocppclients.Init()

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Print("Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	go ocppserver.StartOCPPServer(ctx)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		httpserver.StartHTTPServer(ctx)
	}()

	wg.Wait()
	log.Print("Shutdown complete")

}
