package main

import (
	"flag"
	"fmt"
	"os"

	"syncflow-backend/cmd/app"
	"syncflow-backend/internal/config"
)

var Version = ""
var flagConfig = flag.String("config", "", "path to the config file (optional, uses env vars)")

func main() {
	flag.Parse()

	fmt.Printf("Loading application configuration...\n")
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load application configuration: %s\n", err)
		os.Exit(-1)
	}
	fmt.Printf("Application configuration loaded successfully.\n")

	// Initialize and start the application
	application := &app.App{}
	application.NewApplication(cfg)
	fmt.Printf("Application initialized successfully.\n")
}





