package main

import (
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

func TestEveryToolIsAnnotated(t *testing.T) {
	mcpServer := server.NewMCPServer("test", "test")
	if err := (&MinifluxServer{}).RegisterTools(mcpServer, toolFilter{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	tools := mcpServer.ListTools()
	for name, tool := range tools {
		a := tool.Tool.Annotations
		if a.ReadOnlyHint == nil || a.OpenWorldHint == nil {
			t.Errorf("%s: missing readOnlyHint or openWorldHint", name)
			continue
		}
		if !*a.ReadOnlyHint && (a.DestructiveHint == nil || a.IdempotentHint == nil) {
			t.Errorf("%s: write tool is missing destructiveHint or idempotentHint", name)
		}
	}

	for name := range toolAnnotations {
		if _, ok := tools[name]; !ok {
			t.Errorf("annotation for unknown tool %q", name)
		}
	}
}
