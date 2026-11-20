package cli

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
	"github.com/voidint/g/build"
)

type updateRoundTripper func(*http.Request) (*http.Response, error)

func (f updateRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSelfUpdateRepository(t *testing.T) {
	originalRepo, originalClient := build.UpdateRepo, http.DefaultClient
	t.Cleanup(func() { build.UpdateRepo, http.DefaultClient = originalRepo, originalClient })
	for _, tt := range []struct {
		name, defaultRepo, override, wantRepo, tag string
		legacy, invalid                            bool
	}{
		{name: "upstream default", defaultRepo: "voidint/g", wantRepo: "voidint/g", tag: "v" + build.ShortVersion},
		{name: "fork build", defaultRepo: "zmj159809/g", wantRepo: "zmj159809/g", tag: "v" + build.ShortVersion},
		{name: "override upstream no downgrade", defaultRepo: "zmj159809/g", override: "voidint/g", wantRepo: "voidint/g", tag: "v1.0.0"},
		{name: "override fork", defaultRepo: "voidint/g", override: "zmj159809/g", wantRepo: "zmj159809/g", tag: "v" + build.ShortVersion},
		{name: "legacy command", defaultRepo: "voidint/g", override: "zmj159809/g", wantRepo: "zmj159809/g", tag: "v" + build.ShortVersion, legacy: true},
		{name: "reject URL", defaultRepo: "voidint/g", override: "https://github.com/voidint/g", invalid: true},
		{name: "reject traversal", defaultRepo: "voidint/g", override: "owner/..", invalid: true},
		{name: "reject invalid build default", defaultRepo: "owner/repo/extra", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			build.UpdateRepo = tt.defaultRepo
			calls := 0
			http.DefaultClient = &http.Client{Transport: updateRoundTripper(func(r *http.Request) (*http.Response, error) {
				calls++
				wantURL := "https://api.github.com/repos/" + tt.wantRepo + "/releases/latest"
				if r.URL.String() != wantURL {
					t.Errorf("request URL = %s, want %s", r.URL, wantURL)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"tag_name":%q,"assets":[]}`, tt.tag)))}, nil
			})}
			app := &cli.App{
				Commands: commands, Writer: io.Discard, ErrWriter: io.Discard,
				ExitErrHandler: func(*cli.Context, error) {},
			}
			args := []string{"g", "self", "update"}
			if tt.legacy {
				args = []string{"g", "update"}
			}
			if tt.override != "" {
				args = append(args, "--repo", tt.override)
			}
			err := app.Run(args)
			if tt.invalid {
				if err == nil || !strings.Contains(err.Error(), "expected owner/repo") {
					t.Fatalf("expected repository validation error, got %v", err)
				}
				if calls != 0 {
					t.Errorf("invalid repository made %d HTTP calls", calls)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if calls != 1 {
					t.Errorf("HTTP calls = %d, want 1 (no download for same/older release)", calls)
				}
			}
			if build.UpdateRepo != tt.defaultRepo {
				t.Error("override changed build default")
			}
		})
	}
}

func TestParseUpdateRepo(t *testing.T) {
	for _, repo := range []string{"", "owner", "/repo", "owner/", "owner/.", "owner/..", "owner/repo?x=y", " owner/repo", "owner/repo/extra"} {
		if _, _, err := parseUpdateRepo(repo); err == nil {
			t.Errorf("accepted %q", repo)
		}
	}
	for _, repo := range []string{"voidint/g", "zmj159809/g", "some-owner/repo.name_1"} {
		owner, name, err := parseUpdateRepo(repo)
		if err != nil || owner+"/"+name != repo {
			t.Errorf("parse %q = %q/%q, %v", repo, owner, name, err)
		}
	}
}
