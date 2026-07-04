package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// codexServerEntry is the on-disk shape of one [mcp_servers.<name>] table in
// Codex's config.toml. Extra fields Codex knows about (cwd, env_vars,
// bearer_token_env_var, env_http_headers, ...) are ignored on read — we never
// write this file, see Writable() below.
type codexServerEntry struct {
	Command string            `toml:"command,omitempty"`
	Args    []string          `toml:"args,omitempty"`
	Env     map[string]string `toml:"env,omitempty"`
	URL     string            `toml:"url,omitempty"`
	Type    string            `toml:"type,omitempty"`
}

type codexConfig struct {
	McpServers map[string]codexServerEntry `toml:"mcp_servers,omitempty"`
}

type codexClient struct {
	path func() string
}

func (c codexClient) ID() string         { return "codex" }
func (c codexClient) Name() string       { return "Codex" }
func (c codexClient) ConfigPath() string { return c.path() }
func (c codexClient) Writable() bool     { return false }

func (c codexClient) Installed() bool {
	_, err := os.Stat(c.path())
	return err == nil
}

func (c codexClient) List() (map[string]Server, error) {
	data, err := os.ReadFile(c.path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]Server{}, nil
		}
		return nil, err
	}
	// Strip a leading UTF-8 BOM. Notepad and PowerShell both emit one on
	// Windows, and the TOML lib chokes on it.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var cfg codexConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", c.path(), err)
	}
	out := make(map[string]Server, len(cfg.McpServers))
	for name, e := range cfg.McpServers {
		out[name] = Server{Name: name, Command: e.Command, Args: e.Args, Env: e.Env, URL: e.URL, Type: e.Type}
	}
	return out, nil
}

func (c codexClient) Add(s Server) error       { return errReadonly }
func (c codexClient) Remove(name string) error { return errReadonly }
