package main

import (
	"context"
	_ "embed"
	"fmt"
	"ikea-dirigera-exporter/internal/dirigera"
	"ikea-dirigera-exporter/internal/webserver"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	dirigeraClient dirigera.Client
	webServer      webserver.Server

	version = "dev"

	//go:embed assets/ascii.art
	asciiArt string
)

func main() {
	// Startup function
	fmt.Printf("%s\n\n", fmt.Sprintf(asciiArt, version))
	err := startup()
	if err != nil {
		log.Fatalf("Error during startup: %v\n", err)
	}

	// Notification context for reacting on process termination - used by shutdown function
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Waiting group used to await finishing the shutdown process when stopping
	var wait sync.WaitGroup

	// Loop function for webserver
	wait.Add(1)
	go func() {
		defer wait.Done()
		fmt.Printf("Web server started\n")
		_ = webServer.Start()
	}()

	// Loop function for event listening
	wait.Add(1)
	go func() {
		defer wait.Done()
		fmt.Printf("IKEA dirigera client started\n")
		_ = dirigeraClient.Start()
	}()

	// Shutdown function waiting for the SIGTERM notification to stop event listening
	wait.Add(1)
	go func() {
		defer wait.Done()
		<-ctx.Done()
		fmt.Printf("\n\U0001F6D1 Shutdown down started...\n")
		shutdown()
	}()

	wait.Wait()
	fmt.Printf("\U0001F3C1 Shutdown finished\n")
	os.Exit(0)
}

func startup() error {
	var err error
	time.Local, err = time.LoadLocation("CET")
	if err != nil {
		return fmt.Errorf("error pinning location: %w", err)
	}
	fmt.Printf("Timezone CET loaded\n")

	webServer = webserver.NewServer(healthCheck)
	fmt.Printf("Web server created\n")

	dirigeraClient, err = dirigera.NewClient()
	if err != nil {
		return fmt.Errorf("error creationg IKEA dirigera client: %w", err)
	}
	fmt.Printf("IKEA dirigera client created for hub %s\n", dirigeraClient.GetHubName())

	return nil
}

func shutdown() {
	err := dirigeraClient.Shutdown()
	if err != nil {
		fmt.Printf("Error stopping event listening: %v\n", err)
	} else {
		fmt.Printf("Event listening stopped\n")
	}

	err = webServer.Shutdown()
	if err != nil {
		fmt.Printf("Error stopping web server: %v\n", err)
	} else {
		fmt.Printf("Web server stopped\n")
	}
}

func healthCheck() map[string]error {
	errors := make(map[string]error)
	if err := dirigeraClient.Health(); err != nil {
		errors["IKEA DIRIGERA Client"] = err
	}
	return errors
}
