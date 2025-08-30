package rendering

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRubyLiquidRenderer_Integration(t *testing.T) {
	// Skip if we're not in the Docker environment with Ruby
	renderer, err := NewRubyLiquidRenderer(".")
	if err != nil {
		t.Skipf("Skipping Ruby integration tests - Ruby environment not available: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	opts := RenderOptions{
		SharedMarkup:   `<div class="shared">Shared content</div>`,
		LayoutTemplate: `<div class="layout">{{ shared }}{{ content }}</div>`,
		Data: map[string]interface{}{
			"content": "Hello from Ruby!",
			"shared":  "Shared data",
		},
		Width:      480,
		Height:     800,
		PluginName: "Test Plugin",
		InstanceID: "test-123",
	}
	
	html, err := renderer.RenderToHTML(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}
	
	// Verify HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("Output should contain DOCTYPE declaration")
	}
	
	if !strings.Contains(html, "Test Plugin") {
		t.Error("Output should contain plugin name as title")
	}
	
	if !strings.Contains(html, "width: 480px") {
		t.Error("Output should contain correct width")
	}
	
	if !strings.Contains(html, "height: 800px") {
		t.Error("Output should contain correct height")
	}
	
	// Check for TRMNL structure
	if !strings.Contains(html, `id="plugin-test-123"`) {
		t.Error("Output should contain plugin instance ID")
	}
	
	if !strings.Contains(html, `class="environment trmnl"`) {
		t.Error("Output should contain TRMNL environment classes")
	}
	
	// Verify no liquidjs is loaded
	if strings.Contains(html, "liquidjs") {
		t.Error("Output should not contain liquidjs references")
	}
	
	// Verify TRMNL scripts are still loaded
	if !strings.Contains(html, "usetrmnl.com/js/latest/plugins.js") {
		t.Error("Output should contain TRMNL plugins script")
	}
}

func TestRubyLiquidRenderer_TRNMLFilters(t *testing.T) {
	// Skip if we're not in the Docker environment with Ruby
	renderer, err := NewRubyLiquidRenderer(".")
	if err != nil {
		t.Skipf("Skipping Ruby integration tests - Ruby environment not available: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	opts := RenderOptions{
		LayoutTemplate: `<div>Count: {{ count | number_with_delimiter }}</div>`,
		Data: map[string]interface{}{
			"count": 1337,
		},
		Width:      480,
		Height:     800,
		PluginName: "TRMNL Filter Test",
		InstanceID: "test-filter",
	}
	
	html, err := renderer.RenderToHTML(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to render HTML with TRMNL filters: %v", err)
	}
	
	// Check that the filter was applied
	if !strings.Contains(html, "1,337") {
		t.Errorf("Output should contain formatted number '1,337', got: %s", html)
	}
}

func TestRubyLiquidRenderer_ErrorHandling(t *testing.T) {
	// Skip if we're not in the Docker environment with Ruby
	renderer, err := NewRubyLiquidRenderer(".")
	if err != nil {
		t.Skipf("Skipping Ruby integration tests - Ruby environment not available: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	opts := RenderOptions{
		LayoutTemplate: `{{ invalid_syntax`,
		Data:           map[string]interface{}{},
		Width:          480,
		Height:         800,
		PluginName:     "Error Test",
		InstanceID:     "test-error",
	}
	
	_, err = renderer.RenderToHTML(ctx, opts)
	if err == nil {
		t.Fatal("Expected error for invalid template syntax, but got none")
	}
	
	// Should be a Ruby liquid rendering error
	if !strings.Contains(err.Error(), "Ruby liquid rendering failed") {
		t.Errorf("Error should mention Ruby liquid rendering, got: %v", err)
	}
}