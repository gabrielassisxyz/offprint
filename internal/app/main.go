package app

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

type Entry struct {
	URL          string
	Title        string
	Subtitle     string
	HTMLContent  string
	PageCount    int
	StartPageNum int
	Config       DomainConfig
}

type Reference struct {
	Number     int
	Title      string
	URL        string
	DisplayURL string
	HideURL    bool
}

var (
	CacheDirHTML   = filepath.Join(defaultCacheDir(), "html")
	CacheDirImages = filepath.Join(defaultCacheDir(), "images")
	CacheDirTitles = filepath.Join(defaultCacheDir(), "titles")
)

// Main runs the Offprint command-line application.
func Main() {
	log.SetFlags(0)
	if len(os.Args) == 1 || (len(os.Args) == 2 && (os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h")) {
		printHelp()
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "archive" {
		if err := runArchiveCommand(os.Args[2:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return
			}
			log.Fatal(err)
		}
		return
	}

	if len(os.Args) >= 2 && (os.Args[1] == "cookie" || os.Args[1] == "cookies") {
		if err := handleCookieCommand(os.Args[2:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return
			}
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "version" {
		fmt.Println("offprint", Version)
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "render" {
		log.Fatal(`command "render" was renamed; use "offprint bundle" instead`)
	}
	if len(os.Args) >= 2 && os.Args[1] == "bundle" {
		cleanup, err := normalizeBundleCommand(os.Args[2:])
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return
			}
			log.Fatal(err)
		}
		defer cleanup()
	}

	// 1. Configuration and Flags
	inputFile := flag.String("i", "", "TXT file with links")
	archiveURL := flag.String("archive", "", "Substack /archive URL to export as Markdown")
	markdownDir := flag.String("markdown-dir", filepath.Join(defaultOutputDir(), "markdown"), "Directory for Markdown files")
	htmlFile := flag.String("h", "", "Path to existing HTML file to convert directly to PDF")
	outputFile := flag.String("o", "ebook.pdf", "Output PDF filename")
	outputDir := flag.String("output-dir", defaultOutputDir(), "Base output directory")
	outputFormat := flag.String("format", "both", "Output format: html, pdf, or both")
	fontPath := flag.String("font", "", "Optional TTF/OTF/WOFF font file")
	cssPath := flag.String("css", "", "Optional CSS applied to the complete document")
	configDir := flag.String("config-dir", defaultConfigDir(), "Offprint configuration directory")
	domainsFile := flag.String("domains-file", "", "Optional domain extraction configuration")
	disableWebSecurity := flag.Bool("disable-web-security", false, "Disable Chromium web security (unsafe; use only for trusted content)")
	ignoreStr := flag.String("ignore", "", "Comma-separated list to ignore in titles (e.g., \" - Wikipedia\")")
	cols := flag.Int("cols", 1, "Number of text columns (e.g., 1 or 2)")
	fontSize := flag.Int("fontsize", 11, "Text font size in px (e.g., 14)")
	flag.Parse()

	// Load stored cookies
	if err := loadCookies(defaultCookiesFile()); err != nil {
		log.Printf("warning: %v", err)
	}
	if err := loadDomainConfigs(*configDir, *domainsFile); err != nil {
		log.Fatal(err)
	}
	if err := configureFont(*fontPath); err != nil {
		log.Fatal(err)
	}
	if err := configureDocumentCSS(*cssPath); err != nil {
		log.Fatal(err)
	}
	if *outputFormat != "html" && *outputFormat != "pdf" && *outputFormat != "both" {
		log.Fatalf("invalid --format %q; use html, pdf, or both", *outputFormat)
	}

	if *archiveURL != "" {
		if err := exportSubstackArchive(*archiveURL, *markdownDir, os.Getenv("SUBSTACK_COOKIE")); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Create unified cache directories
	for _, cacheDir := range []string{CacheDirHTML, CacheDirImages, CacheDirTitles} {
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			log.Fatalf("create cache directory %q: %v", cacheDir, err)
		}
	}

	// Determine Output Directories and Filenames
	baseName := strings.TrimSuffix(filepath.Base(*outputFile), filepath.Ext(*outputFile))
	if baseName == "" {
		baseName = "ebook"
	}
	outDir := filepath.Join(*outputDir, baseName)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("create output directory %q: %v", outDir, err)
	}

	finalPDFPath := filepath.Join(outDir, baseName+".pdf")
	finalHTMLPath := filepath.Join(outDir, baseName+".html")

	// ---------------------------------------------------------
	// DIRECT MODE: Convert local HTML to PDF and exit
	// ---------------------------------------------------------
	if *htmlFile != "" {
		fmt.Printf("Direct Mode: Converting '%s' to '%s'...\n", *htmlFile, finalPDFPath)
		htmlBytes, err := os.ReadFile(*htmlFile)
		if err != nil {
			log.Fatalf("Error reading HTML file: %v", err)
		}

		ctx, cancel := createChromeContext(*disableWebSecurity)
		defer cancel()

		if err := printToPDF(ctx, string(htmlBytes), finalPDFPath, true); err != nil {
			log.Fatalf("Conversion error: %v", err)
		}
		color.Green("✓ PDF successfully generated at: %s", finalPDFPath)
		return
	}

	// ---------------------------------------------------------
	// STANDARD MODE: Batch Processing
	// ---------------------------------------------------------
	if *inputFile == "" {
		log.Fatal("Usage: go run . -i links.txt\n Or: go run . -h file.html\n Or: go run . cookie set -d domain.com -c \"...\"")
	}

	tempDir, err := os.MkdirTemp("", "ebook_parts")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	urls, err := parseInputFile(*inputFile)
	if err != nil {
		log.Fatalf("Error reading input file: %v", err)
	}

	ignoreList := buildIgnoreList(*ignoreStr)
	var entries []*Entry
	urlToFootnote := make(map[string]int)
	var allReferences []Reference
	footnoteCounter := 1

	// Phase 1: Download, Clean and Extract Footnotes
	fmt.Println("--- Phase 1: Downloading and Processing Content ---")
	for i, u := range urls {
		color.Blue("Downloading [%d/%d]: %s", i+1, len(urls), u)
		html, title, subtitle, cfg, err := fetchAndCleanHTML(u, ignoreList, urlToFootnote, &allReferences, &footnoteCounter)

		entry := &Entry{URL: u}
		if err != nil {
			color.Red("  └ Error: %v", err)
			entry.Title = "ERROR: " + u
			entry.HTMLContent = "<p>Error downloading this article.</p>"
		} else {
			entry.Title = title
			entry.Subtitle = subtitle
			entry.Config = cfg
			entry.HTMLContent = html
			color.Green("  └ ✓ OK: %s", title)
		}
		entries = append(entries, entry)
		time.Sleep(500 * time.Millisecond)
	}

	if len(allReferences) > 0 {
		entries = append(entries, &Entry{
			URL:         "",
			Title:       "References",
			HTMLContent: buildReferencesHTML(allReferences),
		})
		color.Magenta("\nReferences chapter generated with %d unique links.", len(allReferences))
	}

	// Phase 2: Page Counting
	fmt.Println("\n--- Phase 2: Calculating Article Pagination ---")
	if *outputFormat == "html" {
		currentPage := 1
		for _, entry := range entries {
			entry.StartPageNum = currentPage
			entry.PageCount = 1
			currentPage++
		}
		giantHTML := buildGiantHTML(entries, *cols, *fontSize)
		if err := os.WriteFile(finalHTMLPath, []byte(giantHTML), 0o644); err != nil {
			log.Fatal(err)
		}
		color.Green("HTML saved to: %s", finalHTMLPath)
		return
	}

	ctx, cancel := createChromeContext(*disableWebSecurity)
	defer cancel()

	for i, e := range entries {
		tempPDF := filepath.Join(tempDir, fmt.Sprintf("count_%d.pdf", i))
		fullHtml := buildStandaloneHTML(e, *cols, *fontSize)

		if err := printToPDF(ctx, fullHtml, tempPDF, true); err != nil {
			color.Red("Error rendering preview: %v", err)
			e.PageCount = 1
			continue
		}

		pages, err := countPagesInPDF(tempPDF)
		if err != nil || pages == 0 {
			pages = 1
		}
		e.PageCount = pages
	}

	// Phase 3: Table of Contents Calculation
	fmt.Println("\n--- Phase 3: Calculating Table of Contents ---")
	tocPages := 1
	for {
		currentStart := tocPages + 1
		for _, e := range entries {
			e.StartPageNum = currentStart
			currentStart += e.PageCount
		}

		tocPath := filepath.Join(tempDir, "000_toc.pdf")
		if err := printToPDF(ctx, buildToCHTML(entries), tocPath, true); err != nil {
			log.Fatalf("Error generating TOC: %v", err)
		}

		actualTocPages, err := countPagesInPDF(tocPath)
		if err != nil || actualTocPages == 0 {
			actualTocPages = 1
		}

		if actualTocPages == tocPages {
			break
		}
		tocPages = actualTocPages
	}
	fmt.Printf("Table of Contents finished occupying %d page(s).\n", tocPages)

	// Phase 4: Final Rendering
	fmt.Println("\n--- Phase 4: Generating Complete Ebook ---")
	giantHTML := buildGiantHTML(entries, *cols, *fontSize)

	if *outputFormat == "both" {
		if err := os.WriteFile(finalHTMLPath, []byte(giantHTML), 0o644); err != nil {
			log.Fatal(err)
		}
		color.Cyan("HTML master file saved to: %s", finalHTMLPath)
	}

	if err := printToPDF(ctx, giantHTML, finalPDFPath, true); err != nil {
		log.Fatalf("Final rendering error: %v", err)
	}

	color.Green("\n\nAbsolute Success! Ebook saved to: %s", finalPDFPath)
}

// Version is replaced by GoReleaser at build time.
var Version = "dev"

func printHelp() {
	fmt.Print(`Offprint archives online publications as Markdown, HTML, and printable PDFs.

Usage:
  offprint archive [flags] <archive-url|url-file>
  offprint bundle [flags] <article-url|url-file>
  offprint cookies set --domain DOMAIN [--from-env NAME|--file PATH]
  offprint version

Examples:
  offprint archive https://example.substack.com/archive
  offprint archive --input publications.txt --output ~/Documents/Offprint
  offprint bundle --format both --name reading-list links.txt

Run "offprint <command> --help" for command flags.
`)
}

func runArchiveCommand(args []string) error {
	set := flag.NewFlagSet("archive", flag.ContinueOnError)
	output := set.String("output", defaultOutputDir(), "output directory")
	inputFlag := set.String("input", "", "archive URL or file containing archive URLs")
	cookieFile := set.String("cookie-file", "", "file containing the Substack Cookie header")
	cookieEnv := set.String("cookie-env", "SUBSTACK_COOKIE", "environment variable containing the Substack Cookie header")
	if err := set.Parse(args); err != nil {
		return err
	}
	input := strings.TrimSpace(*inputFlag)
	if input == "" && set.NArg() == 1 {
		input = set.Arg(0)
	}
	if input == "" {
		return fmt.Errorf("archive input is required; example: offprint archive https://example.substack.com/archive")
	}
	archiveURLs, err := readURLInput(input)
	if err != nil {
		return err
	}
	cookie := strings.TrimSpace(os.Getenv(*cookieEnv))
	if *cookieFile != "" {
		data, err := os.ReadFile(*cookieFile)
		if err != nil {
			return fmt.Errorf("read cookie file: %w", err)
		}
		cookie = strings.TrimSpace(string(data))
	}
	if cookie == "" {
		return fmt.Errorf("authenticated cookie missing; set %s or pass --cookie-file", *cookieEnv)
	}

	for _, archiveURL := range archiveURLs {
		parsed, _ := url.Parse(archiveURL)
		destination := *output
		if len(archiveURLs) > 1 || *output == defaultOutputDir() {
			destination = filepath.Join(*output, parsed.Hostname())
		}
		if err := exportSubstackArchive(archiveURL, destination, cookie); err != nil {
			return fmt.Errorf("export %s: %w", archiveURL, err)
		}
	}
	return nil
}

func normalizeBundleCommand(args []string) (func(), error) {
	set := flag.NewFlagSet("bundle", flag.ContinueOnError)
	inputFlag := set.String("input", "", "article URL or file containing article URLs")
	output := set.String("output", defaultOutputDir(), "output directory")
	name := set.String("name", "ebook", "output collection name")
	format := set.String("format", "both", "output format: html, pdf, or both")
	font := set.String("font", "", "optional TTF/OTF/WOFF font file")
	css := set.String("css", "", "optional CSS applied to the complete document")
	configDir := set.String("config-dir", defaultConfigDir(), "Offprint configuration directory")
	siteProfile := set.String("site-profile", "", "additional site profile JSON with highest precedence")
	disableSecurity := set.Bool("disable-web-security", false, "disable Chromium web security (unsafe)")
	columns := set.Int("columns", 1, "number of text columns")
	fontSize := set.Int("font-size", 11, "font size in pixels")
	ignore := set.String("ignore", "", "comma-separated title fragments to remove")
	if err := set.Parse(args); err != nil {
		return func() {}, err
	}
	input := strings.TrimSpace(*inputFlag)
	if input == "" && set.NArg() == 1 {
		input = set.Arg(0)
	}
	if input == "" {
		return func() {}, fmt.Errorf("bundle input is required; pass a URL or URL file")
	}
	urls, err := readURLInput(input)
	if err != nil {
		return func() {}, err
	}
	temp, err := os.CreateTemp("", "offprint-urls-*.txt")
	if err != nil {
		return func() {}, err
	}
	cleanup := func() { _ = os.Remove(temp.Name()) }
	for _, item := range urls {
		if _, err := fmt.Fprintln(temp, item); err != nil {
			_ = temp.Close()
			cleanup()
			return func() {}, err
		}
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return func() {}, err
	}

	os.Args = []string{os.Args[0], "-i", temp.Name(), "-output-dir", *output, "-o", *name + ".pdf", "-format", *format, "-cols", fmt.Sprint(*columns), "-fontsize", fmt.Sprint(*fontSize), "-config-dir", *configDir}
	if *font != "" {
		os.Args = append(os.Args, "-font", *font)
	}
	if *css != "" {
		os.Args = append(os.Args, "-css", *css)
	}
	if *siteProfile != "" {
		os.Args = append(os.Args, "-domains-file", *siteProfile)
	}
	if *disableSecurity {
		os.Args = append(os.Args, "-disable-web-security")
	}
	if *ignore != "" {
		os.Args = append(os.Args, "-ignore", *ignore)
	}
	return cleanup, nil
}

func exportSubstackArchive(archiveURL, outputDir, cookie string) error {
	if strings.TrimSpace(cookie) == "" {
		return fmt.Errorf("SUBSTACK_COOKIE is required for an authenticated archive export")
	}
	client, err := NewSubstackClient(archiveURL, cookie)
	if err != nil {
		return err
	}

	fmt.Printf("Discovering posts from %s...\n", archiveURL)
	posts, err := client.Archive()
	if err != nil {
		return fmt.Errorf("discover archive: %w", err)
	}
	fmt.Printf("Found %d posts.\n", len(posts))

	succeeded := 0
	var failures []string
	for i, item := range posts {
		color.Blue("Downloading [%d/%d]: %s", i+1, len(posts), item.Title)
		post, err := client.FetchPost(item.Slug)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", item.Slug, err))
			color.Red("  └ Error: %v", err)
			continue
		}
		path, err := client.WriteMarkdown(post, outputDir)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", item.Slug, err))
			color.Red("  └ Error: %v", err)
			continue
		}
		succeeded++
		color.Green("  └ ✓ %s", path)
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Printf("Export complete: %d written, %d failed.\n", succeeded, len(failures))
	if len(failures) > 0 {
		return fmt.Errorf("some posts could not be exported:\n  %s", strings.Join(failures, "\n  "))
	}
	return nil
}
