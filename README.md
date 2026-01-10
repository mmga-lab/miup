# MiUp

MiUp is a component manager for [Milvus](https://milvus.io) vector database, inspired by [TiUP](https://github.com/pingcap/tiup).

## Features

- **Component Management**: Install, update, and manage Milvus and its dependencies
- **Playground**: Quick local development environment with Docker (single command)
- **Instance Management**: Deploy and manage Milvus instances on Kubernetes
- **Monitoring Integration**: Built-in Prometheus and Grafana support

## Installation

### From Source

```bash
git clone https://github.com/mmga-lab/miup.git
cd miup
make install
```

### Download Binary

```bash
curl -sSL https://raw.githubusercontent.com/mmga-lab/miup/main/install.sh | sh
```

## Quick Start

### Start Local Playground

```bash
# Start a standalone Milvus instance
miup playground start

# Start with monitoring enabled
miup playground start --with-monitor

# Start in cluster mode
miup playground start --mode cluster
```

### Install Components

```bash
# List available components
miup list --available

# Install Milvus
miup install milvus

# Install specific version
miup install milvus:v2.6.0
```

### Deploy an Instance (Kubernetes)

```bash
# Generate topology template
miup instance template > topology.yaml

# Edit the topology file as needed
vim topology.yaml

# Deploy to Kubernetes (requires Milvus Operator)
miup instance deploy my-instance topology.yaml --kubeconfig ~/.kube/config

# View instance status
miup instance display my-instance

# Start/Stop instance
miup instance start my-instance
miup instance stop my-instance

# Scale components
miup instance scale my-instance --component querynode --replicas 3
```

## Commands

| Command | Description |
|---------|-------------|
| `miup install <component>` | Install a component |
| `miup uninstall <component>` | Uninstall a component |
| `miup list` | List installed components |
| `miup playground start` | Start local Milvus playground (Docker) |
| `miup playground stop` | Stop playground |
| `miup instance deploy` | Deploy a Milvus instance (Kubernetes) |
| `miup instance start` | Start an instance |
| `miup instance stop` | Stop an instance |
| `miup instance scale` | Scale instance components |
| `miup instance upgrade` | Upgrade an instance |
| `miup version` | Show version info |

## Configuration

MiUp stores its data in `~/.miup/` by default. You can change this by setting the `MIUP_HOME` environment variable.

```bash
export MIUP_HOME=/custom/path
```

## Development

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Run linters
make lint
```

## License

Apache License 2.0
