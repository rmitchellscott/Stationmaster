package rendering

import (
	"context"
)

// RenderOptions contains options for rendering
type RenderOptions struct {
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Quality   int               `json:"quality"`   // JPEG quality (1-100)
	Format    string            `json:"format"`    // "png" or "jpeg"
	DPI       int               `json:"dpi"`       // DPI for rendering
	WaitTime  int               `json:"wait_time"` // Time to wait for content to load (ms)
	ExtraCSS  string            `json:"extra_css"` // Additional CSS to inject
	Variables map[string]string `json:"variables"` // CSS variables to set
}

// DefaultRenderOptions returns sensible defaults for TRMNL devices
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		Width:    800,  // TRMNL device width
		Height:   480,  // TRMNL device height
		Quality:  90,   // High quality
		Format:   "png",
		DPI:      125,  // TRMNL device DPI
		WaitTime: 2000, // 2 seconds wait for fonts to load
	}
}

// Renderer defines the interface for rendering HTML to images
type Renderer interface {
	// RenderToImage renders HTML content to an image
	RenderToImage(ctx context.Context, html string, options RenderOptions) ([]byte, error)
	
	// RenderTemplateToImage renders a template with data to an image
	RenderTemplateToImage(ctx context.Context, template string, data map[string]interface{}, options RenderOptions) ([]byte, error)
	
	// Close cleans up any resources used by the renderer
	Close() error
}

// RenderResult contains the result of a rendering operation
type RenderResult struct {
	ImageData []byte `json:"-"`
	Format    string `json:"format"`
	Size      int    `json:"size"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}