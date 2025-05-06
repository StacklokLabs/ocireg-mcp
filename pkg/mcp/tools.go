// Package mcp provides MCP server tools for OCI registry operations.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/StacklokLabs/ocireg-mcp/pkg/oci"
)

// ToolNames defines the names of the tools provided by this MCP server.
const (
	GetImageInfoToolName     = "get_image_info"
	ListTagsToolName         = "list_tags"
	GetImageManifestToolName = "get_image_manifest"
	GetImageConfigToolName   = "get_image_config"
)

// ToolProvider provides MCP tools for OCI registry operations.
type ToolProvider struct {
	client *oci.Client
}

// NewToolProvider creates a new ToolProvider.
func NewToolProvider(client *oci.Client) *ToolProvider {
	return &ToolProvider{
		client: client,
	}
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
func (p *ToolProvider) GetImageInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	img, err := p.client.GetImage(ctx, imageRef)
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
func (p *ToolProvider) ListTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repository := mcp.ParseString(req, "repository", "")
	if repository == "" {
		return mcp.NewToolResultError("repository is required"), nil
	}

	tags, err := p.client.ListTags(ctx, repository)
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
func (p *ToolProvider) GetImageManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	manifest, err := p.client.GetImageManifest(ctx, imageRef)
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
func (p *ToolProvider) GetImageConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	config, err := p.client.GetImageConfig(ctx, imageRef)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get config", err), nil
	}

	resultJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Config for %s:\n\n```json\n%s\n```", imageRef, string(resultJSON))), nil
}
