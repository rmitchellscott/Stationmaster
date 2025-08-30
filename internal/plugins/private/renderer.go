package private

import (
	"context"
	"fmt"

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
	baseRenderer     *rendering.BaseHTMLRenderer
	rubyRenderer     *rendering.RubyLiquidRenderer
}

// NewPrivatePluginRenderer creates a new private plugin renderer
func NewPrivatePluginRenderer(appDir string) (*PrivatePluginRenderer, error) {
	rubyRenderer, err := rendering.NewRubyLiquidRenderer(appDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Ruby renderer (required for private plugins): %w", err)
	}
	
	return &PrivatePluginRenderer{
		baseRenderer: rendering.NewBaseHTMLRenderer(),
		rubyRenderer: rubyRenderer,
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


// RenderToServerSideHTML generates HTML using Ruby server-side liquid rendering
func (r *PrivatePluginRenderer) RenderToServerSideHTML(ctx context.Context, opts RenderOptions) (string, error) {
	
	// Convert private plugin render options to Ruby renderer options
	rubyOpts := rendering.PluginRenderOptions{
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
	
	return r.rubyRenderer.RenderToHTML(ctx, rubyOpts)
}


