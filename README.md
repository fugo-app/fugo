# Fugo

A flexible logs parsing and processing agent.

## Configuration

Fugo uses YAML configuration files. The main configuration file is located at `/etc/fugo/config.yaml`.

```yaml
server:
  listen: 127.0.0.1:2221

storage:
  sqlite:
    path: /var/lib/fugo/fugo.db
```

- `server`: Configuration for the HTTP API server.
- `storage`: Configuration for the log storage backend.

Options for HTTP API server:

- `listen`: The address and port for the HTTP server (e.g., "127.0.0.1:2221" or ":2221").

Options for SQLite storage:

- `path`: Path to the SQLite database file. If the file does not exist, it will be created.
- `journal_mode`: SQLite journal mode. Default is `wal`. Other options are `delete`, `truncate`, `persist`, `memory`, and `off`.
- `synchronous`: SQLite synchronous mode. Default is `normal`. Other options are `full`, `off`, and `extra`.
- `cache_size`: SQLite cache size in pages. Default is `10000` pages. Use negative values for kibibytes.

## Agents Configuration

Agents configuration located in the `agents` sub-directory (e.g. `/etc/fugo/agents/nginx-access.yaml`). Each agent is defined in its own YAML file.

Example, configuration for nginx access log `/etc/fugo/agents/nginx-access.yaml`:

```yaml
fields:
  - name: time
    timestamp:
      format: common
  - name: status
    type: int
  - name: message
    template: "{{.method}} {{.path}}"
file:
  path: /var/log/nginx/access.log
  format: plain
  regex: '^(?P<remote_addr>[^ ]+) - (?P<remote_user>[^ ]+) \[(?P<time>[^\]]+)\] "(?P<method>[^ ]+) (?P<path>[^ ]+) (?P<protocol>[^"]+)" (?P<status>[^ ]+)'
```

Example, configuration for nginx error log `/etc/fugo/agents/nginx-error.yaml`:

```yaml
fields:
  - name: time
    timestamp:
      format: '2006/01/02 15:04:05'
  - name: level
  - name: message
file:
  path: /var/log/nginx/error.log
  format: plain
  regex: '^(?P<time>[^ ]+ [^ ]+) \[(?P<level>[^\]]+)\] \d+#\d+: (?P<message>.*)'
```

- `name`: The name of the agent
- `fields`: A list of fields to store in the log records
- `file`: Configuration for file-based input

## Fields Configuration

Each field can be defined with:

- `name`: The name of the field in the output
- `source`: The source field to extract (defaults to the field name)
- `type`: The type of the field (e.g., `int`, `float`, `string`, `time`). Default is `string`, or `time` if `timestamp` is specified.
- `template`: A Go template to transform source fields into the new field
- `timestamp`: Configuration for timestamp parsing

### Timestamp Configuration

- `format`: Format for the time field (e.g., `rfc3339`, `common`, `stamp`, `unix` or a custom Go layout)

Formats:

- `rfc3339`: Format used in structured logs and JSON `2006-01-02T15:04:05Z07:00`
- `common`: Web server log format, e.g. Apache or Nginx: `02/Jan/2006:15:04:05 -0700`
- `stamp`: Stamp format, common in system logs: `Jan _2 15:04:05`
- `unix`: Unix timestamp in seconds, optionally with fractional part for milliseconds: `1617715200.123`
- Use any valid Go time format string for custom logs, e.g. `02 Jan 2006 15:04:05`

## File-based Input

File-based input has the following configuration:

- `path`: Path to the log file or regex pattern to match multiple files.
- `format`: The format of the log file. Supported formats are `plain` and `json`.
- `regex`: A regex pattern to match the plain log lines. Named capture groups are used to extract fields.

### Path Configuration

The `path` can be a single file or a regex pattern. For example:

```yaml
path: '/var/log/nginx/access_(?P<host>.*)\.log'
```

A named capture group should be in the file name only and can be used in the fields.
