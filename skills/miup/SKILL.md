---
name: miup
description: Manage Milvus instances and components with miup CLI. Use for deploying Milvus on K8s, running local playground, installing tools like birdwatcher/milvus-backup. Trigger when user mentions miup, Milvus deployment, managing Milvus instances, or local Milvus development.
version: "1.0"
---

# MiUp - Milvus Component Manager

MiUp is a CLI tool for managing Milvus vector database instances and ecosystem tools.

## Agent Mode

Always use `--json` flag for structured output in agent mode:

```bash
miup <command> --json
```

## Core Operations

### 1. Local Development (Playground)

Start a local Milvus instance using Docker Compose:

```bash
# Start local Milvus
miup playground start --port 19530

# Check status (JSON output)
miup playground status --json

# List all playgrounds
miup playground list --json

# Stop playground
miup playground stop

# View logs
miup playground logs
```

### 2. Kubernetes Deployment

Deploy Milvus to Kubernetes using Milvus Operator:

```bash
# Pre-flight environment check
miup instance check --json

# Deploy from topology file
miup instance deploy <name> topology.yaml --namespace milvus -y

# List instances
miup instance list --json

# Display instance details
miup instance display <name> --json

# Scale component
miup instance scale <name> --component querynode --replicas 5

# Diagnose issues
miup instance diagnose <name> --json

# Destroy instance
miup instance destroy <name> --force
```

### 3. Component Management

Install and manage Milvus ecosystem tools:

```bash
# Install tools
miup install birdwatcher
miup install milvus-backup

# List installed components
miup list --json

# Run installed tool
miup run birdwatcher -- connect etcd
```

## Output Format

With `--json` flag, output follows this structure:

**Success:**
```json
{
  "success": true,
  "message": "optional message",
  "data": { ... }
}
```

**Error:**
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "error description"
  }
}
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format (agent-friendly) |
| `-v, --verbose` | Enable debug output |
| `--no-color` | Disable color output |

## Reference Documentation

- [Instance Management](references/instance.md) - Deploy, scale, upgrade K8s instances
- [Playground](references/playground.md) - Local development environment
- [Components](references/component.md) - Install and manage tools
