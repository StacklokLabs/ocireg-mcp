package oci

import (
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
	_, err := client.GetImage(t.Context(), "invalid:reference:format")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing image reference")
}

func TestGetImageManifest_InvalidReference(t *testing.T) {
	client := NewClient()
	_, err := client.GetImageManifest(t.Context(), "invalid:reference:format")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing image reference")
}

func TestGetImageConfig_InvalidReference(t *testing.T) {
	client := NewClient()
	_, err := client.GetImageConfig(t.Context(), "invalid:reference:format")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing image reference")
}

func TestListTags_InvalidRepository(t *testing.T) {
	client := NewClient()
	_, err := client.ListTags(t.Context(), "invalid/repo/format")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing tags")
}

func TestListReferrers_InvalidReference(t *testing.T) {
	client := NewClient()
	_, err := client.ListReferrers(t.Context(), "invalid:reference:format", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing image reference")
}

func TestGetArtifactContent_InvalidDigest(t *testing.T) {
	client := NewClient()
	_, _, err := client.GetArtifactContent(t.Context(), "docker.io/library/alpine", "notadigest")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing artifact reference")
}

func TestGetArtifactContent_InvalidRepo(t *testing.T) {
	client := NewClient()
	_, _, err := client.GetArtifactContent(t.Context(), "INVALID", "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing artifact reference")
}

func TestWithBearerToken(t *testing.T) {
	// Test that WithBearerToken returns a valid remote.Option
	option := WithBearerToken("test-token")
	assert.NotNil(t, option)
}
