# Component Management Reference

Install and manage Milvus ecosystem tools.

## Available Components

| Component | Description | Repository |
|-----------|-------------|------------|
| birdwatcher | Milvus diagnostic tool | milvus-io/birdwatcher |
| milvus-backup | Backup and restore | zilliztech/milvus-backup |

## miup install

Install a component.

```bash
miup install <component>[:<version>]
```

**Examples:**
```bash
miup install birdwatcher           # Latest version
miup install birdwatcher:v1.1.0    # Specific version
miup install birdwatcher milvus-backup  # Multiple
```

## miup list

List installed components.

```bash
miup list [--json] [--available]
```

**Flags:**
- `--json` - Output in JSON format
- `--available` - List available (not installed) components

**JSON Output:**
```json
{
  "success": true,
  "data": {
    "components": [
      {
        "name": "birdwatcher",
        "version": "v1.1.0",
        "active": true,
        "installed_at": "2025-01-10T10:00:00Z",
        "path": "~/.miup/components/birdwatcher/v1.1.0/birdwatcher"
      }
    ]
  }
}
```

## miup run

Run an installed component.

```bash
miup run <component>[:<version>] [-- args...]
```

**Example:**
```bash
miup run birdwatcher -- connect etcd
miup run milvus-backup -- --help
```

## miup uninstall

Uninstall a component.

```bash
miup uninstall <component>[:<version>]
```

**Examples:**
```bash
miup uninstall birdwatcher           # Uninstall all versions
miup uninstall birdwatcher:v1.1.0    # Uninstall specific version
```

## miup version

Show miup version.

```bash
miup version [--json]
```

**JSON Output:**
```json
{
  "success": true,
  "data": {
    "version": "v0.1.0",
    "git_hash": "abc123",
    "build_time": "2025-01-10T10:00:00Z",
    "go_version": "go1.21.0"
  }
}
```
