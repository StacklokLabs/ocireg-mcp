# OCI Registry MCP Server

[![Trust Score](https://archestra.ai/mcp-catalog/api/badge/quality/StacklokLabs/ocireg-mcp)](https://archestra.ai/mcp-catalog/stackloklabs__ocireg-mcp)
An MCP (Model Context Protocol) server that provides tools for querying OCI
registries and image references.

## Overview

This project implements an SSE-based MCP server that allows LLM-powered
applications to interact with OCI registries. It provides tools for retrieving
information about container images, listing tags, and more.

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

- Image information including digest, size, architecture, OS, creation date, and
  number of layers

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

## Usage

### Running with ToolHive (Recommended)

The easiest way to run the OCI Registry MCP server is using
[ToolHive](https://github.com/stacklok/toolhive), which provides secure,
containerized deployment of MCP servers:

```bash
# Install ToolHive (if not already installed)
# See: https://docs.stacklok.com/toolhive/guides-cli/install

# Register a supported client so ToolHive can auto-configure your environment
thv client setup
# Run the OCI Registry MCP server (packaged as 'oci-registry' in ToolHive)
thv run oci-registry

# List running servers
thv list

# Get detailed information about the server
thv registry info oci-registry
```

The server will be available to your MCP-compatible clients and can query OCI
registries for image information.

#### Authentication with ToolHive

If you need to access private registries, you can provide authentication
credentials using ToolHive's secret management:

```bash
# For bearer token authentication
thv secret set oci-token
# Enter your bearer token when prompted

thv run --secret oci-token,target=OCI_TOKEN oci-registry

# For username/password authentication
thv secret set oci-username
thv secret set oci-password
# Enter your credentials when prompted

thv run --secret oci-username,target=OCI_USERNAME --secret oci-password,target=OCI_PASSWORD oci-registry
```

## Development

### Prerequisites

- Go 1.21 or later
- Access to OCI registries

### Authentication

The server supports the following authentication methods for accessing private
OCI registries (in order of priority):

1. **HTTP Authorization Header** (Highest Priority): Include a bearer token in
   the HTTP request's `Authorization` header:

   - `Authorization: Bearer <your-token>`
   - This method takes precedence over all other authentication methods
   - When present, environment variables and Docker config are ignored

2. **Bearer Token Environment Variable**: Set the following environment variable:

   - `OCI_TOKEN`: Bearer token for registry authentication

3. **Username and Password**: Set the following environment variables:

   - `OCI_USERNAME`: Username for registry authentication
   - `OCI_PASSWORD`: Password for registry authentication

4. **Docker Config** (Lowest Priority): If no other authentication is provided,
   the server will use the default Docker keychain, which reads credentials from
   `~/.docker/config.json`.

Examples:

```bash
# HTTP Authorization header (for per-request authentication)
# This is handled automatically by the MCP client when making requests
# Example: curl -H "Authorization: Bearer mytoken" http://localhost:8080/...

# Bearer token authentication via environment variable
export OCI_TOKEN=mytoken

# Username/password authentication via environment variables
export OCI_USERNAME=myuser
export OCI_PASSWORD=mypassword
```

### Port Configuration

The server can be configured to listen on a specific port using either:

1. **Environment Variable**:

   - `MCP_PORT`: The port number to listen on (must be between 0 and 65535)
   - If not set or invalid, defaults to port 8080

2. **Command-line Flag**:
   - `-port`: Overrides the environment variable setting (must be between 0
     and 65535)
   - If invalid port provided it defaults to port 8080
   - Example: `./ocireg-mcp -port 9090`

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```

## Contributing

We welcome contributions to this MCP server! If you'd like to contribute, please
review the [CONTRIBUTING guide](./CONTRIBUTING.md) for details on how to get
started.

If you run into a bug or have a feature request, please
[open an issue](https://github.com/StacklokLabs/ocireg-mcp/issues) in the
repository or join us in the `#mcp-servers` channel on our
[community Discord server](https://discord.gg/stacklok).

## License

This project is licensed under the Apache v2 License - see the LICENSE file for
details.
