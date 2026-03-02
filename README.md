# agent-monitor

TUI for monitoring [OpenCode](https://github.com/sst/opencode) agent sessions in real-time.

![Go](https://img.shields.io/badge/Go-1.24-blue)

## Features

- Agent hierarchy tree with parent→child delegation
- Session detail panel with status, duration, and todo progress
- Live activity stream via SSE
- Auto-discovers running OpenCode server from state directory

## Install

```sh
go install github.com/tkozakas/agent-monitor@latest
```

## Usage

Start OpenCode in one terminal, then run:

```sh
agent-monitor
```

### Keybinds

| Key | Action |
|---|---|
| `j` / `k` | Navigate sessions |
| `Tab` | Switch panel |
| `r` | Force refresh |
| `a` | Abort selected session |
| `q` | Quit |

### tmux popup

```sh
bind-key a display-popup -w 80% -h 80% -E "agent-monitor"
```
