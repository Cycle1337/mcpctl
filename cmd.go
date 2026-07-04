package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

func cmdDoctor(args []string) error {
	for _, c := range clients() {
		state := "not found"
		if c.Installed() {
			state = "installed"
		}
		ro := ""
		if !c.Writable() {
			ro = " (read-only)"
		}
		fmt.Printf("%-16s %-10s  %s%s\n", c.ID(), state, c.ConfigPath(), ro)
	}
	return nil
}

func cmdList(args []string) error {
	filter, err := parseClientFlag(args)
	if err != nil {
		return err
	}
	any := false
	for _, c := range clients() {
		if filter != "" && c.ID() != filter {
			continue
		}
		fmt.Printf("# %s — %s\n", c.Name(), c.ConfigPath())
		servers, err := c.List()
		if err != nil {
			fmt.Printf("  (error: %v)\n", err)
			any = true
			fmt.Println()
			continue
		}
		if len(servers) == 0 {
			fmt.Println("  (no servers)")
		}
		names := make([]string, 0, len(servers))
		for n := range servers {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			s := servers[n]
			fmt.Printf("  %-20s [%s] %s\n", n, s.Transport(), describe(s))
		}
		any = true
		fmt.Println()
	}
	if !any {
		return fmt.Errorf("client %q is not known; see `mcpctl doctor`", filter)
	}
	return nil
}

func cmdShow(args []string) error {
	name, filter, err := splitNameClient(args)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("usage: mcpctl show <name> [--client ID]")
	}
	c, err := pickClient(filter)
	if err != nil {
		return err
	}
	servers, err := c.List()
	if err != nil {
		return err
	}
	s, ok := servers[name]
	if !ok {
		return fmt.Errorf("no server %q in %s", name, c.ID())
	}
	fmt.Printf("client:  %s (%s)\n", c.ID(), c.ConfigPath())
	fmt.Printf("name:    %s\n", s.Name)
	fmt.Printf("transport: %s\n", s.Transport())
	if s.URL != "" {
		fmt.Printf("url:     %s\n", s.URL)
	} else {
		fmt.Printf("command: %s\n", s.Command)
		if len(s.Args) > 0 {
			fmt.Printf("args:    %s\n", strings.Join(s.Args, " "))
		}
	}
	if len(s.Env) > 0 {
		keys := make([]string, 0, len(s.Env))
		for k := range s.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Println("env:")
		for _, k := range keys {
			fmt.Printf("  %s=%s\n", k, s.Env[k])
		}
	}
	return nil
}

func cmdAdd(args []string) error {
	name, clientID, url, env, cmd, err := parseAdd(args)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("usage: mcpctl add <name> --client ID [--env K=V ...] -- <command> [args...] | --url URL")
	}
	if clientID == "" {
		return errors.New("add needs --client ID (one of: " + clientIDs() + ")")
	}
	if url == "" && len(cmd) == 0 {
		return errors.New("add needs either a command (after `--`) or --url")
	}
	c, err := findClient(clientID)
	if err != nil {
		return err
	}
	s := Server{Name: name, Env: env}
	if url != "" {
		s.URL = url
		s.Type = "http"
	} else {
		s.Command = cmd[0]
		s.Args = cmd[1:]
	}
	if err := c.Add(s); err != nil {
		return err
	}
	fmt.Printf("added %q to %s\n", name, c.ID())
	return nil
}

func cmdRemove(args []string) error {
	name, clientID, err := splitNameClient(args)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("usage: mcpctl rm <name> --client ID")
	}
	if clientID == "" {
		return errors.New("rm needs --client ID (one of: " + clientIDs() + ")")
	}
	c, err := findClient(clientID)
	if err != nil {
		return err
	}
	if err := c.Remove(name); err != nil {
		return err
	}
	fmt.Printf("removed %q from %s\n", name, c.ID())
	return nil
}

// describe renders a one-line summary of a server.
func describe(s Server) string {
	if s.URL != "" {
		return s.URL
	}
	parts := append([]string{s.Command}, s.Args...)
	return strings.Join(parts, " ")
}

// parseClientFlag pulls a single --client (or --client=X) out of args.
func parseClientFlag(args []string) (string, error) {
	client := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--client":
			i++
			if i >= len(args) {
				return "", errors.New("--client needs a value")
			}
			client = args[i]
		case strings.HasPrefix(a, "--client="):
			client = strings.TrimPrefix(a, "--client=")
		case strings.HasPrefix(a, "-") && a != "-":
			return "", fmt.Errorf("unknown flag %q", a)
		}
	}
	return client, nil
}

// splitNameClient extracts the first positional (name) and --client.
func splitNameClient(args []string) (string, string, error) {
	client := ""
	name := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--client":
			i++
			if i >= len(args) {
				return "", "", errors.New("--client needs a value")
			}
			client = args[i]
		case strings.HasPrefix(a, "--client="):
			client = strings.TrimPrefix(a, "--client=")
		case strings.HasPrefix(a, "-") && a != "-":
			return "", "", fmt.Errorf("unknown flag %q", a)
		default:
			if name == "" {
				name = a
			}
		}
	}
	return name, client, nil
}

func pickClient(id string) (Client, error) {
	if id == "" {
		return nil, errors.New("needs --client ID (one of: " + clientIDs() + ")")
	}
	return findClient(id)
}

// parseAdd parses: <name> --client ID [--env K=V ...] [--url URL | -- <cmd> args...]
func parseAdd(args []string) (name, client, url string, env map[string]string, cmd []string, err error) {
	env = map[string]string{}
	dashDash := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if dashDash {
			cmd = append(cmd, a)
			continue
		}
		switch {
		case a == "--":
			dashDash = true
		case a == "--client":
			i++
			if i >= len(args) {
				err = errors.New("--client needs a value")
				return
			}
			client = args[i]
		case strings.HasPrefix(a, "--client="):
			client = strings.TrimPrefix(a, "--client=")
		case a == "--url":
			i++
			if i >= len(args) {
				err = errors.New("--url needs a value")
				return
			}
			url = args[i]
		case strings.HasPrefix(a, "--url="):
			url = strings.TrimPrefix(a, "--url=")
		case a == "--env":
			i++
			if i >= len(args) {
				err = errors.New("--env needs K=V")
				return
			}
			if e := setEnv(env, args[i]); e != nil {
				err = e
				return
			}
		case strings.HasPrefix(a, "--env="):
			if e := setEnv(env, strings.TrimPrefix(a, "--env=")); e != nil {
				err = e
				return
			}
		case strings.HasPrefix(a, "-") && a != "-":
			err = fmt.Errorf("unknown flag %q", a)
			return
		default:
			if name == "" {
				name = a
			} else {
				cmd = append(cmd, a)
			}
		}
	}
	// a bare URL as the command also counts as a remote server.
	if url == "" && len(cmd) > 0 && (strings.HasPrefix(cmd[0], "http://") || strings.HasPrefix(cmd[0], "https://")) {
		url = cmd[0]
		cmd = nil
	}
	return
}

func setEnv(env map[string]string, kv string) error {
	k, v, ok := strings.Cut(kv, "=")
	if !ok {
		return fmt.Errorf("--env expects K=V, got %q", kv)
	}
	env[k] = v
	return nil
}
