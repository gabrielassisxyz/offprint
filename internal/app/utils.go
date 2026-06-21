package app

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func parseInputFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	return urls, scanner.Err()
}

func buildIgnoreList(ignoreStr string) []string {
	var ignoreList []string
	if ignoreStr != "" {
		parts := strings.Split(ignoreStr, ",")
		for _, p := range parts {
			cleanP := strings.TrimSpace(strings.Trim(p, `"'`))
			if cleanP != "" {
				ignoreList = append(ignoreList, cleanP)
			}
		}
	}
	return ignoreList
}

func cleanTitle(title string, ignoreList []string) string {
	for _, ignore := range ignoreList {
		title = strings.ReplaceAll(title, ignore, "")
	}
	return strings.TrimSpace(title)
}

func sanitizeFilename(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`)
	clean := re.ReplaceAllString(name, "_")
	if len(clean) > 50 {
		clean = clean[:50]
	}
	return clean
}

func defaultOutputDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join("output")
	}
	return filepath.Join(home, "Documents", "Offprint")
}

func defaultConfigDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "offprint")
	}
	return filepath.Join(".offprint")
}

func defaultCacheDir() string {
	if configured := strings.TrimSpace(os.Getenv("OFFPRINT_CACHE_DIR")); configured != "" {
		return configured
	}
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "offprint")
	}
	return filepath.Join(".cache", "offprint")
}

var unsafeSlugCharacters = regexp.MustCompile(`[^a-z0-9._-]+`)

func sanitizeSlug(slug, fallback string) string {
	clean := strings.ToLower(strings.TrimSpace(filepath.Base(slug)))
	clean = unsafeSlugCharacters.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, ".-_")
	if clean == "" {
		clean = strings.ToLower(strings.TrimSpace(fallback))
		clean = unsafeSlugCharacters.ReplaceAllString(clean, "-")
		clean = strings.Trim(clean, ".-_")
	}
	if clean == "" {
		clean = "untitled"
	}
	if len(clean) > 100 {
		clean = strings.TrimRight(clean[:100], ".-_")
	}
	return clean
}

func readURLInput(input string) ([]string, error) {
	if parsed, err := url.ParseRequestURI(input); err == nil &&
		(parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" {
		return []string{input}, nil
	}

	urls, err := parseInputFile(input)
	if err != nil {
		return nil, fmt.Errorf("input must be an HTTP(S) URL or a readable URL file: %w", err)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("URL file %q contains no URLs", input)
	}
	for line, raw := range urls {
		parsed, err := url.ParseRequestURI(raw)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, fmt.Errorf("invalid URL on content line %d: %q", line+1, raw)
		}
	}
	return urls, nil
}
