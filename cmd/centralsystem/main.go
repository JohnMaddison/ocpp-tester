package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	httpserver "github.com/johnmaddison/ocpp-tester/internal/http-server"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/data"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/service"
	ocppserver "github.com/johnmaddison/ocpp-tester/internal/ocpp-server"
)

func main() {
	log.Print("Starting...")

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

	sessionStore := data.NewSessionStore()
	ocppLogStore, err := data.OpenOCPPLogStore(ctx, data.DefaultOCPPLogDBPath)
	if err != nil {
		log.Fatalf("Failed to open OCPP log database: %v", err)
	}
	defer func() {
		if err := ocppLogStore.Close(); err != nil {
			log.Printf("Failed to close OCPP log database: %v", err)
		}
	}()

	sessionsService := service.NewSessionsService(sessionStore)
	ocppLogService := service.NewOCPPLogService(ocppLogStore)
	ocppServer := ocppserver.NewOCPPServer(sessionStore, ocppLogStore)
	ocpp16Service := service.NewOCPP16Service(ocppServer)

	go ocppserver.StartOCPPServer(ctx, ocppServer)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		httpserver.StartHTTPServer(ctx, sessionsService, ocpp16Service, ocppLogService)
	}()

	wg.Wait()
	log.Print("Shutdown complete")

}
