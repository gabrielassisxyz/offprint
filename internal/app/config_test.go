package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDomainConfigsLoadsUserProfilesAndRelativeCSS(t *testing.T) {
	configDir := t.TempDir()
	sitesDir := filepath.Join(configDir, "sites")
	stylesDir := filepath.Join(configDir, "styles")
	if err := os.MkdirAll(sitesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stylesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cssPath := filepath.Join(stylesDir, "personal.css")
	if err := os.WriteFile(cssPath, []byte(".content { color: rebeccapurple; }"), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := `{
		"example.com": {
			"body_selector": "main article",
			"custom_css_path": "../styles/personal.css"
		}
	}`
	if err := os.WriteFile(filepath.Join(sitesDir, "example.json"), []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := loadDomainConfigs(configDir, ""); err != nil {
		t.Fatal(err)
	}
	got := getDomainConfig("https://notes.example.com/post")
	if got.BodySelector != "main article" {
		t.Fatalf("body selector = %q", got.BodySelector)
	}
	if !strings.Contains(got.CustomCSS, "rebeccapurple") {
		t.Fatalf("custom CSS was not loaded: %q", got.CustomCSS)
	}
	if got.CustomCSSPath != cssPath {
		t.Fatalf("CSS path = %q, want %q", got.CustomCSSPath, cssPath)
	}
}

func TestConfigureDocumentCSSAddsUserOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "print.css")
	if err := os.WriteFile(path, []byte("body { color: navy; }"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := configureDocumentCSS(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = configureDocumentCSS("") })

	css, _ := getBaseStyles(12, 1)
	if !strings.Contains(css, "body { color: navy; }") {
		t.Fatalf("base styles do not contain custom CSS: %s", css)
	}
}

func TestTransformationCanExtractInnerHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>Example</title></head><body><article><div class="embed"><span>kept</span></div></article></body></html>`))
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	previousConfigs := domainConfigs
	previousHTMLCache := CacheDirHTML
	previousTitleCache := CacheDirTitles
	domainConfigs = map[string]DomainConfig{
		parsed.Hostname(): {
			BodySelector: "article",
			Transformations: []TransformRule{{
				TargetSelector: ".embed",
				Extractors: map[string]ExtractRule{
					"Content": {Attr: "html"},
				},
				ReplacementHTML: `<section>{Content}</section>`,
			}},
		},
	}
	CacheDirHTML = filepath.Join(t.TempDir(), "html")
	CacheDirTitles = filepath.Join(t.TempDir(), "titles")
	if err := os.MkdirAll(CacheDirHTML, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		domainConfigs = previousConfigs
		CacheDirHTML = previousHTMLCache
		CacheDirTitles = previousTitleCache
	})

	html, _, _, _, err := fetchAndCleanHTML(server.URL, nil, map[string]int{}, &[]Reference{}, new(int))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `<section><span>kept</span></section>`) {
		t.Fatalf("transformed HTML = %q", html)
	}
}
