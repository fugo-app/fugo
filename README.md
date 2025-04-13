# Fugo

A lightweight log collection and querying agent. Tail your logs, store in SQLite, and query via HTTP.

## Features

- Collect logs from json or text files
- Convert logs structured data
- Store logs in SQLite
- Query logs via HTTP

## Installation

```bash
curl -sSfL https://fugo.app/install.sh | sudo sh
```

Start the service:

```bash
sudo systemctl start fugo
```

Enable the service to start on boot:

```bash
sudo systemctl enable fugo
```

## Documentation

[Read the documentation](https://fugo.app/guides/quick-start/)
