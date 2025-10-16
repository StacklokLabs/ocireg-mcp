// Package mcp provides MCP server tools for OCI registry operations.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/StacklokLabs/ocireg-mcp/pkg/oci"
)

// defaultTimeout is the default timeout for OCI registry operations
const defaultTimeout = 30 * time.Second

// ToolNames defines the names of the tools provided by this MCP server.
const (
	GetImageInfoToolName     = "get_image_info"
	ListTagsToolName         = "list_tags"
	GetImageManifestToolName = "get_image_manifest"
	GetImageConfigToolName   = "get_image_config"
)

// ClientFactory is a function that creates an OCI client from HTTP headers
type ClientFactory func(http.Header) *oci.Client

// ToolProvider provides MCP tools for OCI registry operations.
type ToolProvider struct {
	client        *oci.Client
	clientFactory ClientFactory
}

// NewToolProvider creates a new ToolProvider.
func NewToolProvider(client *oci.Client) *ToolProvider {
	return &ToolProvider{
		client: client,
	}
}

// NewToolProviderWithFactory creates a new ToolProvider with a custom client factory.
// The factory will be used to create clients per-request based on HTTP headers.
func NewToolProviderWithFactory(clientFactory ClientFactory) *ToolProvider {
	return &ToolProvider{
		clientFactory: clientFactory,
	}
}

// getClient returns the appropriate OCI client for the request.
// If a client factory is configured, it creates a new client from the request headers.
// Otherwise, it uses the default client.
func (p *ToolProvider) getClient(req mcp.CallToolRequest) *oci.Client {
	if p.clientFactory != nil {
		return p.clientFactory(req.Header)
	}
	return p.client
}

// GetTools returns the list of tools provided by this MCP server.
func (*ToolProvider) GetTools() []mcp.Tool {
	return []mcp.Tool{
		mcp.NewTool(
			GetImageInfoToolName,
			mcp.WithDescription("Get information about an OCI image"),
			mcp.WithString("image_ref",
				mcp.Description("The image reference (e.g., docker.io/library/alpine:latest)"),
				mcp.Required(),
			),
		),
		mcp.NewTool(
			ListTagsToolName,
			mcp.WithDescription("List tags for a repository"),
			mcp.WithString("repository",
				mcp.Description("The repository name (e.g., docker.io/library/alpine)"),
				mcp.Required(),
			),
		),
		mcp.NewTool(
			GetImageManifestToolName,
			mcp.WithDescription("Get the manifest for an OCI image"),
			mcp.WithString("image_ref",
				mcp.Description("The image reference (e.g., docker.io/library/alpine:latest)"),
				mcp.Required(),
			),
		),
		mcp.NewTool(
			GetImageConfigToolName,
			mcp.WithDescription("Get the config for an OCI image"),
			mcp.WithString("image_ref",
				mcp.Description("The image reference (e.g., docker.io/library/alpine:latest)"),
				mcp.Required(),
			),
		),
	}
}

// GetImageInfo handles the get_image_info tool.
func (p *ToolProvider) GetImageInfo(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	// Get the appropriate client for this request
	client := p.getClient(req)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	img, err := client.GetImage(reqCtx, imageRef)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get image", err), nil
	}

	manifest, err := img.Manifest()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get manifest", err), nil
	}

	config, err := img.ConfigFile()
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get config", err), nil
	}

	// Prepare the result
	result := map[string]interface{}{
		"digest":       manifest.Config.Digest.String(),
		"size":         manifest.Config.Size,
		"architecture": config.Architecture,
		"os":           config.OS,
		"created":      config.Created.Format("2006-01-02T15:04:05Z07:00"),
		"layers":       len(manifest.Layers),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Image information for %s:\n\n```json\n%s\n```", imageRef, string(resultJSON))), nil
}

// ListTags handles the list_tags tool.
func (p *ToolProvider) ListTags(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repository := mcp.ParseString(req, "repository", "")
	if repository == "" {
		return mcp.NewToolResultError("repository is required"), nil
	}

	// Get the appropriate client for this request
	client := p.getClient(req)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tags, err := client.ListTags(reqCtx, repository)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list tags", err), nil
	}

	if len(tags) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No tags found for repository %s", repository)), nil
	}

	resultJSON, err := json.MarshalIndent(tags, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Tags for %s:\n\n```json\n%s\n```", repository, string(resultJSON))), nil
}

// GetImageManifest handles the get_image_manifest tool.
func (p *ToolProvider) GetImageManifest(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	// Get the appropriate client for this request
	client := p.getClient(req)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	manifest, err := client.GetImageManifest(reqCtx, imageRef)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get manifest", err), nil
	}

	resultJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Manifest for %s:\n\n```json\n%s\n```", imageRef, string(resultJSON))), nil
}

// GetImageConfig handles the get_image_config tool.
func (p *ToolProvider) GetImageConfig(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	// Get the appropriate client for this request
	client := p.getClient(req)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	config, err := client.GetImageConfig(reqCtx, imageRef)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get config", err), nil
	}

	resultJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Config for %s:\n\n```json\n%s\n```", imageRef, string(resultJSON))), nil
}
