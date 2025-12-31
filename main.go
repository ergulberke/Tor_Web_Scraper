package main

import (
	"context"
	"flag"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"time"
)

func main() {
	var (
		targetsPath = flag.String("targets", "targets.yaml", "targets file path")
		socksAddr   = flag.String("socks", "auto", "tor socks5 address (auto, 127.0.0.1:9050, 127.0.0.1:9150)")
		timeout     = flag.Duration("timeout", 50*time.Second, "http timeout")
		checkTor    = flag.Bool("checktor", true, "tor ip kontrolü yap")
	)
	flag.Parse()

	chosenSocks := AutoDetectSocks(*socksAddr)

	out, err := NewOutputWriter(".")
	if err != nil {
		panic(err)
	}

	logFile, err := os.OpenFile(out.ReportLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)
	also := log.New(os.Stdout, "", log.LstdFlags)

	client, err := NewTorHTTPClient(chosenSocks, *timeout)

	if err != nil {
		also.Printf("[FATAL] SOCKS5 setup error: %v", err)
		logger.Printf("[FATAL] SOCKS5 setup error: %v", err)
		return
	}

	if *checkTor {
		tc, err := CheckTorIP(client)
		if err != nil {
			also.Printf("[WARN] Tor check failed: %v", err)
			logger.Printf("[WARN] Tor check failed: %v", err)
		} else {
			also.Printf("[INFO] TorCheck: IsTor=%v IP=%s", tc.IsTor, tc.IP)
			logger.Printf("[INFO] TorCheck: IsTor=%v IP=%s", tc.IsTor, tc.IP)
		}
	}

	targets, err := ReadTargets(*targetsPath)
	if err != nil {
		also.Printf("[FATAL] targets read error: %v", err)
		logger.Printf("[FATAL] targets read error: %v", err)
		return
	}

	also.Printf("[INFO] Loaded %d targets", len(targets))
	logger.Printf("[INFO] Loaded %d targets", len(targets))

	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("proxy-server", "socks5://"+chosenSocks),
	)

	browserAllocCtx, cancelBrowser := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer cancelBrowser()

	s := &Scraper{
		Client:          client,
		Logger:          logger,
		Out:             out,
		SocksProxy:      chosenSocks,
		BrowserAllocCtx: browserAllocCtx,
	}

	for i, t := range targets {
		s.ScanOne(i+1, t)
	}

	also.Printf("[DONE] Finished. Log: %s", out.ReportLogPath)
	logger.Printf("[DONE] Finished.")
}
