// MCP subcommand for the opendray binary. Inspect and load the MCP
// server registry without starting the gateway. Mirror of `opendray
// skill` for the MCP equivalent.
//
//	opendray mcp list                # all servers (id source enabled name)
//	opendray mcp describe <id>       # full mcp.json to stdout
//	opendray mcp path                # registry root + secrets file path

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/opendray/opendray-v2/internal/config"
	"github.com/opendray/opendray-v2/internal/mcp"
)

func runMcp(args []string) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "path to config.toml (paths are resolved as the gateway does)")
	root := fs.String("root", "", "override MCP registry root")
	asJSON := fs.Bool("json", false, "list/describe: JSON output")
	fs.Usage = func() { fmt.Fprintln(os.Stderr, mcpUsage) }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return 2
	}

	loader, secretsPath, err := openMcp(*cfgPath, *root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	switch rest[0] {
	case "path":
		fmt.Printf("registry: %s\nsecrets:  %s\n", loader.VaultRoot(), secretsPath)
		return 0

	case "list":
		all, err := loader.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(all)
			return 0
		}
		for _, s := range all {
			state := "disabled"
			if s.Enabled {
				state = "enabled"
			}
			transport := s.Transport
			if transport == "" {
				transport = "stdio"
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", s.ID, state, transport, s.Description)
		}
		return 0

	case "describe":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "describe: missing <id>")
			return 2
		}
		s, err := loader.Get(rest[1])
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "MCP server %q not found\n", rest[1])
				return 4
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *asJSON {
			_ = json.NewEncoder(os.Stdout).Encode(s)
			return 0
		}
		body, err := mcp.Marshal(s)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(body))
		return 0

	default:
		fmt.Fprintf(os.Stderr, "unknown mcp command: %s\n", rest[0])
		fs.Usage()
		return 2
	}
}

// openMcp resolves the MCP registry root and secrets file path through
// the gateway's own resolver, so the CLI and the running gateway can
// never operate on different files.
func openMcp(cfgPath, override string) (*mcp.Loader, string, error) {
	paths := loadCLIConfig(cfgPath).Resolve()
	mcpRoot := override
	if mcpRoot == "" {
		mcpRoot = paths.MCP
	} else {
		mcpRoot = config.ExpandPath(mcpRoot)
	}
	return mcp.NewLoader(mcpRoot), paths.MCPSecrets, nil
}

const mcpUsage = `opendray mcp — MCP server registry

usage:
  opendray mcp [flags] <command> [args]

commands:
  path                    print the registry root + secrets file path
  list                    list all MCP servers; cols: id state transport description
  describe <id>           print mcp.json for one server (substituted form)

flags:
  -config FILE            config.toml ([mcp] + [vault] are consulted)
  --root PATH             override MCP registry root
  --json                  list/describe: JSON output

servers load from <registry root>/<id>/mcp.json
secrets file (${KEY} substitution): ~/.opendray/secrets.env (dotenv format)`
