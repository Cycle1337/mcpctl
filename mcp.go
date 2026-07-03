package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Server is one MCP server entry as it appears in a client's config.
type Server struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
	URL     string // non-empty => remote (http/sse) transport
	Type    string // "http" or "sse"; defaults to "http" when URL is set
}

func (s Server) Transport() string {
	if s.URL != "" {
		if s.Type != "" {
			return s.Type
		}
		return "http"
	}
	return "stdio"
}

// entry renders a server the way a client config file expects it.
func (s Server) entry() map[string]any {
	e := map[string]any{}
	if s.URL != "" {
		t := s.Type
		if t == "" {
			t = "http"
		}
		e["type"] = t
		e["url"] = s.URL
		return e
	}
	e["command"] = s.Command
	if len(s.Args) > 0 {
		e["args"] = s.Args
	}
	if len(s.Env) > 0 {
		e["env"] = s.Env
	}
	return e
}

func serverFromEntry(name string, e any) Server {
	s := Server{Name: name}
	m, ok := e.(map[string]any)
	if !ok {
		return s
	}
	if v, ok := m["command"].(string); ok {
		s.Command = v
	}
	if v, ok := m["type"].(string); ok {
		s.Type = v
	}
	if v, ok := m["url"].(string); ok {
		s.URL = v
	}
	if arr, ok := m["args"].([]any); ok {
		for _, a := range arr {
			if str, ok := a.(string); ok {
				s.Args = append(s.Args, str)
			}
		}
	}
	if env, ok := m["env"].(map[string]any); ok {
		s.Env = map[string]string{}
		for k, v := range env {
			if str, ok := v.(string); ok {
				s.Env[k] = str
			}
		}
	}
	return s
}

var errReadonly = errors.New("client is read-only in mcpctl; use the client's own command (e.g. `claude mcp add`)")

// Client is something that owns an MCP server config.
type Client interface {
	ID() string
	Name() string
	ConfigPath() string
	Installed() bool
	Writable() bool
	List() (map[string]Server, error)
	Add(s Server) error
	Remove(name string) error
}

// jsonClient handles the { "mcpServers": { ... } } shape used by Claude
// Desktop, Cursor and Claude Code.
type jsonClient struct {
	id, name  string
	path      func() string
	installed func() bool
	writable  bool
}

func (c jsonClient) ID() string         { return c.id }
func (c jsonClient) Name() string       { return c.name }
func (c jsonClient) ConfigPath() string { return c.path() }
func (c jsonClient) Writable() bool     { return c.writable }

func (c jsonClient) Installed() bool {
	if c.installed != nil {
		return c.installed()
	}
	if p := c.path(); p != "" {
		if _, err := os.Stat(p); err == nil {
			return true
		}
		if d := filepath.Dir(p); d != "" {
			if info, err := os.Stat(d); err == nil && info.IsDir() {
				return true
			}
		}
	}
	return false
}

func (c jsonClient) List() (map[string]Server, error) {
	data, err := os.ReadFile(c.path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]Server{}, nil
		}
		return nil, err
	}
	top := map[string]any{}
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("%s: %w", c.path(), err)
	}
	out := map[string]Server{}
	if servers, ok := top["mcpServers"].(map[string]any); ok {
		for name, e := range servers {
			out[name] = serverFromEntry(name, e)
		}
	}
	return out, nil
}

func (c jsonClient) Add(s Server) error {
	if !c.writable {
		return errReadonly
	}
	return c.modify(func(servers map[string]any) {
		servers[s.Name] = s.entry()
	})
}

func (c jsonClient) Remove(name string) error {
	if !c.writable {
		return errReadonly
	}
	return c.modify(func(servers map[string]any) {
		delete(servers, name)
	})
}

func (c jsonClient) modify(fn func(map[string]any)) error {
	path := c.path()
	top := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		if e := json.Unmarshal(data, &top); e != nil {
			return fmt.Errorf("%s: %w", path, e)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	servers, _ := top["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	fn(servers)
	top["mcpServers"] = servers
	out, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// clients returns the clients mcpctl knows about, in display order.
func clients() []Client {
	home, _ := os.UserHomeDir()

	desktop := ""
	switch runtime.GOOS {
	case "darwin":
		desktop = filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		desktop = filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json")
	default:
		desktop = filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	}

	return []Client{
		jsonClient{
			id:   "claude-code",
			name: "Claude Code",
			path: func() string { return filepath.Join(home, ".claude.json") },
			installed: func() bool {
				_, err := os.Stat(filepath.Join(home, ".claude.json"))
				return err == nil
			},
			writable: false, // use `claude mcp add`; ~/.claude.json is large and hand-managed
		},
		jsonClient{
			id:       "cursor",
			name:     "Cursor",
			path:     func() string { return filepath.Join(home, ".cursor", "mcp.json") },
			writable: true,
		},
		jsonClient{
			id:       "claude-desktop",
			name:     "Claude Desktop",
			path:     func() string { return desktop },
			writable: true,
		},
	}
}

func findClient(id string) (Client, error) {
	for _, c := range clients() {
		if c.ID() == id {
			return c, nil
		}
	}
	return nil, fmt.Errorf("unknown client %q (want one of %s)", id, clientIDs())
}

func clientIDs() string {
	var ids []string
	for _, c := range clients() {
		ids = append(ids, c.ID())
	}
	return strings.Join(ids, ", ")
}
