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

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run