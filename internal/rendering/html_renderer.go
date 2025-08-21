package rendering

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// HTMLRenderer renders HTML to images using headless Chrome via rod
type HTMLRenderer struct {
	browser *rod.Browser
	engine  *TemplateEngine
	options RenderOptions
}

// NewHTMLRenderer creates a new HTML renderer
func NewHTMLRenderer(options RenderOptions) (*HTMLRenderer, error) {
	// Get Chromium binary path from environment or use default system paths
	chromiumBin, err := getChromiumBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find Chromium binary: %w", err)
	}
	
	log.Printf("[HTML_RENDERER] Using Chromium binary: %s", chromiumBin)
	
	// Launch headless browser with system binary
	l := launcher.New().
		Headless(true).
		NoSandbox(true).
		Set("disable-web-security").
		Set("disable-features", "VizDisplayCompositor").
		Set("disable-dev-shm-usage").
		Set("disable-gpu").
		Set("disable-extensions").
		Set("disable-plugins").
		Set("no-first-run").
		Set("disable-background-timer-throttling").
		Set("disable-backgrounding-occluded-windows").
		Set("disable-renderer-backgrounding").
		Bin(chromiumBin)

	// Try to launch browser
	url, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser (binary: %s): %w", chromiumBin, err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	return &HTMLRenderer{
		browser: browser,
		engine:  NewTemplateEngine(),
		options: options,
	}, nil
}

// getChromiumBinary attempts to find the Chromium binary path
func getChromiumBinary() (string, error) {
	// Check environment variables first
	if bin := os.Getenv("CHROMIUM_BIN"); bin != "" {
		if _, err := os.Stat(bin); err == nil {
			return bin, nil
		}
		return "", fmt.Errorf("CHROMIUM_BIN path does not exist: %s", bin)
	}
	if bin := os.Getenv("CHROME_BIN"); bin != "" {
		if _, err := os.Stat(bin); err == nil {
			return bin, nil
		}
		return "", fmt.Errorf("CHROME_BIN path does not exist: %s", bin)
	}
	
	// Try common system paths
	commonPaths := []string{
		"/usr/bin/chromium-browser",
		"/usr/bin/chromium",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/google-chrome",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", // macOS
	}
	
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	// No Chromium binary found
	return "", fmt.Errorf("no Chromium binary found - install Chromium or set CHROMIUM_BIN/CHROME_BIN environment variable")
}

// RenderToImage renders HTML content to an image
func (r *HTMLRenderer) RenderToImage(ctx context.Context, html string, options RenderOptions) ([]byte, error) {
	// Merge with default options
	opts := r.mergeOptions(options)
	
	// Create a new page
	page, err := r.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()
	
	// Set viewport to exact pixel dimensions
	err = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             opts.Width,
		Height:            opts.Height,
		DeviceScaleFactor: 1.0, // No scaling - render at exact pixel dimensions
		Mobile:            false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set viewport: %w", err)
	}
	
	// Wrap HTML in a complete document
	fullHTML := r.wrapHTML(html, opts)
	
	// Navigate to data URL
	err = page.Navigate("data:text/html;charset=utf-8," + fullHTML)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to HTML: %w", err)
	}
	
	// Wait for content to load
	if opts.WaitTime > 0 {
		time.Sleep(time.Duration(opts.WaitTime) * time.Millisecond)
	}
	
	// Take screenshot
	var imageBytes []byte
	if opts.Format == "jpeg" || opts.Format == "jpg" {
		imageBytes, err = page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormatJpeg,
			Quality: &opts.Quality,
		})
	} else {
		imageBytes, err = page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to take screenshot: %w", err)
	}
	
	return imageBytes, nil
}

// RenderTemplateToImage renders a template with data to an image
func (r *HTMLRenderer) RenderTemplateToImage(ctx context.Context, template string, data map[string]interface{}, options RenderOptions) ([]byte, error) {
	// Render template to HTML
	html, err := r.engine.RenderTemplate(template, data)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}
	
	// Render HTML to image
	return r.RenderToImage(ctx, html, options)
}

// Close cleans up the browser instance
func (r *HTMLRenderer) Close() error {
	if r.browser != nil {
		r.browser.Close()
	}
	return nil
}

// mergeOptions merges provided options with defaults
func (r *HTMLRenderer) mergeOptions(options RenderOptions) RenderOptions {
	opts := r.options
	
	if options.Width > 0 {
		opts.Width = options.Width
	}
	if options.Height > 0 {
		opts.Height = options.Height
	}
	if options.Quality > 0 {
		opts.Quality = options.Quality
	}
	if options.Format != "" {
		opts.Format = options.Format
	}
	if options.DPI > 0 {
		opts.DPI = options.DPI
	}
	if options.WaitTime > 0 {
		opts.WaitTime = options.WaitTime
	}
	if options.ExtraCSS != "" {
		opts.ExtraCSS = options.ExtraCSS
	}
	if options.Variables != nil {
		if opts.Variables == nil {
			opts.Variables = make(map[string]string)
		}
		for k, v := range options.Variables {
			opts.Variables[k] = v
		}
	}
	
	return opts
}

// wrapHTML wraps the provided HTML in a complete document with TRMNL styling
func (r *HTMLRenderer) wrapHTML(html string, options RenderOptions) string {
	var cssVars []string
	for k, v := range options.Variables {
		cssVars = append(cssVars, fmt.Sprintf("--%s: %s;", k, v))
	}
	
	// Fetch TRMNL CSS framework
	trmnlCSS := r.getTRMNLCSS()
	
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        :root {
            %s
            --screen-width: %dpx;
            --screen-height: %dpx;
        }
        
        %s
        
        %s
    </style>
</head>
<body>
    <div class="trmnl">
        <div class="screen">
            <div class="view view--full">
                %s
            </div>
        </div>
    </div>
</body>
</html>`,
		strings.Join(cssVars, "\n            "),
		options.Width,
		options.Height,
		trmnlCSS,
		options.ExtraCSS,
		html)
}

// DefaultHTMLRenderer creates a renderer with default TRMNL options
func DefaultHTMLRenderer() (*HTMLRenderer, error) {
	return NewHTMLRenderer(DefaultRenderOptions())
}

// Quick render function for simple use cases
func RenderHTML(html string) ([]byte, error) {
	renderer, err := DefaultHTMLRenderer()
	if err != nil {
		return nil, err
	}
	defer renderer.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	return renderer.RenderToImage(ctx, html, DefaultRenderOptions())
}

// Quick template render function
func RenderTemplate(template string, data map[string]interface{}) ([]byte, error) {
	renderer, err := DefaultHTMLRenderer()
	if err != nil {
		return nil, err
	}
	defer renderer.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	return renderer.RenderTemplateToImage(ctx, template, data, DefaultRenderOptions())
}

// Cache for TRMNL CSS to avoid repeated downloads
var (
	trmnlCSSCache string
	trmnlCSSOnce  sync.Once
)

// getTRMNLCSS fetches and caches the TRMNL CSS framework
func (r *HTMLRenderer) getTRMNLCSS() string {
	trmnlCSSOnce.Do(func() {
		resp, err := http.Get("https://usetrmnl.com/css/latest/plugins.css")
		if err != nil {
			log.Printf("[HTML_RENDERER] Failed to fetch TRMNL CSS: %v", err)
			trmnlCSSCache = ""
			return
		}
		defer resp.Body.Close()

		cssBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[HTML_RENDERER] Failed to read TRMNL CSS: %v", err)
			trmnlCSSCache = ""
			return
		}

		trmnlCSSCache = string(cssBytes)
		
		// Fix font loading issues by replacing custom fonts with web-safe alternatives
		trmnlCSSCache = fixFontReferences(trmnlCSSCache)
		
		log.Printf("[HTML_RENDERER] Successfully loaded TRMNL CSS (%d bytes)", len(trmnlCSSCache))
	})

	return trmnlCSSCache
}

// fixFontReferences updates the TRMNL CSS to use absolute URLs for fonts and images
func fixFontReferences(css string) string {
	// Replace relative font paths with absolute URLs to TRMNL's servers
	urlReplacements := map[string]string{
		// Font URLs
		`url("/fonts/BlockKie.ttf")`:                       `url("https://usetrmnl.com/fonts/BlockKie.ttf")`,
		`url("/fonts/dogicapixel.ttf")`:                    `url("https://usetrmnl.com/fonts/dogicapixel.ttf")`,
		`url("/fonts/dogicapixelbold.ttf")`:                `url("https://usetrmnl.com/fonts/dogicapixelbold.ttf")`,
		`url("/fonts/NicoClean-Regular.ttf")`:              `url("https://usetrmnl.com/fonts/NicoClean-Regular.ttf")`,
		`url("/fonts/NicoBold-Regular.ttf")`:               `url("https://usetrmnl.com/fonts/NicoBold-Regular.ttf")`,
		`url("/fonts/NicoPups-Regular.ttf")`:               `url("https://usetrmnl.com/fonts/NicoPups-Regular.ttf")`,
		// Image URLs for backgrounds and borders
		`url("/images/`:                                    `url("https://usetrmnl.com/images/`,
	}
	
	fixedCSS := css
	for oldURL, newURL := range urlReplacements {
		fixedCSS = strings.ReplaceAll(fixedCSS, oldURL, newURL)
	}
	
	return fixedCSS
}

func init() {
	// Set minimal logging for better performance
	log.SetFlags(0)
}