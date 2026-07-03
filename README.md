# mcpctl

Manage MCP servers across AI coding clients from one terminal.

Claude Desktop, Cursor and Claude Code each keep their own MCP server list,
in their own JSON, in their own place. You end up hand-editing three config
files and getting the indentation wrong. `mcpctl` reads and writes them all
from one command.

## Install

```
go install github.com/Cycle1337/mcpctl@latest
```

or build from source:

```
git clone https://github.com/Cycle1337/mcpctl
cd mcpctl && go build && sudo mv mcpctl /usr/local/bin/
```

Prebuilt binaries are on the [releases page](https://github.com/Cycle1337/mcpctl/releases) once the first
tag is cut.

## Use

```
mcpctl doctor                       # what's installed, where's the config
mcpctl list                         # servers, across all clients
mcpctl add filesystem --client cursor -- npx -y @modelcontextprotocol/server-filesystem /tmp
mcpctl add memory --client claude-desktop --url https://mcp.example.com/sse
mcpctl show memory --client cursor
mcpctl rm memory --client cursor
```

`add` takes `--env K=V` (repeatable). If the thing after `--` starts with
`http(s)://` it's treated as a remote server, so you can drop the `--url`.

What it looks like:

```
$ mcpctl list
# Claude Code — /home/you/.claude.json
  (no servers)

# Cursor — /home/you/.cursor/mcp.json
  filesystem     [stdio] npx -y @modelcontextprotocol/server-filesystem /tmp
  github         [stdio] npx -y @modelcontextprotocol/server-github

# Claude Desktop — ~/.config/Claude/claude_desktop_config.json
  memory         [http] https://mcp.example.com/sse
```

## Clients

| id | client | can mcpctl write? |
| --- | --- | --- |
| `cursor` | Cursor (`~/.cursor/mcp.json`) | yes |
| `claude-desktop` | Claude Desktop | yes |
| `claude-code` | Claude Code (`~/.claude.json`) | no — read-only |

Claude Code is read-only on purpose: `~/.claude.json` is large, holds a lot
of unrelated state, and Claude Code already ships `claude mcp add/remove`.
`mcpctl` lists it so you can see everything in one place, but points you at
`claude mcp` to change it.

## Status

v0.1 — the three clients above, stdio and http/sse transports. Planned:
Codex (`~/.codex/config.toml`), `enable`/`disable`, and a `--json` output
flag for scripting. Issues and PRs welcome.

## Why not just edit the JSON

You can. But I kept forgetting which file holds Cursor's servers, kept
breaking Claude Desktop's config with a trailing comma, and kept wanting to
see all my servers in one list to remember which one I'd put the GitHub token
in. So this.

## License

MIT.
