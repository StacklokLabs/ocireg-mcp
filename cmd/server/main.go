// Package main provides the entry point for the OCI registry MCP server.
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

	"github.com/google/go-containerregistry/pkg/v1/remote"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/StacklokLabs/ocireg-mcp/pkg/mcp"
	"github.com/StacklokLabs/ocireg-mcp/pkg/oci"
)

var (
	// version is set during build using ldflags
	version = "dev"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Create the OCI client with authentication options
	var ociClientOptions []remote.Option

	// Check for username/password authentication
	username := os.Getenv("OCI_USERNAME")
	password := os.Getenv("OCI_PASSWORD")
	if username != "" && password != "" {
		log.Println("Using username/password authentication for OCI registry")
		ociClientOptions = append(ociClientOptions, oci.WithBasicAuth(username, password))
	} else {
		// If no explicit credentials, use the default keychain
		// This will use credentials from the Docker config file
		log.Println("Using default keychain for OCI registry authentication")
		ociClientOptions = append(ociClientOptions, oci.WithDefaultKeychain())
	}

	ociClient := oci.NewClient(ociClientOptions...)

	// Create the tool provider
	toolProvider := mcp.NewToolProvider(ociClient)

	serverVersion := version
	serverName := "ocireg-mcp"

	// Create the MCP server
	server := mcpserver.NewMCPServer(serverName, serverVersion)

	// Add the tools to the server
	for _, tool := range toolProvider.GetTools() {
		switch tool.Name {
		case mcp.GetImageInfoToolName:
			server.AddTool(tool, toolProvider.GetImageInfo)
		case mcp.ListTagsToolName:
			server.AddTool(tool, toolProvider.ListTags)
		case mcp.GetImageManifestToolName:
			server.AddTool(tool, toolProvider.GetImageManifest)
		case mcp.GetImageConfigToolName:
			server.AddTool(tool, toolProvider.GetImageConfig)
		}
	}

	// Create an SSE server
	sseServer := mcpserver.NewSSEServer(server)

	// Channel to receive server errors
	serverErrCh := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%d", *port)
		log.Printf("Starting %s v%s on %s", serverName, serverVersion, addr)
		if err := sseServer.Start(addr); err != nil {
			log.Printf("Server error: %v", err)
			serverErrCh <- err
		}
	}()

	// Wait for either a server error or a shutdown signal
	select {
	case err := <-serverErrCh:
		log.Fatalf("Server failed to start: %v", err)
	case <-ctx.Done():
		log.Println("Shutting down server...")
	}

	// Create a context with a timeout for the graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Attempt to shut down the server gracefully
	shutdownCh := make(chan error, 1)
	go func() {
		log.Println("Initiating server shutdown...")
		err := sseServer.Shutdown(shutdownCtx)
		if err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		shutdownCh <- err
		close(shutdownCh)
	}()

	// Wait for shutdown to complete or timeout
	select {
	case err, ok := <-shutdownCh:
		if ok {
			if err != nil {
				log.Printf("Server shutdown error: %v", err)
			} else {
				log.Println("Server shutdown completed gracefully")
			}
		}
	case <-shutdownCtx.Done():
		log.Println("Server shutdown timed out, forcing exit...")
		os.Exit(1)
	}

	log.Println("Server shutdown complete, exiting...")
	os.Exit(0)
}
