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

	// Create the OCI client
	ociClient := oci.NewClient()

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

	// Start the server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%d", *port)
		log.Printf("Starting %s v%s on %s", serverName, serverVersion, addr)
		if err := sseServer.Start(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a context with a timeout for the graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to gracefully shut down the server
	if err := sseServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}
