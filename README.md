# Fugo

A flexible and lightweight logs parsing and processing agent.

## Configuration

Fugo uses YAML configuration files. The main configuration file is located at `/etc/fugo/config.yaml`.

```yaml
server:
  listen: 127.0.0.1:2221

storage:
  sqlite:
    path: /var/lib/fugo/fugo.db

file_input:
  offsets: /var/lib/fugo/offsets.yaml
  limit: 100
```

- `server`: Configuration for the HTTP API server.
- `storage`: Configuration for the log storage backend.
- `file_input`: Configuration for file-based input.

Options for HTTP API server:

- `listen`: The address and port for the HTTP server (e.g., "127.0.0.1:2221" or ":2221").

Options for SQLite storage:

- `path`: Path to the SQLite database file. If the file does not exist, it will be created.
- `journal_mode`: SQLite journal mode. Default is `wal`. Other options are `delete`, `truncate`, `persist`, `memory`, and `off`.
- `synchronous`: SQLite synchronous mode. Default is `normal`. Other options are `full`, `off`, and `extra`.
- `cache_size`: SQLite cache size in pages. Default is `10000` pages. Use negative values for kibibytes.

Options for file-based input:

- `offsets`: Path to the offsets file. This file stores the last read position for each log file.
- `limit`: Maximum number of lines to read when the log file is opened for the first time. Default is `100`. Set to `0` for no limit.

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
retention:
  period: 7d
  interval: 1h
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
retention:
  period: 3d
  interval: 1h
```

- `name`: The name of the agent
- `fields`: A list of fields to store in the log records
- `file`: Configuration for file-based input
- `retention`: Configuration for log retention

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

## File-based Input Configuration

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

## Retention Configuration

Retention configuration has the following options:

- `period`: The retention period for the logs. It can be specified in minutes, hours, days (e.g., `3h`, `7d`, `3d12h`). Default is `3d`.
- `interval`: The interval for log retention. It can be specified in minutes, hours, days (e.g., `10m`, `1h`). Default is `1h`.

## Querying Logs

Use GET requests to query logs. The API supports filtering and pagination.

```
/api/query/{agent_name}?{query}
```

- `agent_name`: The name of the agent to query logs from (same as yaml file with agent configuration).

### Query Parameters

- `limit`: Maximum number of records to return (default is 100).
- `after`: Return records after the specified cursor.
- `before`: Return records before the specified cursor.

### Query Filters for numeric fields

- `{field_name}__eq`: Exact match
- `{field_name}__ne`: Not equal
- `{field_name}__gt`: Greater than
- `{field_name}__gte`: Greater than or equal to
- `{field_name}__lt`: Less than
- `{field_name}__lte`: Less than or equal to

### Query Filters for string fields

- `{field_name}__exact`: Exact match
- `{field_name}__like`: Partial match (case-insensitive)
- `{field_name}__prefix`: Starts with (case-insensitive)
- `{field_name}__suffix`: Ends with (case-insensitive)

### Query Filters for time fields

- `{field_name}__since`: Return records since the specified time
- `{field_name}__until`: Return records until the specified time

Supported formats:

- "2006-01-02 15:04:05": date and time format
- "2006-01-02": date only format
- "5d": relative time (now minus 5 days), supported units are `s`, `m`, `h`, `d`

### Curl

For example to get last 10 records from the nginx access log, with status not equal to 200:

```bash
curl -G https://example.com/api/query/nginx-access \
     --data-urlencode "limit=10" \
     --data-urlencode "status__ne=200"
```

### Response Format

The response is in JSON Lines format.
Each line has a `_cursor` field. And fields defined in the agent configuration.

```json
{"_cursor":"0000000000000cb1","time":1744321776000,"status":200,"message":"GET /"}
{"_cursor":"0000000000000cb2","time":1744321777000,"status":200,"message":"GET /favicon.ico"}
{"_cursor":"0000000000000cb3","time":1744321778000,"status":404,"message":"GET /not-found"}
```
