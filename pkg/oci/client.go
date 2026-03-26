// Package oci provides functionality for interacting with OCI registries.
package oci

import (
	"context"
	"fmt"
	"io"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
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

// WithBasicAuth returns a remote.Option for basic authentication with username and password.
func WithBasicAuth(username, password string) remote.Option {
	return remote.WithAuth(&authn.Basic{
		Username: username,
		Password: password,
	})
}

// WithDefaultKeychain returns a remote.Option that uses the default keychain for authentication.
// The default keychain reads credentials from the Docker config file (~/.docker/config.json).
func WithDefaultKeychain() remote.Option {
	return remote.WithAuthFromKeychain(authn.DefaultKeychain)
}

// optionsWith returns a new slice containing c.options plus the given extras,
// avoiding mutation of the original backing array under concurrent use.
func (c *Client) optionsWith(extras ...remote.Option) []remote.Option {
	opts := make([]remote.Option, 0, len(c.options)+len(extras))
	opts = append(opts, c.options...)
	opts = append(opts, extras...)
	return opts
}

// WithBearerToken returns a remote.Option for token-based authentication.
func WithBearerToken(token string) remote.Option {
	return remote.WithAuth(&authn.Bearer{
		Token: token,
	})
}

// GetImage retrieves an image from a registry.
func (c *Client) GetImage(ctx context.Context, imageRef string) (v1.Image, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference: %w", err)
	}

	options := c.optionsWith(remote.WithContext(ctx))
	img, err := remote.Image(ref, options...)
	if err != nil {
		return nil, fmt.Errorf("fetching image: %w", err)
	}

	return img, nil
}

// GetImageManifest retrieves the manifest for an image.
func (c *Client) GetImageManifest(ctx context.Context, imageRef string) (*v1.Manifest, error) {
	img, err := c.GetImage(ctx, imageRef)
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
	img, err := c.GetImage(ctx, imageRef)
	if err != nil {
		return nil, err
	}

	config, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("getting config: %w", err)
	}

	return config, nil
}

// ListReferrers lists OCI artifacts that refer to the given image via the Referrers API.
// If artifactTypeFilter is non-empty, only referrers matching that artifact type are returned.
func (c *Client) ListReferrers(
	ctx context.Context, imageRef, artifactTypeFilter string,
) (*v1.IndexManifest, error) {
	repo, digest, err := c.ResolveDigest(ctx, imageRef)
	if err != nil {
		return nil, err
	}

	digestRef := repo.Digest(digest.String())
	options := c.optionsWith(remote.WithContext(ctx))

	if artifactTypeFilter != "" {
		options = append(options,
			remote.WithFilter("artifactType", artifactTypeFilter))
	}

	idx, err := remote.Referrers(digestRef, options...)
	if err != nil {
		return nil, fmt.Errorf("listing referrers: %w", err)
	}

	indexManifest, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("getting index manifest: %w", err)
	}

	return indexManifest, nil
}

// LegacyCosignArtifact represents an artifact found via legacy cosign tag scheme.
type LegacyCosignArtifact struct {
	Digest       v1.Hash
	Size         int64
	MediaType    types.MediaType
	ArtifactType string
	TagSuffix    string // "sig", "att", or "sbom"
}

// legacyCosignSuffixes maps tag suffixes to their artifact types.
var legacyCosignSuffixes = map[string]string{
	"sig":  "application/vnd.dev.cosign.artifact.sig.v1+json",
	"att":  "application/vnd.dsse.envelope.v1+json",
	"sbom": "application/vnd.dev.cosign.artifact.sbom.v1+json",
}

// ListLegacyCosignArtifacts discovers artifacts stored via the legacy
// cosign tag scheme (sha256-<hex>.sig, .att, .sbom).
func (c *Client) ListLegacyCosignArtifacts(
	ctx context.Context, repo name.Repository, imageDigest v1.Hash,
) []LegacyCosignArtifact {
	options := c.optionsWith(remote.WithContext(ctx))
	hex := imageDigest.Hex

	var artifacts []LegacyCosignArtifact
	for suffix, artifactType := range legacyCosignSuffixes {
		tagName := fmt.Sprintf("sha256-%s.%s", hex, suffix)
		tag := repo.Tag(tagName)

		desc, err := remote.Head(tag, options...)
		if err != nil {
			continue
		}

		artifacts = append(artifacts, LegacyCosignArtifact{
			Digest:       desc.Digest,
			Size:         desc.Size,
			MediaType:    desc.MediaType,
			ArtifactType: artifactType,
			TagSuffix:    suffix,
		})
	}

	return artifacts
}

// ResolveDigest resolves an image reference to a digest and repository.
func (c *Client) ResolveDigest(
	ctx context.Context, imageRef string,
) (name.Repository, v1.Hash, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return name.Repository{}, v1.Hash{},
			fmt.Errorf("parsing image reference: %w", err)
	}

	options := c.optionsWith(remote.WithContext(ctx))

	digestRef, ok := ref.(name.Digest)
	if ok {
		digest, err := v1.NewHash(digestRef.Identifier())
		if err != nil {
			return name.Repository{}, v1.Hash{},
				fmt.Errorf("parsing digest: %w", err)
		}
		return ref.Context(), digest, nil
	}

	desc, err := remote.Head(ref, options...)
	if err != nil {
		return name.Repository{}, v1.Hash{},
			fmt.Errorf("resolving image digest: %w", err)
	}

	return ref.Context(), desc.Digest, nil
}

// GetArtifactContent fetches the content of an artifact by repository and digest.
// It returns the first layer's content, its media type, and any error.
func (c *Client) GetArtifactContent(ctx context.Context, repo, digest string) ([]byte, types.MediaType, error) {
	ref, err := name.NewDigest(repo + "@" + digest)
	if err != nil {
		return nil, "", fmt.Errorf("parsing artifact reference: %w", err)
	}

	options := c.optionsWith(remote.WithContext(ctx))
	img, err := remote.Image(ref, options...)
	if err != nil {
		return nil, "", fmt.Errorf("fetching artifact: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, "", fmt.Errorf("getting artifact layers: %w", err)
	}

	if len(layers) == 0 {
		return nil, "", fmt.Errorf("artifact has no layers")
	}

	layer := layers[0]
	mediaType, err := layer.MediaType()
	if err != nil {
		return nil, "", fmt.Errorf("getting layer media type: %w", err)
	}

	rc, err := layer.Compressed()
	if err != nil {
		return nil, "", fmt.Errorf("reading layer content: %w", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", fmt.Errorf("reading layer content: %w", err)
	}

	return content, mediaType, nil
}

// ListTags lists all tags for a repository.
func (c *Client) ListTags(ctx context.Context, repoName string) ([]string, error) {
	repo, err := name.NewRepository(repoName)
	if err != nil {
		return nil, fmt.Errorf("parsing repository name: %w", err)
	}

	options := c.optionsWith(remote.WithContext(ctx))
	tags, err := remote.List(repo, options...)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	return tags, nil
}
