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
			mcp.WithOutputSchema[ImageInfoResult](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewTool(
			ListTagsToolName,
			mcp.WithDescription("List tags for a repository with pagination support"),
			mcp.WithString("repository",
				mcp.Description("The repository name (e.g., docker.io/library/alpine)"),
				mcp.Required(),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of tags to return per page (default: 100, max: 1000)"),
			),
			mcp.WithString("cursor",
				mcp.Description("Opaque pagination cursor from a previous list_tags response"),
			),
			mcp.WithString("sort",
				mcp.Description("Sort order for tags"),
				mcp.Enum("alphabetical", "alphabetical-desc", "semver", "semver-desc"),
			),
			mcp.WithOutputSchema[ListTagsResult](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewTool(
			GetImageManifestToolName,
			mcp.WithDescription("Get the manifest for an OCI image"),
			mcp.WithString("image_ref",
				mcp.Description("The image reference (e.g., docker.io/library/alpine:latest)"),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewTool(
			GetImageConfigToolName,
			mcp.WithDescription("Get the config for an OCI image"),
			mcp.WithString("image_ref",
				mcp.Description("The image reference (e.g., docker.io/library/alpine:latest)"),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
	}
}

// GetImageInfo handles the get_image_info tool.
func (p *ToolProvider) GetImageInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	// Get the appropriate client for this request
	client := p.getClient(req)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
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

	result := ImageInfoResult{
		Digest:       manifest.Config.Digest.String(),
		Size:         manifest.Config.Size,
		Architecture: config.Architecture,
		OS:           config.OS,
		Created:      config.Created.Format("2006-01-02T15:04:05Z07:00"),
		Layers:       len(manifest.Layers),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	fallback := fmt.Sprintf("Image information for %s:\n\n```json\n%s\n```", imageRef, string(resultJSON))
	return mcp.NewToolResultStructured(result, fallback), nil
}

// ListTags handles the list_tags tool.
func (p *ToolProvider) ListTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repository := mcp.ParseString(req, "repository", "")
	if repository == "" {
		return mcp.NewToolResultError("repository is required"), nil
	}

	// Parse and clamp limit
	limit := mcp.ParseInt(req, "limit", DefaultPageSize)
	if limit < 1 {
		limit = 1
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	// Parse sort order
	sortOrder := mcp.ParseString(req, "sort", SortAlphabetical)
	if !isValidSortOrder(sortOrder) {
		return mcp.NewToolResultError(fmt.Sprintf(
			"invalid sort order %q: must be one of alphabetical, alphabetical-desc, semver, semver-desc",
			sortOrder,
		)), nil
	}

	// Parse cursor
	var offset int
	cursorStr := mcp.ParseString(req, "cursor", "")
	if cursorStr != "" {
		cursor, err := decodeCursor(cursorStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid cursor: %v", err)), nil
		}
		if cursor.Sort != sortOrder {
			return mcp.NewToolResultError(fmt.Sprintf(
				"sort order mismatch: cursor was created with %q but request specifies %q",
				cursor.Sort, sortOrder,
			)), nil
		}
		offset = cursor.Offset
	}

	// Get the appropriate client for this request
	client := p.getClient(req)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	tags, err := client.ListTags(reqCtx, repository)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list tags", err), nil
	}

	if len(tags) == 0 {
		result := ListTagsResult{Tags: []string{}, TotalCount: 0, Sort: sortOrder}
		return mcp.NewToolResultStructured(result,
			fmt.Sprintf("No tags found for repository %s", repository)), nil
	}

	// Sort
	sorted := sortTags(tags, sortOrder)

	// Validate offset
	if offset >= len(sorted) {
		return mcp.NewToolResultError(fmt.Sprintf(
			"cursor offset %d is beyond the end of %d tags", offset, len(sorted),
		)), nil
	}

	// Paginate
	page, nextOffset := paginateTags(sorted, offset, limit)

	// Build result
	result := ListTagsResult{
		Tags:       page,
		TotalCount: len(sorted),
		Sort:       sortOrder,
	}
	if nextOffset > 0 {
		result.NextCursor = encodeCursor(nextOffset, sortOrder)
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	fallback := fmt.Sprintf("Tags for %s (showing %d of %d, sorted by %s):\n\n```json\n%s\n```",
		repository, len(page), len(sorted), sortOrder, string(resultJSON))
	return mcp.NewToolResultStructured(result, fallback), nil
}

// GetImageManifest handles the get_image_manifest tool.
func (p *ToolProvider) GetImageManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	// Get the appropriate client for this request
	client := p.getClient(req)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	manifest, err := client.GetImageManifest(reqCtx, imageRef)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get manifest", err), nil
	}

	resultJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	fallback := fmt.Sprintf("Manifest for %s:\n\n```json\n%s\n```", imageRef, string(resultJSON))
	return mcp.NewToolResultStructured(manifest, fallback), nil
}

// GetImageConfig handles the get_image_config tool.
func (p *ToolProvider) GetImageConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	// Get the appropriate client for this request
	client := p.getClient(req)

	// Create a context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	config, err := client.GetImageConfig(reqCtx, imageRef)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to get config", err), nil
	}

	resultJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	fallback := fmt.Sprintf("Config for %s:\n\n```json\n%s\n```", imageRef, string(resultJSON))
	return mcp.NewToolResultStructured(config, fallback), nil
}
