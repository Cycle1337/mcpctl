package main

import (
	"fmt"
	"os"
)

const usage = `mcpctl — manage MCP servers across AI coding clients

Claude Desktop, Cursor and Claude Code each keep their own MCP server list,
in their own JSON, in their own place. mcpctl reads and writes them all from
one command so you stop hand-editing three config files.

usage:
  mcpctl doctor                       show detected clients and their config paths
  mcpctl list [--client ID]           list servers across clients (or one)
  mcpctl show <name> --client ID      show one server's config
  mcpctl add <name> --client ID [--env K=V ...] -- <command> [args...]
                                      add a stdio server; use --url for a remote one
  mcpctl rm <name> --client ID        remove a server

clients: claude-code, cursor, claude-desktop
  (claude-code is read-only here — its ~/.claude.json is large and hand-managed,
   so use "claude mcp add" to change it. mcpctl still lists it.)

examples:
  mcpctl add filesystem --client cursor -- npx -y @modelcontextprotocol/server-filesystem /tmp
  mcpctl add memory --client claude-desktop --url https://mcp.example.com/sse
  mcpctl rm memory --client cursor
  mcpctl list`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "mcpctl:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Println(usage)
		return nil
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "doctor":
		return cmdDoctor(rest)
	case "list", "ls":
		return cmdList(rest)
	case "show":
		return cmdShow(rest)
	case "add":
		return cmdAdd(rest)
	case "rm", "remove":
		return cmdRemove(rest)
	case "-h", "--help", "help":
		fmt.Println(usage)
		return nil
	case "-v", "--version":
		fmt.Println("mcpctl 0.1.0")
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}
