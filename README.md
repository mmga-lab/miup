# MiUp

MiUp is a component manager for [Milvus](https://milvus.io) vector database, inspired by [TiUP](https://github.com/pingcap/tiup).

## Features

- **Component Management**: Install, update, and manage Milvus and its dependencies
- **Playground**: Quick local development environment with a single command
- **Cluster Management**: Deploy and manage production clusters (local or Kubernetes)
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

### Deploy a Cluster

```bash
# Generate topology template
miup cluster template > topology.yaml

# Edit the topology file as needed
vim topology.yaml

# Deploy the cluster
miup cluster deploy my-cluster topology.yaml

# For Kubernetes deployment
miup cluster deploy my-cluster topology.yaml --kubernetes

# View cluster status
miup cluster display my-cluster

# Start/Stop cluster
miup cluster start my-cluster
miup cluster stop my-cluster
```

## Commands

| Command | Description |
|---------|-------------|
| `miup install <component>` | Install a component |
| `miup uninstall <component>` | Uninstall a component |
| `miup list` | List installed components |
| `miup playground start` | Start local Milvus playground |
| `miup playground stop` | Stop playground |
| `miup cluster deploy` | Deploy a Milvus cluster |
| `miup cluster start` | Start a cluster |
| `miup cluster stop` | Stop a cluster |
| `miup cluster scale-out` | Scale out a cluster |
| `miup cluster upgrade` | Upgrade a cluster |
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
