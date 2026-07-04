package main

import (
	"os"
	"path/filepath"
	"testing"
)

func tempCodex(t *testing.T) (codexClient, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	return codexClient{path: func() string { return path }}, path
}

const codexSample = `# user's codex config — do not reformat
model = "gpt-5"
provider = "openai"

[mcp_servers.filesystem]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]

[mcp_servers.filesystem.env]
API_KEY = "secret"

[mcp_servers.memory]
url = "https://mcp.example.com/sse"

[mcp_servers.github]
command = "npx"
args = ["-y", "@modelcontextprotocol/server-github"]
env = { GITHUB_PERSONAL_TOKEN = "ghp_x", DEBUG = "1" }
`

func TestCodexListReadsAllTransports(t *testing.T) {
	c, path := tempCodex(t)
	if err := os.WriteFile(path, []byte(codexSample), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := c.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 servers, got %d (%v)", len(got), got)
	}
	fs, ok := got["filesystem"]
	if !ok {
		t.Fatal("filesystem missing")
	}
	if fs.Command != "npx" || len(fs.Args) != 3 {
		t.Fatalf("filesystem stdio not parsed: %+v", fs)
	}
	if fs.Env["API_KEY"] != "secret" {
		t.Fatalf("env sub-table not parsed: %+v", fs.Env)
	}
	mem, ok := got["memory"]
	if !ok || mem.URL != "https://mcp.example.com/sse" {
		t.Fatalf("remote url not parsed: %+v", mem)
	}
	gh, ok := got["github"]
	if !ok || gh.Env["GITHUB_PERSONAL_TOKEN"] != "ghp_x" || gh.Env["DEBUG"] != "1" {
		t.Fatalf("inline env table not parsed: %+v", gh.Env)
	}
}

func TestCodexMissingFileIsEmpty(t *testing.T) {
	c, _ := tempCodex(t)
	got, err := c.List()
	if err != nil {
		t.Fatalf("List on missing file: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 servers for missing file, got %d", len(got))
	}
}

func TestCodexReadOnlyRefusesWrites(t *testing.T) {
	c, _ := tempCodex(t)
	if err := c.Add(Server{Name: "x", Command: "echo"}); err != errReadonly {
		t.Fatalf("Add: want errReadonly, got %v", err)
	}
	if err := c.Remove("x"); err != errReadonly {
		t.Fatalf("Remove: want errReadonly, got %v", err)
	}
}

func TestCodexListIgnoresOtherTopLevelKeys(t *testing.T) {
	c, path := tempCodex(t)
	if err := os.WriteFile(path, []byte("model = \"gpt-5\"\nfoo = 42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := c.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 servers, got %d", len(got))
	}
}
