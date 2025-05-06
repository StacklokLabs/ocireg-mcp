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
