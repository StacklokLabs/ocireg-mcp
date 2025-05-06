package oci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client)
	assert.Empty(t, client.options)
}

// The following tests would typically use mocks or a test registry
// For now, we'll just test the error cases to ensure proper error handling

func TestGetImage_InvalidReference(t *testing.T) {
	client := NewClient()
	_, err := client.GetImage(context.Background(), "invalid:reference:format")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing image reference")
}

func TestGetImageManifest_InvalidReference(t *testing.T) {
	client := NewClient()
	_, err := client.GetImageManifest(context.Background(), "invalid:reference:format")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing image reference")
}

func TestGetImageConfig_InvalidReference(t *testing.T) {
	client := NewClient()
	_, err := client.GetImageConfig(context.Background(), "invalid:reference:format")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing image reference")
}

func TestListTags_InvalidRepository(t *testing.T) {
	client := NewClient()
	_, err := client.ListTags(context.Background(), "invalid/repo/format")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing tags")
}
