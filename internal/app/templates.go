package app

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gabrielassisxyz/offprint/internal/assets"
)

var (
	fontCSS    string
	fontFamily = `Georgia, "Times New Roman", serif`
)

func configureFont(path string) error {
	fontCSS = ""
	fontFamily = `Georgia, "Times New Roman", serif`
	if path == "" {
		return nil
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read font %q: %w", path, err)
	}
	mimeType := http.DetectContentType(bytes)
	b64 := base64.StdEncoding.EncodeToString(bytes)
	fontCSS = fmt.Sprintf(`
		@font-face {
			font-family: 'Offprint Custom';
			src: url(data:%s;base64,%s);
		}
	`, mimeType, b64)
	fontFamily = `'Offprint Custom', Georgia, "Times New Roman", serif`
	return nil
}

func getFontCSS() string {
	return fontCSS
}

func getBaseStyles(fontSize, cols int) (string, string) {
	columnCSS := ""
	imgCSS := `img { max-width: 45%; height: auto; float: right; margin: 10px 0 15px 20px; border-radius: 4px; }`

	if cols > 1 {
		columnCSS = fmt.Sprintf("column-count: %d; column-gap: 8mm;", cols)
		imgCSS = `img { max-width: 100%%; height: auto; display: block; margin: 15px auto; border-radius: 4px; }`
	}

	coreCSS := fmt.Sprintf(`
		%s
		@page { size: A4; margin: 20mm; }
		body { font-family: %s; font-size: %dpx; line-height: 1.6; color: #333; -webkit-text-size-adjust: 100%%; text-size-adjust: 100%%; }
		.content { display: flow-root; %s }
		h1.doc-title { text-align: center; color: #000; margin-top: 0px; margin-bottom: 10px; font-size: 24px; }
		.source-link { text-align: center; font-size: 12px; color: #666; margin-bottom: 20px; display: block; font-style: italic; border-bottom: 1px dotted #000; }
		p { margin-bottom: 1.2em; text-align: justify; }
		a { color: #0056b3; text-decoration: none; border-bottom: 1px dotted #0056b3; }
		%s
	`, getFontCSS(), fontFamily, fontSize, columnCSS, imgCSS)

	return coreCSS, columnCSS
}

func buildReferencesHTML(refs []Reference) string {
	var sb strings.Builder
	sb.WriteString(`<ol style="font-size: 9px; color: #555; padding-left: 25px; line-height: 1.8;">`)
	for _, r := range refs {
		if r.HideURL {
			fmt.Fprintf(&sb, `
				<li value="%d" style="margin-bottom: 8px; word-wrap: break-word;">
					<strong>%s</strong>
				</li>`, r.Number, r.Title)
		} else {
			fmt.Fprintf(&sb, `
				<li value="%d" style="margin-bottom: 8px; word-wrap: break-word;">
					<strong>%s</strong>: <a href="%s" style="color: #666; text-decoration: underline;">%s</a>
				</li>`, r.Number, r.Title, r.URL, r.DisplayURL)
		}
	}
	sb.WriteString(`</ol>`)
	return sb.String()
}

func buildGiantHTML(entries []*Entry, cols int, fontSize int) string {
	var sb strings.Builder
	coreCSS, _ := getBaseStyles(fontSize, cols)

	fmt.Fprintf(&sb, `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<style>
			%s
			/* TOC & Pagination */
			h1.toc-title { text-align: center; margin-bottom: 40px; font-size: 24px; }
			.toc-entry { margin-bottom: 12px; font-size: 15px; display: flex; justify-content: space-between; }
			.dots { flex-grow: 1; border-bottom: 1px dotted #ccc; margin: 0 10px; position: relative; top: -5px; }
			.page-break { page-break-before: always; margin: 0; padding: 0; border: 0; }
		</style>
	</head>
	<body>
		<h1 class="toc-title">Table of Contents</h1>
	`, coreCSS)

	// Table of Contents
	for _, e := range entries {
		fmt.Fprintf(&sb, `
			<div class="toc-entry">
				<span>%s</span>
				<span class="dots"></span>
				<span>%d</span>
			</div>`, e.Title, e.StartPageNum)
	}

	// Articles
	for _, e := range entries {
		header, footer, customCSS := buildArticleWrapper(e)

		sb.WriteString(`<div class="page-break"></div>`)
		sb.WriteString(customCSS) // Inject custom CSS if any
		sb.WriteString(header)
		fmt.Fprintf(&sb, `<div class="content">%s</div>`, e.HTMLContent)
		sb.WriteString(footer)
	}

	sb.WriteString(`</body></html>`)
	return sb.String()
}

func buildStandaloneHTML(entry *Entry, cols int, fontSize int) string {
	coreCSS, _ := getBaseStyles(fontSize, cols)
	header, footer, customCSS := buildArticleWrapper(entry)

	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>%s</style>
			%s
		</head>
		<body>
			%s
			<div class="content">%s</div>
			%s
		</body>
		</html>
	`, coreCSS, customCSS, header, entry.HTMLContent, footer)
}

func buildToCHTML(entries []*Entry) string {
	var sb strings.Builder
	sb.WriteString(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<style>
			` + getFontCSS() + `
			@page { size: A4; margin: 20mm; }
			body { font-family: ` + fontFamily + `; }
			h1.toc-title { text-align: center; margin-bottom: 40px; font-size: 24px; }
			.toc-entry { margin-bottom: 12px; font-size: 15px; display: flex; justify-content: space-between; }
			.dots { flex-grow: 1; border-bottom: 1px dotted #ccc; margin: 0 10px; position: relative; top: -5px; }
		</style>
	</head>
	<body>
		<h1 class="toc-title">Table of Contents</h1>
	`)

	for _, e := range entries {
		fmt.Fprintf(&sb, `
			<div class="toc-entry">
				<span>%s</span>
				<span class="dots"></span>
				<span>%d</span>
			</div>`, e.Title, e.StartPageNum)
	}
	sb.WriteString(`</body></html>`)
	return sb.String()
}

func buildArticleWrapper(e *Entry) (string, string, string) {
	titleAlign := "center"
	if e.Config.TitleAlign != "" {
		titleAlign = e.Config.TitleAlign
	}

	subAlign := "center"
	if e.Config.SubtitleAlign != "" {
		subAlign = e.Config.SubtitleAlign
	}

	sourceLink := ""
	if e.URL != "" {
		sourceLink = fmt.Sprintf(`<a href="%s" class="source-link" style="text-align: %s">Source: %s</a>`, e.URL, titleAlign, e.URL)
	}

	header := fmt.Sprintf(`<h1 class="doc-title" style="text-align: %s">%s</h1>`, titleAlign, e.Title)
	if e.Subtitle != "" {
		header += fmt.Sprintf(`<h3 class="doc-subtitle" style="text-align: %s; color: #555; margin-bottom: 15px;">%s</h3>`, subAlign, e.Subtitle)
	}

	if e.Config.ShowSeparator {
		header += `<hr style="border: 0; border-bottom: 1px solid #ccc; margin-bottom: 20px;">`
	}

	footer := ""
	if e.Config.SourcePosition == "bottom" {
		footer = `<div style="margin-top: 40px;">` + sourceLink + `</div>`
	} else {
		header += sourceLink
	}

	customCSS := ""
	if e.Config.CustomCSSPath != "" {
		cssBytes, err := os.ReadFile(e.Config.CustomCSSPath)
		if err != nil && e.Config.CustomCSSPath == "css/karlsson.css" {
			cssBytes = assets.KarlssonCSS
			err = nil
		}
		if err == nil {
			customCSS = fmt.Sprintf("<style>\n%s\n</style>\n", string(cssBytes))
		}
	}

	return header, footer, customCSS
}
