package private

import (
	"context"
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/rendering"
)

// RenderOptions contains all options needed to render a private plugin to HTML
type RenderOptions struct {
	SharedMarkup      string
	LayoutTemplate    string
	Data              map[string]interface{}
	Width             int
	Height            int
	PluginName        string
	InstanceID        string
	InstanceName      string
	RemoveBleedMargin bool
	EnableDarkMode    bool
	Layout            string // Layout type for proper mashup CSS structure (e.g. "full", "half_vertical", "quadrant")
	LayoutWidth       int    // Layout-specific width for positioning
	LayoutHeight      int    // Layout-specific height for positioning
}

// PrivatePluginRenderer handles HTML generation for private plugins
type PrivatePluginRenderer struct {
	baseRenderer    *rendering.BaseHTMLRenderer
	unifiedRenderer *rendering.UnifiedRenderer
}

// NewPrivatePluginRenderer creates a new private plugin renderer
func NewPrivatePluginRenderer(appDir string) (*PrivatePluginRenderer, error) {
	// Check if external Ruby service is available
	unifiedRenderer := rendering.NewUnifiedRenderer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if !unifiedRenderer.IsServiceAvailable(ctx) {
		return nil, fmt.Errorf("external Ruby service is not available - required for private plugins")
	}
	
	return &PrivatePluginRenderer{
		baseRenderer:    rendering.NewBaseHTMLRenderer(),
		unifiedRenderer: unifiedRenderer,
	}, nil
}

// getLayoutMashupInfo returns mashup CSS class and slot position for a given layout
func getLayoutMashupInfo(layout string) (string, string) {
	switch layout {
	case "half_vertical":
		return "mashup--1Lx1R", "left" // Could also be "right", but "left" works for preview
	case "half_horizontal":
		return "mashup--1Tx1B", "top"  // Could also be "bottom", but "top" works for preview
	case "quadrant":
		return "mashup--2Lx2R", "q1"   // Top-left quadrant for preview
	default:
		return "", "" // Full layout doesn't need mashup wrapper
	}
}


// RenderToServerSideHTML generates HTML using external Ruby service
func (r *PrivatePluginRenderer) RenderToServerSideHTML(ctx context.Context, opts RenderOptions) (string, error) {
	// Convert private plugin render options to unified renderer options
	unifiedOpts := rendering.PluginRenderOptions{
		SharedMarkup:      opts.SharedMarkup,
		LayoutTemplate:    opts.LayoutTemplate,
		Data:              opts.Data,
		Width:             opts.Width,
		Height:            opts.Height,
		PluginName:        opts.PluginName,
		InstanceID:        opts.InstanceID,
		InstanceName:      opts.InstanceName,
		RemoveBleedMargin: opts.RemoveBleedMargin,
		EnableDarkMode:    opts.EnableDarkMode,
		Layout:            opts.Layout,
		LayoutWidth:       opts.LayoutWidth,
		LayoutHeight:      opts.LayoutHeight,
	}
	
	return r.unifiedRenderer.RenderToHTML(ctx, unifiedOpts)
}


