package github

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
)

type releaseTransport func(*http.Request) (*http.Response, error)

func (f releaseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPrereleasePaginationAndVersionOrder(t *testing.T) {
	old := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = old })
	calls := 0
	http.DefaultClient = &http.Client{Transport: releaseTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		header := make(http.Header)
		body := `[{"tag_name":"v1.8.1-dev.2"},{"tag_name":"v99.0.0","draft":true},{"tag_name":"invalid"}]`
		if calls == 1 {
			header.Set("Link", `<https://api.github.com/repos/test/g/releases?per_page=100&page=2>; rel="next"`)
		} else {
			body = `[{"tag_name":"v1.8.1-dev.10"},{"tag_name":"v1.8.1-dev.3"}]`
		}
		if r.URL.Query().Get("page") != string(rune('0'+calls)) {
			t.Errorf("unexpected page %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	rel, yes, err := NewReleaseUpdater().CheckForUpdatesIncludingPrereleases(semver.MustParse("1.8.1-dev.1"), "test", "g")
	if err != nil || !yes || rel == nil || rel.TagName != "v1.8.1-dev.10" || calls != 2 {
		t.Fatalf("got %+v, %v, %v; calls %d", rel, yes, err, calls)
	}
}

func TestPrereleaseStableAndNoDowngrade(t *testing.T) {
	old := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = old })
	http.DefaultClient = &http.Client{Transport: releaseTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"tag_name":"v1.8.1-dev.10"},{"tag_name":"v1.8.1"}]`))}, nil
	})}
	for _, current := range []string{"1.8.1-dev.1", "1.8.1", "1.9.0"} {
		rel, yes, err := NewReleaseUpdater().CheckForUpdatesIncludingPrereleases(semver.MustParse(current), "test", "g")
		if err != nil {
			t.Fatal(err)
		}
		if current == "1.8.1-dev.1" {
			if !yes || rel.TagName != "v1.8.1" {
				t.Fatalf("stable version not selected: %+v", rel)
			}
		} else if yes || rel != nil {
			t.Fatalf("downgrade selected for %s", current)
		}
	}
}
