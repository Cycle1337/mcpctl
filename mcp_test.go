package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func tempClient(t *testing.T, writable bool) (jsonClient, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	return jsonClient{
		id:       "test",
		name:     "Test",
		path:     func() string { return path },
		writable: writable,
	}, path
}

func TestAddListRemove(t *testing.T) {
	c, _ := tempClient(t, true)

	if err := c.Add(Server{Name: "fs", Command: "npx", Args: []string{"-y", "srv"}, Env: map[string]string{"K": "v"}}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := c.Add(Server{Name: "mem", URL: "https://x/sse", Type: "sse"}); err != nil {
		t.Fatalf("add http: %v", err)
	}

	got, err := c.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 servers, got %d", len(got))
	}
	if got["fs"].Command != "npx" || len(got["fs"].Args) != 2 || got["fs"].Env["K"] != "v" {
		t.Fatalf("fs not round-tripped: %+v", got["fs"])
	}
	if got["mem"].URL != "https://x/sse" || got["mem"].Transport() != "sse" {
		t.Fatalf("mem not round-tripped: %+v", got["mem"])
	}

	if err := c.Remove("fs"); err != nil {
		t.Fatalf("rm: %v", err)
	}
	got, _ = c.List()
	if _, ok := got["fs"]; ok {
		t.Fatalf("fs still present after rm")
	}
	if len(got) != 1 {
		t.Fatalf("want 1 server after rm, got %d", len(got))
	}
}

func TestPreservesUnknownTopLevelKeys(t *testing.T) {
	c, path := tempClient(t, true)
	// seed with an unrelated key the tool does not know about
	seed := map[string]any{
		"allowHttp": true,
		"mcpServers": map[string]any{
			"old": map[string]any{"command": "true"},
		},
	}
	b, _ := json.MarshalIndent(seed, "", "  ")
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := c.Add(Server{Name: "new", Command: "echo"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	out, _ := os.ReadFile(path)
	var after map[string]any
	if err := json.Unmarshal(out, &after); err != nil {
		t.Fatalf("invalid json after add: %v", err)
	}
	if allow, _ := after["allowHttp"].(bool); !allow {
		t.Fatalf("allowHttp not preserved: %v", after["allowHttp"])
	}
	servers, _ := after["mcpServers"].(map[string]any)
	if _, ok := servers["old"]; !ok {
		t.Fatalf("existing server 'old' was clobbered")
	}
	if _, ok := servers["new"]; !ok {
		t.Fatalf("new server not written")
	}
}

func TestReadonlyRefusesWrites(t *testing.T) {
	c, _ := tempClient(t, false)
	if err := c.Add(Server{Name: "x", Command: "echo"}); err != errReadonly {
		t.Fatalf("want errReadonly, got %v", err)
	}
	if err := c.Remove("x"); err != errReadonly {
		t.Fatalf("want errReadonly, got %v", err)
	}
}

func TestServerTransport(t *testing.T) {
	cases := []struct {
		s    Server
		want string
	}{
		{Server{Command: "x"}, "stdio"},
		{Server{URL: "https://u"}, "http"},
		{Server{URL: "https://u", Type: "sse"}, "sse"},
	}
	for _, tc := range cases {
		if got := tc.s.Transport(); got != tc.want {
			t.Errorf("Transport() = %q, want %q", got, tc.want)
		}
	}
}

func TestEntryRoundTrip(t *testing.T) {
	// This is the path List() uses: entry -> JSON -> serverFromEntry.
	want := Server{Name: "fs", Command: "npx", Args: []string{"-y", "srv"}, Env: map[string]string{"K": "v"}}
	b, err := json.Marshal(want.entry())
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	got := serverFromEntry("fs", m)
	if got.Command != want.Command || !reflect.DeepEqual(got.Args, want.Args) || !reflect.DeepEqual(got.Env, want.Env) {
		t.Fatalf("stdio round-trip mismatch:\nwant=%+v\ngot =%+v", want, got)
	}

	// remote server
	rm := Server{Name: "mem", URL: "https://u/sse", Type: "sse"}
	b, _ = json.Marshal(rm.entry())
	var m2 map[string]any
	_ = json.Unmarshal(b, &m2)
	got2 := serverFromEntry("mem", m2)
	if got2.URL != rm.URL || got2.Transport() != "sse" {
		t.Fatalf("http round-trip mismatch:\nwant=%+v\ngot =%+v", rm, got2)
	}
}
