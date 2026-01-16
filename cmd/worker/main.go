package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("K-ERP Worker starting...")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Worker is running. Press Ctrl+C to stop.")

	// TODO: Initialize NATS consumer
	// TODO: Process background jobs

	<-sigChan
	log.Println("Worker shutting down...")
}
