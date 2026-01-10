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

# Start with monitoring enabled (Prometheus + Grafana)
miup playground start --with-monitor

# View playground status
miup playground status

# View logs
miup playground logs

# Stop playground
miup playground stop
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

### Component Management

| Command | Description |
|---------|-------------|
| `miup install <component>` | Install a component (e.g., birdwatcher, milvus-backup) |
| `miup uninstall <component>` | Uninstall a component |
| `miup list` | List installed components |
| `miup list --available` | List available components |
| `miup run <component>` | Run an installed component |

### Playground (Local Docker)

| Command | Description |
|---------|-------------|
| `miup playground start` | Start local Milvus instance |
| `miup playground stop` | Stop playground |
| `miup playground status` | Show playground status |
| `miup playground list` | List all playground instances |
| `miup playground logs` | View playground logs |
| `miup playground clean` | Remove playground data |

### Instance Management (Kubernetes)

| Command | Description |
|---------|-------------|
| `miup instance deploy` | Deploy a Milvus instance |
| `miup instance list` | List all instances |
| `miup instance display` | Show instance details |
| `miup instance start` | Start an instance |
| `miup instance stop` | Stop an instance |
| `miup instance destroy` | Destroy an instance |
| `miup instance scale` | Scale instance components |
| `miup instance replicas` | Show current replica counts |
| `miup instance upgrade` | Upgrade instance version |
| `miup instance logs` | View instance logs |
| `miup instance diagnose` | Run health diagnostics |
| `miup instance config show` | Show instance configuration |
| `miup instance config set` | Set configuration value |
| `miup instance config import` | Import configuration from file |
| `miup instance config export` | Export configuration to stdout |
| `miup instance template` | Print topology template |

### Image Mirror (Offline Deployment)

| Command | Description |
|---------|-------------|
| `miup mirror pull` | Pull images from registry |
| `miup mirror save` | Save images to tar file |
| `miup mirror load` | Load images from tar file |
| `miup mirror push` | Push images to private registry |
| `miup mirror list` | List required images |

### Benchmark

| Command | Description |
|---------|-------------|
| `miup bench milvus prepare` | Prepare benchmark data |
| `miup bench milvus search` | Run search benchmark |
| `miup bench milvus insert` | Run insert benchmark |
| `miup bench milvus cleanup` | Clean up benchmark data |

### Utility

| Command | Description |
|---------|-------------|
| `miup version` | Show version info |
| `miup completion` | Generate shell completion |

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
