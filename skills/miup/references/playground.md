# Playground Reference

Local Milvus development environment using Docker Compose.

## miup playground start

Start a local Milvus instance.

```bash
miup playground start [flags]
```

**Flags:**
- `--tag` - Instance tag name (default: "default")
- `--port` - Milvus port (default: 19530)
- `--milvus.version` - Milvus version
- `--with-monitor` - Include Prometheus + Grafana

**Example:**
```bash
miup playground start --tag dev --port 19530 --with-monitor
```

## miup playground status

Show playground status.

```bash
miup playground status [--tag <tag>] [--json]
```

**JSON Output:**
```json
{
  "success": true,
  "data": {
    "tag": "default",
    "status": "running",
    "mode": "standalone",
    "version": "v2.5.4",
    "port": 19530,
    "created_at": "2025-01-10T10:00:00Z"
  }
}
```

## miup playground list

List all playground instances.

```bash
miup playground list [--json]
```

**JSON Output:**
```json
{
  "success": true,
  "data": {
    "playgrounds": [
      {
        "tag": "default",
        "status": "running",
        "mode": "standalone",
        "version": "v2.5.4",
        "port": 19530,
        "created_at": "2025-01-10T10:00:00Z"
      }
    ]
  }
}
```

## miup playground stop

Stop a running playground.

```bash
miup playground stop [--tag <tag>]
```

## miup playground logs

View playground logs.

```bash
miup playground logs [--tag <tag>] [--service <service>] [--tail <n>]
```

**Flags:**
- `--tag` - Playground tag
- `--service` - Specific service to show logs for
- `--tail` - Number of lines to show

## miup playground clean

Clean up playground data.

```bash
miup playground clean [--tag <tag>]
```
