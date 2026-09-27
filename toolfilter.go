package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// toolFilter decides which tools are registered with the MCP server.
//
// It is configured with at most one of MCP_TOOLS_ALLOW or MCP_TOOLS_DENY,
// each a comma- or whitespace-separated list of tool names. With neither set,
// every tool is registered.
type toolFilter struct {
	allow map[string]bool
	deny  map[string]bool
}

func loadToolFilter() (toolFilter, error) {
	return parseToolFilter(os.Getenv("MCP_TOOLS_ALLOW"), os.Getenv("MCP_TOOLS_DENY"))
}

func parseToolFilter(allow, deny string) (toolFilter, error) {
	allowSet := parseToolNames(allow)
	denySet := parseToolNames(deny)
	if allowSet != nil && denySet != nil {
		return toolFilter{}, fmt.Errorf("MCP_TOOLS_ALLOW and MCP_TOOLS_DENY are mutually exclusive")
	}
	return toolFilter{allow: allowSet, deny: denySet}, nil
}

func parseToolNames(value string) map[string]bool {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(fields) == 0 {
		return nil
	}
	names := make(map[string]bool, len(fields))
	for _, name := range fields {
		names[name] = true
	}
	return names
}

// validate rejects names that match no known tool. A typo in a denylist would
// otherwise silently expose the tool it was meant to hide.
func (f toolFilter) validate(known []string) error {
	knownSet := make(map[string]bool, len(known))
	for _, name := range known {
		knownSet[name] = true
	}

	var unknown []string
	for _, names := range []map[string]bool{f.allow, f.deny} {
		for name := range names {
			if !knownSet[name] {
				unknown = append(unknown, name)
			}
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("unknown tool name(s) in MCP_TOOLS_ALLOW/MCP_TOOLS_DENY: %s", strings.Join(unknown, ", "))
	}
	return nil
}

func (f toolFilter) allows(name string) bool {
	if f.allow != nil {
		return f.allow[name]
	}
	return !f.deny[name]
}
