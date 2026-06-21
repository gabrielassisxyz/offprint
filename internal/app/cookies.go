package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type CookieMap map[string]map[string]string

var domainCookies map[string][]*http.Cookie

func defaultCookiesFile() string {
	return filepath.Join(defaultConfigDir(), "cookies.json")
}

func handleCookieCommand(args []string) error {
	if len(args) < 1 || args[0] != "set" {
		return fmt.Errorf("usage: offprint cookies set --domain <domain> [--from-env NAME | --file PATH | --value STRING]")
	}

	setCmd := flag.NewFlagSet("cookies set", flag.ContinueOnError)
	domain := setCmd.String("domain", "", "domain that receives the cookies")
	fromEnv := setCmd.String("from-env", "OFFPRINT_COOKIE", "environment variable containing the Cookie header")
	fromFile := setCmd.String("file", "", "file containing the Cookie header")
	value := setCmd.String("value", "", "Cookie header (visible in shell history; prefer --from-env)")
	store := setCmd.String("store", defaultCookiesFile(), "cookie store path")

	if err := setCmd.Parse(args[1:]); err != nil {
		return err
	}

	if strings.TrimSpace(*domain) == "" {
		return fmt.Errorf("--domain is required; example: offprint cookies set --domain example.com")
	}

	cookieStr := strings.TrimSpace(*value)
	if *fromFile != "" {
		data, err := os.ReadFile(*fromFile)
		if err != nil {
			return fmt.Errorf("read cookie file: %w", err)
		}
		cookieStr = strings.TrimSpace(string(data))
	} else if envValue := strings.TrimSpace(os.Getenv(*fromEnv)); envValue != "" {
		cookieStr = envValue
	} else if cookieStr == "" {
		stat, err := os.Stdin.Stat()
		if err == nil && stat.Mode()&os.ModeCharDevice == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read cookies from stdin: %w", err)
			}
			cookieStr = strings.TrimSpace(string(data))
		}
	}
	if cookieStr == "" {
		return fmt.Errorf("no cookies provided; set %s, use --file, or pipe the Cookie header on stdin", *fromEnv)
	}

	count, err := saveCookies(*store, *domain, cookieStr)
	if err != nil {
		return err
	}
	fmt.Printf("Saved %d cookies for %s in %s\n", count, *domain, *store)
	return nil
}

func saveCookies(storePath, domain, cookieStr string) (int, error) {
	var cmap CookieMap
	data, err := os.ReadFile(storePath)
	if err == nil {
		if err := json.Unmarshal(data, &cmap); err != nil {
			return 0, fmt.Errorf("parse cookie store: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("read cookie store: %w", err)
	}

	if cmap == nil {
		cmap = make(CookieMap)
	}
	if cmap[domain] == nil {
		cmap[domain] = make(map[string]string)
	}

	parts := strings.Split(cookieStr, ";")
	count := 0
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			cmap[domain][strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			count++
		}
	}

	out, err := json.MarshalIndent(cmap, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(storePath), 0o700); err != nil {
		return 0, err
	}
	if err := os.WriteFile(storePath, out, 0o600); err != nil {
		return 0, err
	}
	return count, nil
}

func loadCookies(storePath string) error {
	domainCookies = make(map[string][]*http.Cookie)
	data, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read cookie store: %w", err)
	}

	var cmap CookieMap
	if err := json.Unmarshal(data, &cmap); err != nil {
		return fmt.Errorf("parse cookie store: %w", err)
	}

	for domain, cookies := range cmap {
		for name, value := range cookies {
			domainCookies[domain] = append(domainCookies[domain], &http.Cookie{
				Name:  name,
				Value: value,
			})
		}
	}
	return nil
}

func getCookiesForURL(targetURL string) []*http.Cookie {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil
	}

	host := parsed.Hostname()
	var matchedCookies []*http.Cookie

	for domain, cookies := range domainCookies {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			matchedCookies = append(matchedCookies, cookies...)
		}
	}
	return matchedCookies
}
