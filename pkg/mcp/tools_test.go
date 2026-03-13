package mcp

import (
	"context"
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

	assert.Len(t, tools, 4)

	// Check that all expected tools are present
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames[GetImageInfoToolName])
	assert.True(t, toolNames[ListTagsToolName])
	assert.True(t, toolNames[GetImageManifestToolName])
	assert.True(t, toolNames[GetImageConfigToolName])
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

	result, err := provider.GetImageInfo(context.Background(), req)
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

	result, err := provider.ListTags(context.Background(), req)
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

	result, err := provider.ListTags(context.Background(), req)
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

	result, err := provider.ListTags(context.Background(), req)
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

	result, err := provider.ListTags(context.Background(), req)
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
	result, err := provider.ListTags(context.Background(), req)
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

	result, err := provider.GetImageManifest(context.Background(), req)
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

	result, err := provider.GetImageConfig(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content)

	textContent, ok := mcp.AsTextContent(result.Content[0])
	assert.True(t, ok)
	assert.Contains(t, textContent.Text, "image_ref is required")
}
