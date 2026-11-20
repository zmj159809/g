// Copyright (c) 2025 voidint <voidint@126.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"github.com/voidint/g/build"
)

func runMcpServer(c *cli.Context) error {
	if err := newMcpServer().Run(c.Context, &mcp.StdioTransport{}); err != nil {
		return cli.Exit(errstring(err), 1)
	}
	return nil
}

func newMcpServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "g", Version: build.ShortVersion}, nil)
	mcp.AddTool(s, &mcp.Tool{Name: "env", Description: "Show environment variables of g"}, envHandler)
	mcp.AddTool(s, &mcp.Tool{Name: "clean", Description: "Delete the cached installation package files"}, cleanHandler)
	mcp.AddTool(s, &mcp.Tool{Name: "ls", Description: "List installed go sdk versions"}, lsHandler)
	mcp.AddTool(s, &mcp.Tool{Name: "ls-remote", Description: "List remote go sdk versions available for install"}, lsRemoteHandler)
	mcp.AddTool(s, &mcp.Tool{Name: "install", Description: "Download and install a go sdk version"}, installHandler)
	mcp.AddTool(s, &mcp.Tool{Name: "uninstall", Description: "Uninstall a go sdk version"}, uninstallHandler)
	mcp.AddTool(s, &mcp.Tool{Name: "use", Description: "Switch to specified go sdk version"}, useHandler)
	return s
}

func textResult(output []byte) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(output)}}}
}

func envHandler(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	output, err := exec.CommandContext(ctx, "g", "env").Output()
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return textResult(output), nil, nil
}

func cleanHandler(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	output, err := exec.CommandContext(ctx, "g", "clean").Output()
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if len(output) == 0 {
		output = []byte("Done")
	}
	return textResult(output), nil, nil
}

type UseReq struct {
	Version string `json:"version" jsonschema:"The specific go sdk version number(e.g. '1.21.4')."`
}

func useHandler(ctx context.Context, _ *mcp.CallToolRequest, req UseReq) (*mcp.CallToolResult, any, error) {
	output, err := exec.CommandContext(ctx, "g", "use", req.Version).Output()
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return textResult(output), nil, nil
}

type LsReq struct {
	Output string `json:"output,omitempty" jsonschema:"Output format. Optional values: json, text"`
}

func lsHandler(ctx context.Context, _ *mcp.CallToolRequest, req LsReq) (*mcp.CallToolResult, any, error) {
	switch req.Output {
	case "json", "text":
	default:
		req.Output = "json"
	}
	output, err := exec.CommandContext(ctx, "g", "ls", "-o", req.Output).Output()
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return textResult(output), nil, nil
}

type LsRemoteReq struct {
	Version string `json:"version,omitempty" jsonschema:"Go sdk version number keywords. Optional values include: stable, unstable, archived, or a version pattern. Supported patterns: specific version (e.g. '1.21.4'), latest, wildcards (e.g. '1.21.x', '1.x', '1.18.*'), caret ranges (e.g. '^1', '^1.18', '^1.18.10'), tilde ranges (e.g. '~1.18'), comparisons (e.g. '>1.18', '<1.16'), version ranges (e.g. '1.18-1.20')."`
	Output  string `json:"output,omitempty" jsonschema:"Output format. Optional values: json, text"`
}

func lsRemoteHandler(ctx context.Context, _ *mcp.CallToolRequest, req LsRemoteReq) (*mcp.CallToolResult, any, error) {
	switch req.Output {
	case "json", "text":
	default:
		req.Output = "json"
	}
	output, err := exec.CommandContext(ctx, "g", "ls-remote", "-o", req.Output, req.Version).Output()
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	// Text output is already formatted by the CLI and must not be decoded as JSON.
	if req.Output == "text" {
		return textResult(output), nil, nil
	}
	var items []struct {
		Version   string `json:"version"`
		InUse     bool   `json:"inUse"`
		Installed bool   `json:"installed"`
	}
	if err = json.Unmarshal(output, &items); err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if output, err = json.Marshal(items); err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return textResult(output), nil, nil
}

type InstallReq struct {
	Version      string `json:"version" jsonschema:"Go sdk version number keywords. Supported patterns: specific version (e.g. '1.21.4'), latest, wildcards (e.g. '1.21.x', '1.x', '1.18.*'), caret ranges (e.g. '^1', '^1.18', '^1.18.10'), tilde ranges (e.g. '~1.18'), comparisons (e.g. '>1.18', '<1.16'), version ranges (e.g. '1.18-1.20')."`
	Nouse        bool   `json:"nouse,omitempty" jsonschema:"Don't use the version after installed."`
	SkipChecksum bool   `json:"skip-checksum,omitempty" jsonschema:"Skip checksum verification."`
}

func installHandler(ctx context.Context, _ *mcp.CallToolRequest, req InstallReq) (*mcp.CallToolResult, any, error) {
	output, err := exec.CommandContext(ctx, "g", "install", fmt.Sprintf("--nouse=%t", req.Nouse), fmt.Sprintf("--skip-checksum=%t", req.SkipChecksum), req.Version).CombinedOutput()
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	if !strings.Contains(string(output), "installed") {
		if output, err = exec.CommandContext(ctx, "go", "version").Output(); err != nil {
			return nil, nil, errors.WithStack(err)
		}
	}
	return textResult(output), nil, nil
}

type UninstallReq struct {
	Version string `json:"version" jsonschema:"The specific go sdk version number(e.g. '1.21.4')."`
}

func uninstallHandler(ctx context.Context, _ *mcp.CallToolRequest, req UninstallReq) (*mcp.CallToolResult, any, error) {
	output, err := exec.CommandContext(ctx, "g", "uninstall", req.Version).CombinedOutput()
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return textResult(output), nil, nil
}
