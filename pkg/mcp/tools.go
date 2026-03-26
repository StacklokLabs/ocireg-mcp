// Package mcp provides MCP server tools for OCI registry operations.
package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/StacklokLabs/ocireg-mcp/pkg/oci"
)

// defaultTimeout is the default timeout for OCI registry operations
const defaultTimeout = 30 * time.Second

// defaultMaxBytes is the default maximum payload size for get_referrer_content (512KB)
const defaultMaxBytes = 524288

// Content format constants
const (
	formatCycloneDX      = "cyclonedx"
	formatSPDX           = "spdx"
	formatSLSA           = "slsa"
	formatOpenVEX        = "openvex"
	formatCosign         = "cosign"
	formatSigstoreBundle = "sigstore-bundle"

	contentTypeSBOM       = "sbom"
	contentTypeProvenance = "provenance"
	contentTypeVEX        = "vex"
	contentTypeSignature  = "signature"
)

// ToolNames defines the names of the tools provided by this MCP server.
const (
	GetImageInfoToolName       = "get_image_info"
	ListTagsToolName           = "list_tags"
	GetImageManifestToolName   = "get_image_manifest"
	GetImageConfigToolName     = "get_image_config"
	ListReferrersToolName      = "list_referrers"
	GetReferrerContentToolName = "get_referrer_content"
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
		mcp.NewTool(
			ListReferrersToolName,
			mcp.WithDescription(
				"List OCI artifacts (SBOMs, signatures, provenance, VEX) attached to an image via the OCI Referrers API. "+
					"Returns descriptors with artifact type, digest, size, and annotations. "+
					"Use this to discover what attestations exist before fetching their content with get_referrer_content."),
			mcp.WithString("image_ref",
				mcp.Description("The image reference (e.g., docker.io/library/alpine:latest). Tags are automatically resolved to digests."),
				mcp.Required(),
			),
			mcp.WithString("artifact_type",
				mcp.Description("Filter referrers by artifact type (e.g., application/vnd.cyclonedx+json)"),
			),
			mcp.WithOutputSchema[ListReferrersResult](),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		mcp.NewTool(
			GetReferrerContentToolName,
			mcp.WithDescription(
				"Fetch the content of a specific referrer artifact. "+
					"Use list_referrers first to discover artifacts and their digests. "+
					"Returns content as an embedded resource with proper MIME type. "+
					"For cosign attestations (DSSE envelopes), automatically decodes "+
					"the base64 payload unless decode_payload is false."),
			mcp.WithString("image_ref",
				mcp.Description(
					"The parent image reference containing the repository "+
						"(e.g., docker.io/library/alpine:latest)"),
				mcp.Required(),
			),
			mcp.WithString("digest",
				mcp.Description(
					"The digest of the referrer artifact from list_referrers "+
						"(e.g., sha256:abc123...)"),
				mcp.Required(),
			),
			mcp.WithBoolean("decode_payload",
				mcp.Description(
					"When true (default), unwrap DSSE envelopes to return "+
						"the decoded predicate. When false, return raw blob."),
			),
			mcp.WithString("content_type",
				mcp.Description("Hint about the expected content type to help label output metadata"),
				mcp.Enum("sbom", "provenance", "vex", "signature"),
			),
			mcp.WithNumber("max_bytes",
				mcp.Description("Maximum payload size in bytes. Content exceeding this is truncated. Default 512KB (524288)."),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
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

// ListReferrers handles the list_referrers tool.
func (p *ToolProvider) ListReferrers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	artifactType := mcp.ParseString(req, "artifact_type", "")

	client := p.getClient(req)

	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	indexManifest, err := client.ListReferrers(reqCtx, imageRef, artifactType)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to list referrers", err), nil
	}

	referrers := make([]ReferrerDescriptor, 0, len(indexManifest.Manifests))
	for _, desc := range indexManifest.Manifests {
		referrers = append(referrers, ReferrerDescriptor{
			MediaType:    string(desc.MediaType),
			Digest:       desc.Digest.String(),
			Size:         desc.Size,
			ArtifactType: desc.ArtifactType,
			Annotations:  desc.Annotations,
		})
	}

	result := ListReferrersResult{
		Referrers: referrers,
		Count:     len(referrers),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}

	fallback := fmt.Sprintf("Referrers for %s (%d found):\n\n```json\n%s\n```", imageRef, len(referrers), string(resultJSON))
	return mcp.NewToolResultStructured(result, fallback), nil
}

// dsseEnvelope represents a DSSE (Dead Simple Signing Envelope) structure.
type dsseEnvelope struct {
	PayloadType string          `json:"payloadType"`
	Payload     string          `json:"payload"`
	Signatures  json.RawMessage `json:"signatures"`
}

// sigstoreBundle represents the Sigstore bundle format (v0.3+).
// Attestations have a nested dsseEnvelope; signatures use messageSignature.
type sigstoreBundle struct {
	MediaType    string        `json:"mediaType"`
	DSSEEnvelope *dsseEnvelope `json:"dsseEnvelope,omitempty"`
}

// inTotoStatement represents the top-level fields of an in-toto statement.
type inTotoStatement struct {
	Type          string `json:"_type"`
	PredicateType string `json:"predicateType"`
}

// dsseDecodeResult holds the result of attempting to decode a DSSE envelope.
type dsseDecodeResult struct {
	payload       []byte
	mimeType      string
	predicateType string
	decoded       bool
}

// decodeDSSEEnvelope decodes a DSSE envelope and returns the result.
func decodeDSSEEnvelope(
	envelope *dsseEnvelope, content []byte, originalMIME string,
) dsseDecodeResult {
	decoded, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return dsseDecodeResult{payload: content, mimeType: originalMIME}
	}

	result := dsseDecodeResult{
		payload:  decoded,
		mimeType: envelope.PayloadType,
		decoded:  true,
	}

	var stmt inTotoStatement
	if err := json.Unmarshal(decoded, &stmt); err == nil {
		result.predicateType = stmt.PredicateType
	}

	return result
}

// tryDecodeDSSE attempts to unwrap a DSSE envelope from raw content.
// It handles both top-level DSSE envelopes (legacy cosign) and
// Sigstore bundles (v0.3+) where the envelope is nested in dsseEnvelope.
// If no DSSE envelope is found, returns the original content unchanged.
func tryDecodeDSSE(content []byte, originalMIME string) dsseDecodeResult {
	fallback := dsseDecodeResult{payload: content, mimeType: originalMIME}

	// Try top-level DSSE envelope (legacy cosign format)
	var envelope dsseEnvelope
	if err := json.Unmarshal(content, &envelope); err == nil {
		if envelope.PayloadType != "" && envelope.Payload != "" {
			return decodeDSSEEnvelope(
				&envelope, content, originalMIME)
		}
	}

	// Try Sigstore bundle format (v0.3+) with nested dsseEnvelope
	var bundle sigstoreBundle
	if err := json.Unmarshal(content, &bundle); err == nil {
		if bundle.DSSEEnvelope != nil &&
			bundle.DSSEEnvelope.PayloadType != "" &&
			bundle.DSSEEnvelope.Payload != "" {
			return decodeDSSEEnvelope(
				bundle.DSSEEnvelope, content, originalMIME)
		}
	}

	return fallback
}

// validContentTypes is the set of accepted content_type hint values.
var validContentTypes = map[string]bool{
	contentTypeSBOM:       true,
	contentTypeProvenance: true,
	contentTypeVEX:        true,
	contentTypeSignature:  true,
}

// GetReferrerContent handles the get_referrer_content tool.
func (p *ToolProvider) GetReferrerContent(
	ctx context.Context, req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	imageRef := mcp.ParseString(req, "image_ref", "")
	if imageRef == "" {
		return mcp.NewToolResultError("image_ref is required"), nil
	}

	digest := mcp.ParseString(req, "digest", "")
	if digest == "" {
		return mcp.NewToolResultError("digest is required"), nil
	}

	decodePayload := mcp.ParseBoolean(req, "decode_payload", true)
	contentTypeHint := mcp.ParseString(req, "content_type", "")
	maxBytes := mcp.ParseInt(req, "max_bytes", defaultMaxBytes)
	if maxBytes < 1 {
		maxBytes = 1
	}

	if contentTypeHint != "" && !validContentTypes[contentTypeHint] {
		return mcp.NewToolResultError(fmt.Sprintf(
			"invalid content_type %q: must be one of sbom, provenance, vex, signature",
			contentTypeHint,
		)), nil
	}

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return mcp.NewToolResultErrorFromErr(
			"failed to parse image reference", err), nil
	}
	repo := ref.Context().String()

	client := p.getClient(req)
	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	content, layerMediaType, err := client.GetArtifactContent(
		reqCtx, repo, digest)
	if err != nil {
		return mcp.NewToolResultErrorFromErr(
			"failed to get artifact content", err), nil
	}

	meta := ReferrerContentMetadata{
		ContentType: contentTypeHint,
		Size:        len(content),
	}

	// Apply size limit before DSSE decoding
	if len(content) > maxBytes {
		content = content[:maxBytes]
		meta.Truncated = true
	}

	payload := content
	mimeType := string(layerMediaType)

	if decodePayload {
		dr := tryDecodeDSSE(content, mimeType)
		payload = dr.payload
		mimeType = dr.mimeType
		meta.DecodedFromDSSE = dr.decoded
		meta.PredicateType = dr.predicateType
		if dr.decoded {
			meta.Size = len(payload)
		}
	}

	// Apply truncation after DSSE decoding if decoded payload exceeds max
	if len(payload) > maxBytes {
		payload = payload[:maxBytes]
		meta.Truncated = true
	}

	meta.Format, meta.ContentType = detectContentFormat(
		mimeType, meta.PredicateType, contentTypeHint)
	outputMIME := detectOutputMIMEType(mimeType, meta.Format)
	summary := buildContentSummary(repo, digest, meta)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(summary),
			mcp.NewEmbeddedResource(mcp.TextResourceContents{
				URI:      fmt.Sprintf("oci://%s@%s", repo, digest),
				MIMEType: outputMIME,
				Text:     string(payload),
			}),
		},
		StructuredContent: meta,
	}, nil
}

// detectContentFormat determines the content type and format
// from media type and predicate type.
func detectContentFormat(
	mimeType, predicateType, hint string,
) (format, contentType string) {
	// Check predicate type first (most specific)
	switch {
	case strings.Contains(predicateType, "cyclonedx.org/bom"):
		return formatCycloneDX, contentTypeSBOM
	case strings.Contains(predicateType, "spdx.dev/Document"):
		return formatSPDX, contentTypeSBOM
	case strings.Contains(predicateType, "slsa.dev/provenance"):
		return formatSLSA, contentTypeProvenance
	case strings.Contains(predicateType, "openvex.dev"):
		return formatOpenVEX, contentTypeVEX
	}

	// Check mime type
	switch {
	case strings.Contains(mimeType, "cyclonedx"):
		return formatCycloneDX, contentTypeSBOM
	case strings.Contains(mimeType, "spdx"):
		return formatSPDX, contentTypeSBOM
	case strings.Contains(mimeType, "cosign.artifact.sig"):
		return formatCosign, contentTypeSignature
	case strings.Contains(mimeType, "sigstore.bundle"):
		return formatSigstoreBundle, ""
	}

	// Fall back to hint
	if hint != "" {
		return "", hint
	}

	return "", ""
}

// detectOutputMIMEType determines the MIME type to set on the
// embedded resource.
func detectOutputMIMEType(layerMIME, format string) string {
	switch format {
	case formatCycloneDX:
		return "application/vnd.cyclonedx+json"
	case formatSPDX:
		return "application/spdx+json"
	case formatSLSA, formatOpenVEX:
		return "application/json"
	}
	if layerMIME != "" {
		return layerMIME
	}
	return "application/octet-stream"
}

// buildContentSummary creates a human-readable summary line for the content.
func buildContentSummary(repo, digest string, meta ReferrerContentMetadata) string {
	parts := []string{}
	if meta.ContentType != "" {
		parts = append(parts, strings.ToUpper(meta.ContentType))
	}
	if meta.Format != "" {
		parts = append(parts, fmt.Sprintf("format=%s", meta.Format))
	}

	label := "Artifact content"
	if len(parts) > 0 {
		label = strings.Join(parts, " ")
	}

	summary := fmt.Sprintf("%s from %s@%s (%d bytes", label, repo, digest, meta.Size)
	if meta.DecodedFromDSSE {
		summary += ", decoded from DSSE envelope"
	}
	if meta.Truncated {
		summary += ", truncated"
	}
	summary += ")"
	return summary
}
