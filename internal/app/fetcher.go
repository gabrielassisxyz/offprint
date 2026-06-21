package app

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func fetchPageTitle(linkURL string) string {
	hash := md5.Sum([]byte(linkURL))
	safeName := hex.EncodeToString(hash[:]) + ".txt"
	cachePath := filepath.Join(CacheDirTitles, safeName)

	if cachedData, err := os.ReadFile(cachePath); err == nil {
		return string(cachedData)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", linkURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	for _, c := range getCookiesForURL(linkURL) {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ""
	}

	title := strings.TrimSpace(doc.Find("title").Text())
	_ = os.WriteFile(cachePath, []byte(title), 0o644)

	return title
}

func fetchAndCleanHTML(baseURLStr string, ignoreList []string, urlToFootnote map[string]int, allReferences *[]Reference, footnoteCounter *int) (string, string, string, DomainConfig, error) {
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return "", "", "", DomainConfig{}, fmt.Errorf("invalid URL")
	}

	cfg := getDomainConfig(baseURLStr)

	safeCacheName := sanitizeFilename(baseURLStr) + ".html"
	cachePath := filepath.Join(CacheDirHTML, safeCacheName)
	var rawHTML string

	if cachedData, err := os.ReadFile(cachePath); err == nil {
		rawHTML = string(cachedData)
	} else {
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequest("GET", baseURLStr, nil)
		if err != nil {
			return "", "", "", cfg, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		for _, c := range getCookiesForURL(baseURLStr) {
			req.AddCookie(c)
		}

		res, err := client.Do(req)
		if err != nil {
			return "", "", "", cfg, err
		}
		defer func() { _ = res.Body.Close() }()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			return "", "", "", cfg, err
		}

		rawHTML = string(data)
		_ = os.WriteFile(cachePath, data, 0o644)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return "", "", "", cfg, err
	}

	title := ""
	if cfg.TitleSelector != "" {
		title = doc.Find(cfg.TitleSelector).First().Text()
	}
	if title == "" {
		title = doc.Find("title").Text() // Fallback
	}
	title = cleanTitle(strings.TrimSpace(title), ignoreList)
	if title == "" {
		title = "Untitled Article"
	}

	subtitle := ""
	if cfg.SubtitleSelector != "" {
		subtitle = strings.TrimSpace(doc.Find(cfg.SubtitleSelector).First().Text())
	}

	var articleContainer *goquery.Selection
	if cfg.BodySelector != "" {
		articleContainer = doc.Find(cfg.BodySelector).First()
	} else {
		// Fallback for domains without config
		articleContainer = doc.Find(".inside-article")
		if articleContainer.Length() == 0 {
			articleContainer = doc.Find("article")
			if articleContainer.Length() == 0 {
				articleContainer = doc.Find("body")
			}
		}
	}

	if len(cfg.RemoveSelectors) > 0 {
		for _, sel := range cfg.RemoveSelectors {
			articleContainer.Find(sel).Remove()
		}
	} else {
		// Fallback for domains without config
		articleContainer.Children().Filter("div").Eq(1).Remove()
		articleContainer.Find("script, style, iframe, nav, footer, header").Remove()
	}

	contentSelection := articleContainer.Find(".entry-content")
	if contentSelection.Length() == 0 {
		contentSelection = articleContainer
	}

	// Apply dynamic structural transformations based on domains.json
	for _, transform := range cfg.Transformations {
		contentSelection.Find(transform.TargetSelector).Each(func(i int, s *goquery.Selection) {
			finalHTML := transform.ReplacementHTML

			// Extract each mapped variable
			for varName, extRule := range transform.Extractors {
				var extractedValue string

				// Find the specific node to extract from
				targetNode := s
				if extRule.Selector != "" {
					targetNode = s.Find(extRule.Selector).First()
				}

				// Get either the text or the attribute
				if extRule.Attr == "text" {
					extractedValue = strings.TrimSpace(targetNode.Text())
				} else {
					extractedValue, _ = targetNode.Attr(extRule.Attr)
				}

				// NEW: Apply Regex if defined
				if extRule.Regex != "" {
					re, err := regexp.Compile(extRule.Regex)
					if err == nil {
						matches := re.FindStringSubmatch(extractedValue)
						if len(matches) > 1 {
							// Use the first capture group (inside parentheses)
							extractedValue = strings.TrimSpace(matches[1])
						} else if len(matches) == 1 {
							// Use the full match if no capture groups were defined
							extractedValue = strings.TrimSpace(matches[0])
						} else {
							extractedValue = "" // Regex failed to find a match
						}
					}
				}

				// Replace the placeholder {VarName} in the final HTML template
				placeholder := "{" + varName + "}"
				finalHTML = strings.ReplaceAll(finalHTML, placeholder, extractedValue)
			}

			// Replace the bloated DOM element with the clean HTML
			s.ReplaceWithHtml(finalHTML)
		})
	}

	contentSelection.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		href = strings.TrimSpace(href)
		if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			return
		}

		u, err := url.Parse(href)
		if err == nil {
			href = baseURL.ResolveReference(u).String()
		}

		ext := strings.ToLower(filepath.Ext(u.Path))
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" {
			return
		}

		refNum, exists := urlToFootnote[href]
		if !exists {
			refNum = *footnoteCounter
			urlToFootnote[href] = refNum
			*footnoteCounter++

			linkTitle := fetchPageTitle(href)
			if linkTitle == "" {
				linkTitle = strings.TrimSpace(s.Text())
				if linkTitle == "" {
					linkTitle = "Link"
				}
			}

			isAmazon := strings.Contains(strings.ToLower(href), "amazon.com") || strings.Contains(strings.ToLower(href), "amzn.to")
			hideURL := false

			if isAmazon {
				linkTitle = strings.ReplaceAll(linkTitle, "Amazon.com: ", "")
				linkTitle = strings.ReplaceAll(linkTitle, "Amazon.com.br: ", "")
				linkTitle = strings.TrimSuffix(linkTitle, ": Books")
				linkTitle = strings.TrimSuffix(linkTitle, ": Livros")
				linkTitle = strings.TrimSpace(linkTitle)

				lowerTitle := strings.ToLower(linkTitle)
				if lowerTitle == "amazon.com" || lowerTitle == "amazon" || lowerTitle == "" {
					fallbackText := strings.TrimSpace(s.Text())
					if fallbackText != "" {
						linkTitle = fallbackText
					} else {
						linkTitle = "Amazon"
					}
					hideURL = false // Force display URL to not lose reference
				} else {
					hideURL = true // Hide URL if title is perfectly readable
				}
			}

			linkTitle = cleanTitle(linkTitle, ignoreList)

			displayURL := href
			maxURLLen := 65
			if len(displayURL) > maxURLLen {
				displayURL = displayURL[:maxURLLen-3] + "..."
			}

			*allReferences = append(*allReferences, Reference{
				Number:     refNum,
				Title:      linkTitle,
				URL:        href,
				DisplayURL: displayURL,
				HideURL:    hideURL,
			})
		}

		s.AppendHtml(fmt.Sprintf(`<sup style="font-size: 0.75em; margin-left: 2px;">[%d]</sup>`, refNum))
	})

	contentSelection.Find("img").Each(func(i int, img *goquery.Selection) {
		possibleAttrs := []string{"data-lazy-src", "data-src", "src"}
		var targetURL string
		for _, attr := range possibleAttrs {
			if val, exists := img.Attr(attr); exists && val != "" {
				targetURL = val
				break
			}
		}
		if targetURL == "" || strings.HasPrefix(targetURL, "data:") {
			return
		}

		u, _ := url.Parse(targetURL)
		absURL := baseURL.ResolveReference(u).String()
		base64Data, err := downloadImageAsBase64(absURL)
		if err == nil {
			img.SetAttr("src", base64Data)
			img.RemoveAttr("srcset")
			img.RemoveAttr("loading")
			img.RemoveAttr("width")
			img.RemoveAttr("height")
		}
	})

	html, _ := contentSelection.Html()
	return html, title, subtitle, cfg, nil
}

func downloadImageAsBase64(imgURL string) (string, error) {
	hash := md5.Sum([]byte(imgURL))
	safeName := hex.EncodeToString(hash[:]) + ".txt"
	cachePath := filepath.Join(CacheDirImages, safeName)

	if cachedData, err := os.ReadFile(cachePath); err == nil {
		return string(cachedData), nil
	}

	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", imgURL, nil)
	if err != nil {
		return "", err
	}

	for _, c := range getCookiesForURL(imgURL) {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	mimeType := http.DetectContentType(data)
	base64Str := base64.StdEncoding.EncodeToString(data)
	result := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str)

	_ = os.WriteFile(cachePath, []byte(result), 0o644)

	return result, nil
}
