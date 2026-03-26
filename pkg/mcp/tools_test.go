package mcp

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StacklokLabs/ocireg-mcp/pkg/oci"
)

func TestNewToolProvider(t *testing.T) {
	client := oci.NewClient()
	provider := NewToolProvider(client)
	assert.NotNil(t, provider)
	assert.Equal(t, client, provider.client)
}

func TestGetTools(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())
	tools := provider.GetTools()

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range []string{
		GetImageInfoToolName,
		ListTagsToolName,
		GetImageManifestToolName,
		GetImageConfigToolName,
		ListReferrersToolName,
		GetReferrerContentToolName,
	} {
		assert.True(t, toolNames[expected], "expected tool %q to be present", expected)
	}
}

func TestGetTools_Annotations(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())
	tools := provider.GetTools()

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			require.NotNil(t, tool.Annotations, "tool %s should have annotations", tool.Name)
			require.NotNil(t, tool.Annotations.ReadOnlyHint, "tool %s should have ReadOnlyHint", tool.Name)
			assert.True(t, *tool.Annotations.ReadOnlyHint, "tool %s should be read-only", tool.Name)
			require.NotNil(t, tool.Annotations.DestructiveHint, "tool %s should have DestructiveHint", tool.Name)
			assert.False(t, *tool.Annotations.DestructiveHint, "tool %s should not be destructive", tool.Name)
			require.NotNil(t, tool.Annotations.OpenWorldHint, "tool %s should have OpenWorldHint", tool.Name)
			assert.True(t, *tool.Annotations.OpenWorldHint, "tool %s should be open-world", tool.Name)
		})
	}
}

func TestGetImageInfo_MissingImageRef(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	// Create a request with missing image_ref
	req := mcp.CallToolRequest{}

	result, err := provider.GetImageInfo(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "image_ref is required")
}

func TestListTags_MissingRepository(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	// Create a request with missing repository
	req := mcp.CallToolRequest{}

	result, err := provider.ListTags(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "repository is required")
}

func TestListTags_InvalidSort(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"repository": "docker.io/library/alpine",
		"sort":       "invalid-sort",
	}

	result, err := provider.ListTags(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "invalid sort order")
}

func TestListTags_InvalidCursor(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"repository": "docker.io/library/alpine",
		"cursor":     "!!!bad-cursor!!!",
	}

	result, err := provider.ListTags(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "invalid cursor")
}

func TestListTags_SortCursorMismatch(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	// Create a cursor with alphabetical sort
	cursor := encodeCursor(0, SortAlphabetical)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"repository": "docker.io/library/alpine",
		"cursor":     cursor,
		"sort":       SortSemver,
	}

	result, err := provider.ListTags(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "sort order mismatch")
}

func TestListTags_LimitClamping(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	// Test that negative limit doesn't cause an error (clamped to 1)
	// We can't fully test this without a mock, but we can verify
	// the parameter parsing doesn't error
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"repository": "docker.io/library/alpine",
		"limit":      float64(-5), // JSON numbers are float64
	}

	// This will fail at the network level, but the limit parsing should succeed
	result, err := provider.ListTags(t.Context(), req)
	require.NoError(t, err)
	// The error should be about network/registry access, not about limit validation
	if result.IsError {
		textContent, ok := mcp.AsTextContent(result.Content[0])
		assert.True(t, ok)
		assert.NotContains(t, textContent.Text, "limit")
	}
}

func TestGetImageManifest_MissingImageRef(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	// Create a request with missing image_ref
	req := mcp.CallToolRequest{}

	result, err := provider.GetImageManifest(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "image_ref is required")
}

func TestGetImageConfig_MissingImageRef(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	// Create a request with missing image_ref
	req := mcp.CallToolRequest{}

	result, err := provider.GetImageConfig(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "image_ref is required")
}

func TestListReferrers_MissingImageRef(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	req := mcp.CallToolRequest{}

	result, err := provider.ListReferrers(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "image_ref is required")
}

func TestGetReferrerContent_MissingImageRef(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	req := mcp.CallToolRequest{}

	result, err := provider.GetReferrerContent(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "image_ref is required")
}

func TestGetReferrerContent_MissingDigest(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"image_ref": "docker.io/library/alpine:latest",
	}

	result, err := provider.GetReferrerContent(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "digest is required")
}

func TestGetReferrerContent_InvalidContentType(t *testing.T) {
	provider := NewToolProvider(oci.NewClient())

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"image_ref":    "docker.io/library/alpine:latest",
		"digest":       "sha256:abc123",
		"content_type": "invalid",
	}

	result, err := provider.GetReferrerContent(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "invalid content_type")
}

func TestDetectContentFormat(t *testing.T) {
	tests := []struct {
		name            string
		mimeType        string
		predicateType   string
		hint            string
		expectedFormat  string
		expectedContent string
	}{
		{
			name:            "CycloneDX from predicate type",
			predicateType:   "https://cyclonedx.org/bom",
			expectedFormat:  "cyclonedx",
			expectedContent: "sbom",
		},
		{
			name:            "SPDX from predicate type",
			predicateType:   "https://spdx.dev/Document",
			expectedFormat:  "spdx",
			expectedContent: "sbom",
		},
		{
			name:            "SLSA provenance from predicate type",
			predicateType:   "https://slsa.dev/provenance/v1",
			expectedFormat:  "slsa",
			expectedContent: "provenance",
		},
		{
			name:            "OpenVEX from predicate type",
			predicateType:   "https://openvex.dev/ns/v0.2.0",
			expectedFormat:  "openvex",
			expectedContent: "vex",
		},
		{
			name:            "CycloneDX from mime type",
			mimeType:        "application/vnd.cyclonedx+json",
			expectedFormat:  "cyclonedx",
			expectedContent: "sbom",
		},
		{
			name:            "SPDX from mime type",
			mimeType:        "application/spdx+json",
			expectedFormat:  "spdx",
			expectedContent: "sbom",
		},
		{
			name:            "Cosign signature from mime type",
			mimeType:        "application/vnd.dev.cosign.artifact.sig.v1+json",
			expectedFormat:  "cosign",
			expectedContent: "signature",
		},
		{
			name:            "Sigstore bundle from mime type",
			mimeType:        "application/vnd.dev.sigstore.bundle.v0.3+json",
			expectedFormat:  "sigstore-bundle",
			expectedContent: "",
		},
		{
			name:            "hint fallback",
			mimeType:        "application/octet-stream",
			hint:            "sbom",
			expectedFormat:  "",
			expectedContent: "sbom",
		},
		{
			name:            "no match",
			mimeType:        "application/octet-stream",
			expectedFormat:  "",
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, contentType := detectContentFormat(tt.mimeType, tt.predicateType, tt.hint)
			assert.Equal(t, tt.expectedFormat, format)
			assert.Equal(t, tt.expectedContent, contentType)
		})
	}
}

func TestDetectOutputMIMEType(t *testing.T) {
	assert.Equal(t, "application/vnd.cyclonedx+json", detectOutputMIMEType("", "cyclonedx"))
	assert.Equal(t, "application/spdx+json", detectOutputMIMEType("", "spdx"))
	assert.Equal(t, "application/json", detectOutputMIMEType("", "slsa"))
	assert.Equal(t, "application/json", detectOutputMIMEType("", "openvex"))
	assert.Equal(t, "text/plain", detectOutputMIMEType("text/plain", ""))
	assert.Equal(t, "application/octet-stream", detectOutputMIMEType("", ""))
}

func TestTryDecodeDSSE_LegacyEnvelope(t *testing.T) {
	// Simulate a legacy cosign DSSE envelope with an in-toto SLSA provenance
	statement := `{"_type":"https://in-toto.io/Statement/v1","predicateType":"https://slsa.dev/provenance/v1","predicate":{}}`
	payload := base64.StdEncoding.EncodeToString([]byte(statement))
	envelope := fmt.Sprintf(
		`{"payloadType":"application/vnd.in-toto+json","payload":"%s","signatures":[]}`,
		payload,
	)

	result := tryDecodeDSSE([]byte(envelope), "application/octet-stream")
	assert.True(t, result.decoded)
	assert.Equal(t, "application/vnd.in-toto+json", result.mimeType)
	assert.Equal(t, "https://slsa.dev/provenance/v1", result.predicateType)
	assert.JSONEq(t, statement, string(result.payload))
}

func TestTryDecodeDSSE_SigstoreBundle(t *testing.T) {
	// Simulate a Sigstore bundle (v0.3+) with nested dsseEnvelope
	statement := `{"_type":"https://in-toto.io/Statement/v1","predicateType":"https://cyclonedx.org/bom","predicate":{}}`
	payload := base64.StdEncoding.EncodeToString([]byte(statement))
	bundle := fmt.Sprintf(`{
		"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json",
		"verificationMaterial":{"certificate":{}},
		"dsseEnvelope":{
			"payloadType":"application/vnd.in-toto+json",
			"payload":"%s",
			"signatures":[]
		}
	}`, payload)

	result := tryDecodeDSSE([]byte(bundle), "application/octet-stream")
	assert.True(t, result.decoded)
	assert.Equal(t, "application/vnd.in-toto+json", result.mimeType)
	assert.Equal(t, "https://cyclonedx.org/bom", result.predicateType)
	assert.JSONEq(t, statement, string(result.payload))
}

func TestTryDecodeDSSE_SigstoreBundleSignature(t *testing.T) {
	// A Sigstore bundle for a signature (no dsseEnvelope, has messageSignature)
	bundle := `{
		"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json",
		"verificationMaterial":{"certificate":{}},
		"messageSignature":{"messageDigest":{"algorithm":"SHA2_256","digest":"abc"}}
	}`

	result := tryDecodeDSSE(
		[]byte(bundle), "application/vnd.dev.sigstore.bundle.v0.3+json")
	assert.False(t, result.decoded)
	assert.Equal(t,
		"application/vnd.dev.sigstore.bundle.v0.3+json", result.mimeType)
}

func TestTryDecodeDSSE_NotDSSE(t *testing.T) {
	// Plain JSON that isn't a DSSE envelope or bundle
	content := `{"name":"test","version":"1.0"}`

	result := tryDecodeDSSE([]byte(content), "application/json")
	assert.False(t, result.decoded)
	assert.Equal(t, "application/json", result.mimeType)
	assert.Equal(t, content, string(result.payload))
}

func TestTryDecodeDSSE_InvalidBase64(t *testing.T) {
	// DSSE envelope with invalid base64 payload — should fall back
	envelope := `{"payloadType":"application/vnd.in-toto+json","payload":"!!!not-base64!!!","signatures":[]}`

	result := tryDecodeDSSE([]byte(envelope), "application/octet-stream")
	assert.False(t, result.decoded)
	assert.Equal(t, "application/octet-stream", result.mimeType)
}
