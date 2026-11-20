package cli

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMcpToolSchemasAndValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := newMcpServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := serverSession.Close(); err != nil {
			t.Error(err)
		}
	}()
	client := mcp.NewClient(&mcp.Implementation{Name: "g-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	}()
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantRequired := map[string][]string{
		"env": nil, "clean": nil, "ls": nil, "ls-remote": nil,
		"install": {"version"}, "uninstall": {"version"}, "use": {"version"},
	}
	if len(result.Tools) != len(wantRequired) {
		t.Fatalf("got %d tools, want %d", len(result.Tools), len(wantRequired))
	}
	for _, tool := range result.Tools {
		want, ok := wantRequired[tool.Name]
		if !ok {
			t.Fatalf("unexpected tool %q", tool.Name)
		}
		data, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Type       string   `json:"type"`
			Required   []string `json:"required"`
			Properties map[string]struct {
				Description string `json:"description"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatal(err)
		}
		if schema.Type != "object" || len(schema.Required) != len(want) {
			t.Fatalf("%s: unexpected schema %s", tool.Name, data)
		}
		for i, name := range want {
			if schema.Required[i] != name {
				t.Errorf("%s: required = %v, want %v", tool.Name, schema.Required, want)
			}
		}
		for name, property := range schema.Properties {
			if property.Description == "" {
				t.Errorf("%s.%s: missing description", tool.Name, name)
			}
		}
	}
	// Invalid requests must be rejected before executing destructive CLI commands.
	for _, name := range []string{"install", "uninstall", "use"} {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: map[string]any{}})
		if err == nil && (res == nil || !res.IsError) {
			t.Errorf("%s accepted missing required version", name)
		}
	}
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "ls", Arguments: map[string]any{"output": 42}})
	if err == nil && (res == nil || !res.IsError) {
		t.Error("ls accepted a non-string output format")
	}
}
