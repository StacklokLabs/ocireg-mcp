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

// setupContextWithGracefulShutdown creates a cancellable context and configures signal handling
// for graceful shutdown on SIGINT and SIGTERM signals
func setupContextWithGracefulShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()
	}()

	return ctx, cancel
}

// createOCIClient creates an OCI client with appropriate authentication
func createOCIClient() *oci.Client {
	var ociClientOptions []remote.Option

	// Check for authentication method based on available environment variables
	token := os.Getenv("OCI_TOKEN")
	username := os.Getenv("OCI_USERNAME")
	password := os.Getenv("OCI_PASSWORD")

	switch {
	case token != "":
		log.Println("Using bearer token authentication for OCI registry")
		ociClientOptions = append(ociClientOptions, oci.WithBearerToken(token))
	case username != "" && password != "":
		log.Println("Using username/password authentication for OCI registry")
		ociClientOptions = append(ociClientOptions, oci.WithBasicAuth(username, password))
	default:
		// If no explicit credentials, use the default keychain
		// This will use credentials from the Docker config file
		log.Println("Using default keychain for OCI registry authentication")
		ociClientOptions = append(ociClientOptions, oci.WithDefaultKeychain())
	}

	return oci.NewClient(ociClientOptions...)
}

// setupServer creates and configures the MCP server with tools
func setupServer(ociClient *oci.Client, serverName, serverVersion string) *mcpserver.SSEServer {
	// Create the tool provider
	toolProvider := mcp.NewToolProvider(ociClient)

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
	return mcpserver.NewSSEServer(server)
}

// startServer starts the server and returns a channel for errors
func startServer(sseServer *mcpserver.SSEServer, port int, serverName, serverVersion string) chan error {
	serverErrCh := make(chan error, 1)

	go func() {
		addr := fmt.Sprintf(":%d", port)
		log.Printf("Starting %s v%s on %s", serverName, serverVersion, addr)
		if err := sseServer.Start(addr); err != nil {
			log.Printf("Server error: %v", err)
			serverErrCh <- err
		}
	}()

	return serverErrCh
}

// shutdownServer attempts to gracefully shut down the server
func shutdownServer(sseServer *mcpserver.SSEServer) {
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
}

func main() {
	// Parse command-line flags
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	// Setup context with signal handling for graceful shutdown
	ctx, cancel := setupContextWithGracefulShutdown()
	defer cancel()

	// Create the OCI client
	ociClient := createOCIClient()

	// Server configuration
	serverName := "ocireg-mcp"
	serverVersion := version

	// Setup the server
	sseServer := setupServer(ociClient, serverName, serverVersion)

	// Start the server
	serverErrCh := startServer(sseServer, *port, serverName, serverVersion)

	// Wait for either a server error or a shutdown signal
	select {
	case err := <-serverErrCh:
		log.Fatalf("Server failed to start: %v", err)
	case <-ctx.Done():
		log.Println("Shutting down server...")
		shutdownServer(sseServer)
	}
}
