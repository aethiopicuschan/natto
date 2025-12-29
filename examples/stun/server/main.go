package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aethiopicuschan/natto/stun"
)

func main() {
	var (
		addr     = flag.String("addr", "0.0.0.0:3478", "UDP listen address")
		software = flag.String("software", "go-stun-server", "SOFTWARE attribute value")
	)
	flag.Parse()

	fmt.Println("Starting STUN server")
	fmt.Println(" Listen:", *addr)
	fmt.Println(" Software:", *software)

	// Create STUN server.
	server, err := stun.ListenUDP(*addr)
	if err != nil {
		log.Fatalf("failed to listen UDP: %v", err)
	}

	server.Software = *software
	server.ReadTimeout = 1 * time.Second

	// Graceful shutdown handling.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Run server in background.
	go func() {
		if err := server.ServeContext(ctx); err != nil {
			// ctx cancellation is expected on shutdown
			if ctx.Err() == nil {
				log.Printf("server error: %v", err)
			}
		}
	}()

	fmt.Println("STUN server is running")
	fmt.Println("Press Ctrl+C to stop")

	// Wait for signal.
	<-ctx.Done()

	fmt.Println("\nShutting down STUN server...")
	if err := server.Close(); err != nil {
		log.Printf("error during close: %v", err)
	}

	fmt.Println("Server stopped")
}
