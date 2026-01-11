# Instance Management Reference

Kubernetes Milvus instance management commands.

## miup instance list

List all managed Milvus instances.

```bash
miup instance list [--json]
```

**JSON Output:**
```json
{
  "success": true,
  "data": {
    "instances": [
      {
        "name": "prod",
        "status": "running",
        "mode": "distributed",
        "backend": "kubernetes",
        "version": "v2.5.4",
        "port": 19530,
        "namespace": "milvus",
        "created_at": "2025-01-10T10:00:00Z"
      }
    ]
  }
}
```

## miup instance deploy

Deploy a new Milvus instance to Kubernetes.

```bash
miup instance deploy <name> <topology.yaml> [flags]
```

**Flags:**
- `--namespace` - Kubernetes namespace (default: milvus)
- `--milvus.version` - Milvus version
- `--kubeconfig` - Path to kubeconfig
- `--with-monitor` - Enable Prometheus monitoring
- `-y, --yes` - Skip confirmation

**Example:**
```bash
miup instance deploy prod topology.yaml --namespace milvus -y
```

## miup instance display

Show instance details.

```bash
miup instance display <name> [--json]
```

**JSON Output:**
```json
{
  "success": true,
  "data": {
    "name": "prod",
    "status": "running",
    "mode": "distributed",
    "backend": "kubernetes",
    "version": "v2.5.4",
    "port": 19530,
    "namespace": "milvus",
    "created_at": "2025-01-10T10:00:00Z"
  }
}
```

## miup instance scale

Scale a component in the instance.

```bash
miup instance scale <name> --component <comp> [flags]
```

**Flags:**
- `-c, --component` - Component to scale (required)
- `-r, --replicas` - Number of replicas
- `--cpu-request` - CPU request
- `--memory-request` - Memory request

**Components:** proxy, querynode, datanode, indexnode, rootcoord, querycoord, datacoord, indexcoord

**Examples:**
```bash
# Horizontal scaling
miup instance scale prod --component querynode --replicas 5

# Vertical scaling
miup instance scale prod --component querynode --cpu-request 4 --memory-request 16Gi
```

## miup instance diagnose

Run health diagnostics on an instance.

```bash
miup instance diagnose <name> [--json]
```

**JSON Output:**
```json
{
  "healthy": true,
  "summary": "All components healthy",
  "components": [
    {"name": "proxy", "status": "OK", "replicas": 2, "ready": 2}
  ],
  "issues": []
}
```

## miup instance check

Pre-deployment environment check.

```bash
miup instance check [--json] [--namespace <ns>]
```

Checks:
- Kubernetes connectivity
- Kubernetes version (requires 1.20+)
- Milvus Operator installation
- Storage class availability

## Other Commands

| Command | Description |
|---------|-------------|
| `start <name>` | Start stopped instance |
| `stop <name>` | Stop running instance |
| `destroy <name> --force` | Destroy instance and data |
| `upgrade <name> <version>` | Upgrade Milvus version |
| `config show <name>` | Show configuration |
| `config set <name> key=value` | Set configuration |
| `logs <name>` | View instance logs |
| `replicas <name>` | Show replica counts |
| `template` | Print topology template |
