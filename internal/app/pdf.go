package app

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func createChromeContext(disableWebSecurity bool) (context.Context, context.CancelFunc) {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if disableWebSecurity {
		opts = append(opts, chromedp.Flag("disable-web-security", true))
	}
	opts = append(opts,
		chromedp.Flag("allow-file-access-from-files", true),
		chromedp.Flag("headless", true),
	)
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	return chromedp.NewContext(allocCtx)
}

func printToPDF(ctx context.Context, htmlContent string, outputPath string, withFooter bool) error {
	tmpHtml := outputPath + ".html"
	if err := os.WriteFile(tmpHtml, []byte(htmlContent), 0644); err != nil {
		return err
	}
	absPath, _ := filepath.Abs(tmpHtml)
	defer func() { _ = os.Remove(tmpHtml) }()

	var buf []byte
	footerTpl := ""
	if withFooter {
		footerTpl = `
			<div style="font-size: 10px; width: 100%; text-align: center; color: #555; padding-top: 10px;">
				<span class="pageNumber"></span>
			</div>`
	}

	err := chromedp.Run(ctx,
		chromedp.Navigate("file://"+absPath),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			params := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).WithPaperHeight(11.69).
				WithMarginTop(0.6).WithMarginBottom(0.6).
				WithMarginLeft(0.6).WithMarginRight(0.6)

			if withFooter {
				params = params.WithDisplayHeaderFooter(true).
					WithFooterTemplate(footerTpl).
					WithHeaderTemplate("<span></span>")
			}

			buf, _, err = params.Do(ctx)
			return err
		}),
	)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, buf, 0644)
}

func countPagesInPDF(filePath string) (int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	re := regexp.MustCompile(`/Type\s*/Page[^s]`)
	matches := re.FindAll(data, -1)
	return len(matches), nil
}
