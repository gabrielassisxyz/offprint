package app

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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
	CustomCSS        string          `json:"-"`
}

var domainConfigs map[string]DomainConfig

func loadDomainConfigs(configDir, explicitPath string) error {
	domainConfigs = make(map[string]DomainConfig)
	if err := mergeDomainConfigs(assets.SitesJSON, ""); err != nil {
		return fmt.Errorf("parse built-in site profiles: %w", err)
	}

	paths, err := filepath.Glob(filepath.Join(configDir, "sites", "*.json"))
	if err != nil {
		return fmt.Errorf("find user site profiles: %w", err)
	}
	sort.Strings(paths)
	if explicitPath != "" {
		paths = append(paths, explicitPath)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read site profile %q: %w", path, err)
		}
		if err := mergeDomainConfigs(data, path); err != nil {
			return fmt.Errorf("parse site profile %q: %w", path, err)
		}
	}
	return nil
}

func mergeDomainConfigs(data []byte, sourcePath string) error {
	var profiles map[string]DomainConfig
	if err := json.Unmarshal(data, &profiles); err != nil {
		return err
	}
	for domain, profile := range profiles {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			return fmt.Errorf("site profile contains an empty domain")
		}
		if profile.CustomCSSPath != "" {
			cssPath := profile.CustomCSSPath
			if !filepath.IsAbs(cssPath) {
				if sourcePath == "" {
					return fmt.Errorf("built-in profile %q uses a non-embedded CSS path", domain)
				}
				cssPath = filepath.Join(filepath.Dir(sourcePath), cssPath)
			}
			css, err := os.ReadFile(cssPath)
			if err != nil {
				return fmt.Errorf("read CSS for %q from %q: %w", domain, cssPath, err)
			}
			profile.CustomCSSPath = filepath.Clean(cssPath)
			profile.CustomCSS = string(css)
		}
		domainConfigs[domain] = profile
	}
	return nil
}

func getDomainConfig(targetURL string) DomainConfig {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return DomainConfig{}
	}
	host := strings.ToLower(parsed.Hostname())

	bestDomain := ""
	bestConfig := DomainConfig{}
	for domain, config := range domainConfigs {
		if (host == domain || strings.HasSuffix(host, "."+domain)) && len(domain) > len(bestDomain) {
			bestDomain = domain
			bestConfig = config
		}
	}
	return bestConfig
}
