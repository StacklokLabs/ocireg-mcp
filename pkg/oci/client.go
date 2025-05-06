// Package oci provides functionality for interacting with OCI registries.
package oci

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var (
	defaultTimeout = 30 * time.Second
)

// Client provides methods for interacting with OCI registries.
type Client struct {
	options []remote.Option
}

// NewClient creates a new OCI registry client.
func NewClient(options ...remote.Option) *Client {
	return &Client{
		options: options,
	}
}

// GetImage retrieves an image from a registry.
func (c *Client) GetImage(ctx context.Context, imageRef string) (v1.Image, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	options := append(c.options, remote.WithContext(reqCtx))
	img, err := remote.Image(ref, options...)
	if err != nil {
		return nil, fmt.Errorf("fetching image: %w", err)
	}

	return img, nil
}

// GetImageManifest retrieves the manifest for an image.
func (c *Client) GetImageManifest(ctx context.Context, imageRef string) (*v1.Manifest, error) {
	reqCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	img, err := c.GetImage(reqCtx, imageRef)
	if err != nil {
		return nil, err
	}

	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("getting manifest: %w", err)
	}

	return manifest, nil
}

// GetImageConfig retrieves the config for an image.
func (c *Client) GetImageConfig(ctx context.Context, imageRef string) (*v1.ConfigFile, error) {
	reqCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	img, err := c.GetImage(reqCtx, imageRef)
	if err != nil {
		return nil, err
	}

	config, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("getting config: %w", err)
	}

	return config, nil
}

// ListTags lists all tags for a repository.
func (c *Client) ListTags(ctx context.Context, repoName string) ([]string, error) {
	repo, err := name.NewRepository(repoName)
	if err != nil {
		return nil, fmt.Errorf("parsing repository name: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	options := append(c.options, remote.WithContext(reqCtx))
	tags, err := remote.List(repo, options...)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	return tags, nil
}
