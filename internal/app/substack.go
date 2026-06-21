package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

const substackPageSize = 12

type SubstackPost struct {
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle"`
	Slug         string `json:"slug"`
	PostDate     string `json:"post_date"`
	Audience     string `json:"audience"`
	CanonicalURL string `json:"canonical_url"`
	BodyHTML     string `json:"body_html"`
	WordCount    int    `json:"wordcount"`
}

type SubstackClient struct {
	baseURL   *url.URL
	http      *http.Client
	cookie    string
	userAgent string
}

func NewSubstackClient(archiveURL, cookie string) (*SubstackClient, error) {
	u, err := url.Parse(archiveURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid archive URL %q", archiveURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("archive URL must use http or https")
	}

	return &SubstackClient{
		baseURL:   &url.URL{Scheme: u.Scheme, Host: u.Host},
		http:      &http.Client{Timeout: 30 * time.Second},
		cookie:    strings.TrimSpace(cookie),
		userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/146.0.0.0 Safari/537.36",
	}, nil
}

func (c *SubstackClient) requestJSON(path string, query url.Values, dst any) error {
	u := c.baseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned %s", u, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode %s: %w", u, err)
	}
	return nil
}

func (c *SubstackClient) Archive() ([]SubstackPost, error) {
	var posts []SubstackPost
	seen := make(map[string]bool)

	for offset := 0; ; offset += substackPageSize {
		var page []SubstackPost
		query := url.Values{
			"sort":   {"new"},
			"search": {""},
			"offset": {fmt.Sprint(offset)},
			"limit":  {fmt.Sprint(substackPageSize)},
		}
		if err := c.requestJSON("/api/v1/archive", query, &page); err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		for _, post := range page {
			if post.Slug != "" && !seen[post.Slug] {
				seen[post.Slug] = true
				posts = append(posts, post)
			}
		}
		if len(page) < substackPageSize {
			break
		}
	}
	return posts, nil
}

func (c *SubstackClient) FetchPost(slug string) (SubstackPost, error) {
	var post SubstackPost
	path := "/api/v1/posts/" + url.PathEscape(slug)
	if err := c.requestJSON(path, url.Values{"referrer": {""}}, &post); err != nil {
		return post, err
	}
	if strings.TrimSpace(post.BodyHTML) == "" {
		return post, fmt.Errorf("post %q returned no body_html; the session may not have access", slug)
	}
	return post, nil
}

func (c *SubstackClient) WriteMarkdown(post SubstackPost, outputDir string) (string, error) {
	markdown, err := htmltomarkdown.ConvertString(post.BodyHTML, converter.WithDomain(c.baseURL.String()))
	if err != nil {
		return "", fmt.Errorf("convert %q to Markdown: %w", post.Slug, err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	filename := sanitizeSlug(post.Slug, post.Title) + ".md"
	path := filepath.Join(outputDir, filename)
	content := buildMarkdownDocument(post, markdown)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func buildMarkdownDocument(post SubstackPost, body string) string {
	quote := func(value string) string {
		data, _ := json.Marshal(value)
		return string(data)
	}

	var out strings.Builder
	out.WriteString("---\n")
	out.WriteString("title: " + quote(post.Title) + "\n")
	if post.Subtitle != "" {
		out.WriteString("subtitle: " + quote(post.Subtitle) + "\n")
	}
	out.WriteString("url: " + quote(post.CanonicalURL) + "\n")
	out.WriteString("published: " + quote(post.PostDate) + "\n")
	out.WriteString("audience: " + quote(post.Audience) + "\n")
	fmt.Fprintf(&out, "word_count: %d\n", post.WordCount)
	out.WriteString("---\n\n")
	out.WriteString("# " + post.Title + "\n\n")
	if post.Subtitle != "" {
		out.WriteString("_" + post.Subtitle + "_\n\n")
	}
	out.WriteString(strings.TrimSpace(body))
	out.WriteString("\n")
	return out.String()
}
