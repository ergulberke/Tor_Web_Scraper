package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type Scraper struct {
	Client     *http.Client
	Logger     *log.Logger
	Out        *OutputWriter
	SocksProxy string

	BrowserAllocCtx context.Context
}

func (s *Scraper) ScanOne(idx int, target string) {
	name := SafeName(target, idx)

	s.Logger.Printf("[INFO] Scanning: %s", target)

	// 1) HTML çek
	body, status, err := s.fetchHTML(target)
	if err != nil {
		s.Logger.Printf("[ERR]  %s -> %v", target, err)
		_ = s.Out.WriteErrorHTML(name, target, err.Error())
		_ = s.Out.WritePlaceholderPNG(name)
		return
	}

	_ = s.Out.WriteHTML(name, body)
	s.Logger.Printf("[INFO]  %s -> HTTP %d (html saved)", target, status)

	// 2) Screenshot
	shotPath := s.Out.ScreenshotPath(name)
	if err := s.captureFullPage(target, shotPath); err != nil {
		s.Logger.Printf("[ERR]  %s -> screenshot error: %v", target, err)
		_ = s.Out.WritePlaceholderPNG(name)
		return
	}

	s.Logger.Printf("[OK]   %s -> screenshot saved", target)
}

func (s *Scraper) fetchHTML(u string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) Gecko/20100101 Firefox/115.0")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, 12*1024*1024)
	b, err := io.ReadAll(limited)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}

func (s *Scraper) captureFullPage(u string, _ string) error {

	name := SafeName(u, 0) // klasör adı için
	siteDir, err := s.Out.EnsureSiteScreensDir(name)
	if err != nil {
		return err
	}

	// tek browser alloc context’ten yeni tab
	ctx, cancel := chromedp.NewContext(s.BrowserAllocCtx)
	defer cancel()

	ctx, cancelTO := context.WithTimeout(ctx, 140*time.Second)
	defer cancelTO()

	// ram hatasaı için görüntü sabit
	const vpW, vpH = 1280, 720
	var totalH int64

	if err := chromedp.Run(ctx,
		emulation.SetDeviceMetricsOverride(vpW, vpH, 1.0, false),
		chromedp.Navigate(u),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`Math.max(document.body.scrollHeight, document.documentElement.scrollHeight)`, &totalH),
	); err != nil {
		return fmt.Errorf("navigate/measure: %w", err)
	}

	if totalH < vpH {
		totalH = vpH
	}
	// uzun sayfalar için sınır
	if totalH > 60000 {
		totalH = 60000
	}

	step := int64(vpH - 80)
	if step < 300 {
		step = 300
	}

	part := 1
	for y := int64(0); y < totalH; y += step {
		// scroll
		js := fmt.Sprintf(`window.scrollTo(0, %d);`, y)

		var buf []byte
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(js, nil),
			chromedp.Sleep(1200*time.Millisecond),
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				buf, err = page.CaptureScreenshot().
					WithFormat(page.CaptureScreenshotFormatPng).
					WithFromSurface(true).
					Do(ctx)
				return err
			}),
		); err != nil {
			return fmt.Errorf("capture part %d: %w", part, err)
		}

		filename := fmt.Sprintf("part_%03d.png", part)
		fullpath := filepath.Join(siteDir, filename)
		if err := os.WriteFile(fullpath, buf, 0644); err != nil {
			return err
		}
		part++

		// fazla part olmaması için güvenlik
		if part > 120 {
			break
		}
	}

	return nil
}
