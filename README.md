# OCI Registry MCP Server

An MCP (Model Context Protocol) server that provides tools for querying OCI registries and image references.

## Overview

This project implements an SSE-based MCP server that allows LLM-powered applications to interact with OCI registries. It provides tools for retrieving information about container images, listing tags, and more.

## Features

- Get information about OCI images
- List tags for repositories
- Get image manifests
- Get image configs

## MCP Tools

The server provides the following MCP tools:

### get_image_info

Get information about an OCI image.

**Input:**
- `image_ref`: The image reference (e.g., docker.io/library/alpine:latest)

**Output:**
- Image information including digest, size, architecture, OS, creation date, and number of layers

### list_tags

List tags for a repository.

**Input:**
- `repository`: The repository name (e.g., docker.io/library/alpine)

**Output:**
- List of tags for the repository

### get_image_manifest

Get the manifest for an OCI image.

**Input:**
- `image_ref`: The image reference (e.g., docker.io/library/alpine:latest)

**Output:**
- The image manifest

### get_image_config

Get the config for an OCI image.

**Input:**
- `image_ref`: The image reference (e.g., docker.io/library/alpine:latest)

**Output:**
- The image config

## Development

### Prerequisites

- Go 1.21 or later
- Access to OCI registries

### Authentication

The server supports the following authentication methods for accessing private OCI registries:

1. **Bearer Token**: Set the following environment variable:
   - `OCI_TOKEN`: Bearer token for registry authentication

2. **Username and Password**: Set the following environment variables:
   - `OCI_USERNAME`: Username for registry authentication
   - `OCI_PASSWORD`: Password for registry authentication

3. **Docker Config**: If no token or username/password is provided, the server will use the default Docker keychain, which reads credentials from `~/.docker/config.json`.

Examples:
```bash
# Bearer token authentication
export OCI_TOKEN=mytoken

# Username/password authentication
export OCI_USERNAME=myuser
export OCI_PASSWORD=mypassword
```

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run

## Contributing

We welcome contributions to this MCP server! If you'd like to contribute, please review
the [CONTRIBUTING guide](./CONTRIBUTING.md) for details on how to get started.

If you run into a bug or have a feature request, please
[open an issue](https://github.com/StacklokLabs/ocireg-mcp/issues) in the
repository or join us in the `#mcp-servers` channel on our
[community Discord server](https://discord.gg/stacklok).

## License

This project is licensed under the Apache v2 License - see the LICENSE file for details.
