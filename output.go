package main

import (
	"crypto/sha1"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type OutputWriter struct {
	BaseDir       string
	HTMLDir       string
	ScreensDir    string
	LogDir        string
	ReportLogPath string
}

func NewOutputWriter(base string) (*OutputWriter, error) {
	ow := &OutputWriter{
		BaseDir:    base,
		HTMLDir:    filepath.Join(base, "outputs", "html"),
		ScreensDir: filepath.Join(base, "outputs", "screenshots"),
		LogDir:     filepath.Join(base, "log"),
	}

	if err := os.MkdirAll(ow.HTMLDir, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(ow.ScreensDir, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(ow.LogDir, 0755); err != nil {
		return nil, err
	}

	ow.ReportLogPath = filepath.Join(ow.LogDir, "scan_report.log")
	return ow, nil
}

func SafeName(rawURL string, idx int) string {
	u, err := url.Parse(rawURL)
	host := "site"
	if err == nil && u.Host != "" {
		host = u.Host
	}
	host = strings.ReplaceAll(host, ":", "_")
	if len(host) > 40 {
		host = host[:40]
	}

	sum := sha1.Sum([]byte(rawURL))
	short := hex.EncodeToString(sum[:])[:10]

	return pad3(idx) + "_" + host + "_" + short
}

func pad3(i int) string {
	if i < 0 {
		i = -i
	}
	if i < 10 {
		return "00" + itoa(i)
	}
	if i < 100 {
		return "0" + itoa(i)
	}
	return itoa(i)
}

func (ow *OutputWriter) WriteHTML(name string, data []byte) error {
	path := filepath.Join(ow.HTMLDir, name+".html")
	return os.WriteFile(path, data, 0644)
}

func (ow *OutputWriter) WriteErrorHTML(name string, target string, errText string) error {
	path := filepath.Join(ow.HTMLDir, name+"_error.html")
	html := "<!doctype html><html><head><meta charset=\"utf-8\"><title>ERROR</title></head><body>" +
		"<h3>Scan Error</h3>" +
		"<p><b>Target:</b> " + escapeTiny(target) + "</p>" +
		"<pre>" + escapeTiny(errText) + "</pre>" +
		"</body></html>"
	return os.WriteFile(path, []byte(html), 0644)
}

func (ow *OutputWriter) WritePlaceholderPNG(name string) error {
	//placeholder  siteye girilemezsde diye
	path := filepath.Join(ow.ScreensDir, name+"_placeholder.png")

	img := image.NewRGBA(image.Rect(0, 0, 900, 200))
	bg := color.RGBA{240, 240, 240, 255}
	fg := color.RGBA{40, 40, 40, 255}

	for y := 0; y < 200; y++ {
		for x := 0; x < 900; x++ {
			img.Set(x, y, bg)
		}
	}
	//üst kısım
	for y := 0; y < 30; y++ {
		for x := 0; x < 900; x++ {
			img.Set(x, y, fg)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func (ow *OutputWriter) ScreenshotPath(name string) string {
	return filepath.Join(ow.ScreensDir, name+".png")
}

func escapeTiny(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var b [32]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + (i % 10))
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
func (ow *OutputWriter) EnsureSiteScreensDir(siteName string) (string, error) {
	dir := filepath.Join(ow.ScreensDir, siteName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}
