package main

import (
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

func TestParseToolFilter(t *testing.T) {
	known := []string{"get_feeds", "create_user", "delete_user"}

	tests := []struct {
		name    string
		allow   string
		deny    string
		wantErr bool
		allowed map[string]bool
	}{
		{
			name:    "no filter registers everything",
			allowed: map[string]bool{"get_feeds": true, "create_user": true, "delete_user": true},
		},
		{
			name:    "allowlist",
			allow:   "get_feeds",
			allowed: map[string]bool{"get_feeds": true, "create_user": false, "delete_user": false},
		},
		{
			name:    "denylist with mixed separators",
			deny:    "create_user,\n delete_user",
			allowed: map[string]bool{"get_feeds": true, "create_user": false, "delete_user": false},
		},
		{
			name:    "both lists",
			allow:   "get_feeds",
			deny:    "create_user",
			wantErr: true,
		},
		{
			name:    "unknown name",
			deny:    "delete_usr",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parseToolFilter(tt.allow, tt.deny)
			if err == nil {
				err = filter.validate(known)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for name, want := range tt.allowed {
				if got := filter.allows(name); got != want {
					t.Errorf("allows(%q) = %v, want %v", name, got, want)
				}
			}
		})
	}
}

func TestRegisterToolsAppliesFilter(t *testing.T) {
	filter, err := parseToolFilter("", "create_user delete_user")
	if err != nil {
		t.Fatal(err)
	}

	mcpServer := server.NewMCPServer("test", "test")
	if err := (&MinifluxServer{}).RegisterTools(mcpServer, filter); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	tools := mcpServer.ListTools()
	if _, ok := tools["create_user"]; ok {
		t.Error("create_user should not be registered")
	}
	if _, ok := tools["delete_user"]; ok {
		t.Error("delete_user should not be registered")
	}
	if _, ok := tools["get_feeds"]; !ok {
		t.Error("get_feeds should be registered")
	}
}
