package app

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gabrielassisxyz/offprint/internal/assets"
)

// ExtractRule defines how to extract text or an attribute from an HTML element
type ExtractRule struct {
	Selector string `json:"selector"` // CSS selector relative to the target element
	Attr     string `json:"attr"`     // "text" to get text content, or attribute name like "href"
	Regex    string `json:"regex"`
}

// TransformRule defines how to completely rewrite an HTML block
type TransformRule struct {
	TargetSelector  string                 `json:"target_selector"`  // The clunky element to replace
	Extractors      map[string]ExtractRule `json:"extractors"`       // Data to extract (e.g., "Url": {...})
	ReplacementHTML string                 `json:"replacement_html"` // The new HTML (use {VarName} placeholders)
}

// DomainConfig holds the custom scraping and styling rules for a specific website
type DomainConfig struct {
	TitleSelector    string          `json:"title_selector"`
	SubtitleSelector string          `json:"subtitle_selector"`
	BodySelector     string          `json:"body_selector"`
	RemoveSelectors  []string        `json:"remove_selectors"`
	SourcePosition   string          `json:"source_position"`
	TitleAlign       string          `json:"title_align"`
	SubtitleAlign    string          `json:"subtitle_align"`
	ShowSeparator    bool            `json:"show_separator"`
	CustomCSSPath    string          `json:"custom_css_path"`
	Transformations  []TransformRule `json:"transformations"` // NEW: Dynamic structural rewrites
}

var domainConfigs map[string]DomainConfig

func loadDomainConfigs(path string) error {
	domainConfigs = make(map[string]DomainConfig)
	data := assets.DomainsJSON
	if path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read domain configuration %q: %w", path, err)
		}
	}
	if err := json.Unmarshal(data, &domainConfigs); err != nil {
		return fmt.Errorf("parse domain configuration: %w", err)
	}
	return nil
}

func getDomainConfig(targetURL string) DomainConfig {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return DomainConfig{}
	}
	host := parsed.Hostname()

	for domain, config := range domainConfigs {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return config
		}
	}
	return DomainConfig{} // Return empty if no custom config is found
}
