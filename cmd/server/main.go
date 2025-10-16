// Package main provides the entry point for the OCI registry MCP server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

// createOCIClientFromHeaders creates an OCI client using authentication from HTTP headers
// Priority: Authorization header > OCI_TOKEN env > OCI_USERNAME/PASSWORD env > default keychain
func createOCIClientFromHeaders(headers http.Header) *oci.Client {
	var ociClientOptions []remote.Option

	// Priority 1: Check for bearer token from HTTP Authorization header (highest priority)
	authHeader := headers.Get("Authorization")
	if authHeader != "" {
		const bearerPrefix = "Bearer "
		if strings.HasPrefix(authHeader, bearerPrefix) {
			token := strings.TrimPrefix(authHeader, bearerPrefix)
			log.Println("Using bearer token from Authorization header for OCI registry")
			ociClientOptions = append(ociClientOptions, oci.WithBearerToken(token))
			return oci.NewClient(ociClientOptions...)
		}
	}

	// Priority 2: Check for authentication from environment variables
	token := os.Getenv("OCI_TOKEN")
	username := os.Getenv("OCI_USERNAME")
	password := os.Getenv("OCI_PASSWORD")

	switch {
	case token != "":
		log.Println("Using bearer token from OCI_TOKEN environment variable for OCI registry")
		ociClientOptions = append(ociClientOptions, oci.WithBearerToken(token))
	case username != "" && password != "":
		log.Println("Using username/password authentication for OCI registry")
		ociClientOptions = append(ociClientOptions, oci.WithBasicAuth(username, password))
	default:
		// Priority 3: If no explicit credentials, use the default keychain
		// This will use credentials from the Docker config file
		log.Println("Using default keychain for OCI registry authentication")
		ociClientOptions = append(ociClientOptions, oci.WithDefaultKeychain())
	}

	return oci.NewClient(ociClientOptions...)
}

// setupServer creates and configures the MCP server with tools
func setupServer(serverName, serverVersion string) *mcpserver.SSEServer {
	// Create the tool provider with a factory that creates clients per-request
	toolProvider := mcp.NewToolProviderWithFactory(createOCIClientFromHeaders)

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

// validatePort checks if the given port number is valid (between 0 and 65535).
// Returns true if valid, false otherwise.
func validatePort(port int) bool {
	return port >= 0 && port <= 65535
}

// getMCPServerPort returns the port number from MCP_PORT environment variable.
// If the environment variable is not set or contains an invalid value,
// it returns the default port 8080.
func getMCPServerPort() int {
	const defaultPort = 8080

	envPort := os.Getenv("MCP_PORT")
	if envPort == "" {
		return defaultPort
	}

	port, err := strconv.Atoi(envPort)
	if err != nil {
		log.Printf("Invalid MCP_PORT value: %s (must be a valid number), using default port 8080", envPort)
		return defaultPort
	}

	if !validatePort(port) {
		log.Printf("Invalid MCP_PORT value: %s (must be between 0 and 65535), using default port 8080", envPort)
		return defaultPort
	}

	return port
}

func main() {
	// Get port from environment variable or use default
	envPort := getMCPServerPort()

	// Parse command-line flags
	port := flag.Int("port", envPort, "Port to listen on (must be between 0 and 65535)")
	flag.Parse()

	// Validate command-line port
	if !validatePort(*port) {
		log.Printf("Invalid port number: %d (must be between 0 and 65535), using default port 8080", *port)
		*port = 8080
	}

	// Setup context with signal handling for graceful shutdown
	ctx, cancel := setupContextWithGracefulShutdown()
	defer cancel()

	// Server configuration
	serverName := "ocireg-mcp"
	serverVersion := version

	// Setup the server
	sseServer := setupServer(serverName, serverVersion)

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
