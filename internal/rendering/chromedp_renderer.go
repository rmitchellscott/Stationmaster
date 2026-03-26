package rendering

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

const defaultIdleTimeout = 2 * time.Minute

type ChromeRenderer struct {
	allocCtx      context.Context
	allocCancel   context.CancelFunc
	remote        bool

	mu            sync.Mutex
	browserCtx    context.Context
	browserCancel context.CancelFunc
	idleTimer     *time.Timer
	idleTimeout   time.Duration
}

func NewChromeRenderer() (*ChromeRenderer, error) {
	remoteURL := config.Get("CHROME_REMOTE_URL", "")

	r := &ChromeRenderer{
		idleTimeout: defaultIdleTimeout,
	}

	if remoteURL != "" {
		logging.Info("[CHROME] Connecting to remote Chrome", "url", remoteURL)
		r.allocCtx, r.allocCancel = chromedp.NewRemoteAllocator(context.Background(), remoteURL)
		r.remote = true
	} else {
		chromePath := config.Get("CHROME_PATH", "")
		logging.Info("[CHROME] Using embedded headless Chrome", "path", chromePath)
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-software-rasterizer", true),
		)
		if chromePath != "" {
			opts = append(opts, chromedp.ExecPath(chromePath))
		}
		r.allocCtx, r.allocCancel = chromedp.NewExecAllocator(context.Background(), opts...)
	}

	return r, nil
}

// ensureBrowser lazily starts the browser if not running, and resets the idle timer.
func (r *ChromeRenderer) ensureBrowser() (context.Context, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.browserCtx != nil && r.browserCtx.Err() == nil {
		r.resetIdleTimer()
		return r.browserCtx, nil
	}

	logging.Info("[CHROME] Starting browser process")
	r.browserCtx, r.browserCancel = chromedp.NewContext(r.allocCtx)
	if err := chromedp.Run(r.browserCtx); err != nil {
		r.browserCancel()
		r.browserCtx = nil
		r.browserCancel = nil
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}

	logging.Info("[CHROME] Browser process ready")
	r.resetIdleTimer()
	return r.browserCtx, nil
}

// resetIdleTimer resets (or starts) the idle shutdown timer. Must be called with mu held.
func (r *ChromeRenderer) resetIdleTimer() {
	if r.remote {
		return
	}
	if r.idleTimer != nil {
		r.idleTimer.Stop()
	}
	r.idleTimer = time.AfterFunc(r.idleTimeout, r.stopBrowser)
}

// stopBrowser shuts down the browser process to free memory.
func (r *ChromeRenderer) stopBrowser() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.browserCancel != nil {
		logging.Info("[CHROME] Idle timeout reached, stopping browser process")
		r.browserCancel()
		r.browserCtx = nil
		r.browserCancel = nil
	}
}

func (r *ChromeRenderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.idleTimer != nil {
		r.idleTimer.Stop()
	}
	if r.browserCancel != nil {
		r.browserCancel()
	}
	if r.allocCancel != nil {
		r.allocCancel()
	}
	return nil
}

func (r *ChromeRenderer) CaptureScreenshot(ctx context.Context, url string, width, height int, waitTimeSeconds int, headers map[string]string) ([]byte, error) {
	browserCtx, err := r.ensureBrowser()
	if err != nil {
		return nil, err
	}

	taskCtx, cancel := chromedp.NewContext(browserCtx)
	defer cancel()

	var buf []byte
	actions := []chromedp.Action{
		chromedp.EmulateViewport(int64(width), int64(height)),
	}

	if len(headers) > 0 {
		h := make(network.Headers, len(headers))
		for k, v := range headers {
			h[k] = v
		}
		actions = append(actions, network.Enable(), network.SetExtraHTTPHeaders(h))
	}

	actions = append(actions,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.Sleep(time.Duration(waitTimeSeconds)*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, err = page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatPng).
				WithClip(&page.Viewport{
					X: 0, Y: 0,
					Width:  float64(width),
					Height: float64(height),
					Scale:  1,
				}).Do(ctx)
			return err
		}),
	)

	if err := chromedp.Run(taskCtx, actions...); err != nil {
		return nil, fmt.Errorf("chrome screenshot failed: %w", err)
	}

	return buf, nil
}

func (r *ChromeRenderer) RenderHTML(ctx context.Context, html string, width, height int) ([]byte, error) {
	result, err := r.RenderHTMLWithResult(ctx, html, width, height)
	if err != nil {
		return nil, err
	}
	return result.ImageData, nil
}

func (r *ChromeRenderer) RenderHTMLWithResult(ctx context.Context, html string, width, height int) (*RenderHTMLResult, error) {
	browserCtx, err := r.ensureBrowser()
	if err != nil {
		return nil, err
	}

	taskCtx, cancel := chromedp.NewContext(browserCtx)
	defer cancel()

	logging.Renderer("HTML content analysis",
		"html_size_chars", len(html),
		"html_size_bytes", len([]byte(html)),
		"viewport_width", width,
		"viewport_height", height,
	)

	if len(html) > 0 {
		scriptCount := strings.Count(html, "<script")
		styleCount := strings.Count(html, "<style")
		divCount := strings.Count(html, "<div")
		imgCount := strings.Count(html, "<img")

		logging.Renderer("HTML complexity analysis",
			"script_tags", scriptCount,
			"style_tags", styleCount,
			"div_tags", divCount,
			"img_tags", imgCount,
		)

		if strings.Contains(html, "data:image/") {
			dataImageCount := strings.Count(html, "data:image/")
			logging.Renderer("WARNING: Found data:image URLs",
				"data_image_count", dataImageCount,
				"warning", "data:image URLs can cause large payloads",
			)
		}

		if len(html) > 100000 {
			logging.Renderer("WARNING: Large HTML size",
				"html_size_chars", len(html),
				"warning", "HTML size may be too large",
			)
		}
	}

	var buf []byte
	actions := []chromedp.Action{
		chromedp.EmulateViewport(int64(width), int64(height)),
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
		}),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			deadline := time.Now().Add(20 * time.Second)
			for time.Now().Before(deadline) {
				var complete bool
				err := chromedp.Evaluate(
					`document.body.getAttribute('data-render-complete') === 'true'`,
					&complete,
				).Do(ctx)
				if err == nil && complete {
					return nil
				}
				time.Sleep(200 * time.Millisecond)
			}
			logging.Renderer("Render-complete signal not received within 20s, proceeding anyway")
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, err = page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatPng).
				WithClip(&page.Viewport{
					X: 0, Y: 0,
					Width:  float64(width),
					Height: float64(height),
					Scale:  1,
				}).Do(ctx)
			return err
		}),
	}

	if err := chromedp.Run(taskCtx, actions...); err != nil {
		return nil, fmt.Errorf("chrome HTML render failed: %w", err)
	}

	flags := r.parseFlagsFromDOM(taskCtx, html, width, height)

	if flags.SkipScreenGeneration {
		return nil, fmt.Errorf("render skipped due at plugin's request")
	}

	return &RenderHTMLResult{
		ImageData: buf,
		Flags:     flags,
	}, nil
}

func (r *ChromeRenderer) parseFlagsFromDOM(taskCtx context.Context, html string, width, height int) RenderFlags {
	flags := RenderFlags{}

	var domHTML string
	err := chromedp.Run(taskCtx,
		chromedp.OuterHTML("html", &domHTML),
	)
	if err != nil {
		logging.Renderer("Failed to get DOM for flag parsing", "error", err.Error())
		return flags
	}

	if strings.Contains(domHTML, `data-trmnl-skip-screen-generation="true"`) ||
		strings.Contains(domHTML, `data-trmnl-skip-screen-generation='true'`) {
		flags.SkipScreenGeneration = true
		logging.Info("[CHROME] TRMNL_SKIP_SCREEN_GENERATION detected in DOM attributes")
	}

	if strings.Contains(domHTML, `data-trmnl-skip-display="true"`) ||
		strings.Contains(domHTML, `data-trmnl-skip-display='true'`) {
		flags.SkipDisplay = true
		logging.Info("[CHROME] TRMNL_SKIP_DISPLAY detected in DOM attributes")
	}

	return flags
}

func DefaultChromeRenderer() (*ChromeRenderer, error) {
	return NewChromeRenderer()
}
