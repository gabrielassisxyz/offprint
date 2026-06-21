package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubstackArchivePaginationAndCookie(t *testing.T) {
	const cookie = "connect.sid=paid-session; other=value"
	posts := make([]SubstackPost, 13)
	for i := range posts {
		posts[i] = SubstackPost{
			Title:    fmt.Sprintf("Post %d", i),
			Slug:     fmt.Sprintf("post-%d", i),
			Audience: "only_paid",
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got != cookie {
			t.Errorf("Cookie header = %q, want %q", got, cookie)
		}
		if r.URL.Path != "/api/v1/archive" {
			http.NotFound(w, r)
			return
		}
		offset := 0
		if _, err := fmt.Sscan(r.URL.Query().Get("offset"), &offset); err != nil {
			t.Errorf("parse offset: %v", err)
		}
		end := min(offset+substackPageSize, len(posts))
		page := []SubstackPost{}
		if offset < len(posts) {
			page = posts[offset:end]
		}
		if err := json.NewEncoder(w).Encode(page); err != nil {
			t.Errorf("encode archive response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewSubstackClient(server.URL+"/archive", cookie)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Archive()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(posts) {
		t.Fatalf("Archive returned %d posts, want %d", len(got), len(posts))
	}
}

func TestFetchPostAndWriteMarkdown(t *testing.T) {
	post := SubstackPost{
		Title:        "A title: with punctuation",
		Subtitle:     "A useful subtitle",
		Slug:         "a-title",
		PostDate:     "2026-06-21T12:00:00Z",
		Audience:     "only_paid",
		CanonicalURL: "https://example.com/p/a-title",
		BodyHTML:     `<p>Hello <strong>reader</strong>.</p><p><a href="/p/next">Next</a></p>`,
		WordCount:    4,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/posts/a-title" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(post); err != nil {
			t.Errorf("encode post response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewSubstackClient(server.URL+"/archive", "session=test")
	if err != nil {
		t.Fatal(err)
	}
	fetched, err := client.FetchPost("a-title")
	if err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	path, err := client.WriteMarkdown(fetched, outDir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	markdown := string(data)
	for _, want := range []string{
		`title: "A title: with punctuation"`,
		`audience: "only_paid"`,
		"# A title: with punctuation",
		"Hello **reader**.",
		fmt.Sprintf("[Next](%s/p/next)", server.URL),
	} {
		if !strings.Contains(markdown, want) {
			t.Errorf("Markdown does not contain %q:\n%s", want, markdown)
		}
	}
	if filepath.Base(path) != "a-title.md" {
		t.Errorf("output path = %q", path)
	}
}

func TestFetchPostRejectsEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(SubstackPost{Slug: "locked", Audience: "only_paid"}); err != nil {
			t.Errorf("encode locked response: %v", err)
		}
	}))
	defer server.Close()

	client, _ := NewSubstackClient(server.URL+"/archive", "session=test")
	_, err := client.FetchPost("locked")
	if err == nil || !strings.Contains(err.Error(), "no body_html") {
		t.Fatalf("FetchPost error = %v, want missing body error", err)
	}
}

func TestNewSubstackClientRejectsInvalidURL(t *testing.T) {
	for _, raw := range []string{"", "not-a-url", "file:///tmp/archive"} {
		t.Run(url.PathEscape(raw), func(t *testing.T) {
			if _, err := NewSubstackClient(raw, "cookie=value"); err == nil {
				t.Fatalf("NewSubstackClient(%q) unexpectedly succeeded", raw)
			}
		})
	}
}

func TestWriteMarkdownSanitizesSlug(t *testing.T) {
	client, err := NewSubstackClient("https://example.com/archive", "")
	if err != nil {
		t.Fatal(err)
	}
	path, err := client.WriteMarkdown(SubstackPost{
		Title:    "Safe title",
		Slug:     "../../Escape Me?!",
		BodyHTML: "<p>Body</p>",
	}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filepath.Base(path), "escape-me.md"; got != want {
		t.Fatalf("filename = %q, want %q", got, want)
	}
}
