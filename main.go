// Package main provides the entry point for the PiPiMink application.
// PiPiMink is a service that intelligently routes user requests to the most
// appropriate AI language model based on the request's requirements.
package main

import (
	"log"
	"os"

	"PiPiMink/cmd/server"
	_ "PiPiMink/docs" // Import generated Swagger docs
	"PiPiMink/internal/config"
)

// @title PiPiMink API
// @version 1.0
// @description An intelligent router for AI language model requests
// @contact.name API Support
// @contact.url https://github.com/Izzetee/PiPiMink
// @license.name Apache-2.0
// @license.url https://www.apache.org/licenses/LICENSE-2.0
// @host localhost:8080
// @BasePath /

func main() {
	// Initialize logging with timestamp and file information
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
	log.Println("Starting PiPiMink...")

	// Load configuration from environment variables or .env file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded successfully. Using port: %s", cfg.Port)

	// Start the HTTP server with the loaded configuration
	if err := server.Start(cfg); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
