package ruby

import (
	"context"
	"testing"
	"time"
)

func TestLiquidRenderer_Basic(t *testing.T) {
	// Skip if we're not in the Docker environment with Ruby
	renderer, err := NewLiquidRenderer(".")
	if err != nil {
		t.Skipf("Skipping Ruby tests - Ruby environment not available: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	template := "Hello {{ name }}!"
	data := map[string]interface{}{
		"name": "World",
	}
	
	result, err := renderer.RenderTemplate(ctx, template, data)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}
	
	expected := "Hello World!"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestLiquidRenderer_TRNMLFilters(t *testing.T) {
	// Skip if we're not in the Docker environment with Ruby
	renderer, err := NewLiquidRenderer(".")
	if err != nil {
		t.Skipf("Skipping Ruby tests - Ruby environment not available: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	template := "Count: {{ count | number_with_delimiter }}"
	data := map[string]interface{}{
		"count": 1337,
	}
	
	result, err := renderer.RenderTemplate(ctx, template, data)
	if err != nil {
		t.Fatalf("Failed to render template with TRMNL filter: %v", err)
	}
	
	expected := "Count: 1,337"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestLiquidRenderer_ComplexData(t *testing.T) {
	// Skip if we're not in the Docker environment with Ruby
	renderer, err := NewLiquidRenderer(".")
	if err != nil {
		t.Skipf("Skipping Ruby tests - Ruby environment not available: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	template := `<div class="view">
  <h1>{{ user.name }}</h1>
  <p>Temperature: {{ weather.temperature }}°F</p>
</div>`
	
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "Test User",
		},
		"weather": map[string]interface{}{
			"temperature": 72,
		},
	}
	
	result, err := renderer.RenderTemplate(ctx, template, data)
	if err != nil {
		t.Fatalf("Failed to render complex template: %v", err)
	}
	
	// Check for key content
	if !contains(result, "Test User") {
		t.Errorf("Result should contain 'Test User', got: %s", result)
	}
	if !contains(result, "72°F") {
		t.Errorf("Result should contain '72°F', got: %s", result)
	}
	if !contains(result, `class="view"`) {
		t.Errorf("Result should contain class='view', got: %s", result)
	}
}

func TestLiquidRenderer_ValidationError(t *testing.T) {
	// Skip if we're not in the Docker environment with Ruby
	renderer, err := NewLiquidRenderer(".")
	if err != nil {
		t.Skipf("Skipping Ruby tests - Ruby environment not available: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Invalid template syntax
	template := "{{ unclosed_tag"
	data := map[string]interface{}{}
	
	_, err = renderer.RenderTemplate(ctx, template, data)
	if err == nil {
		t.Fatalf("Expected error for invalid template syntax, but got none")
	}
	
	// Should contain error information
	if !contains(err.Error(), "syntax error") && !contains(err.Error(), "Ruby liquid rendering failed") {
		t.Errorf("Error should mention syntax error, got: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     containsAt(s, substr, 1)))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}
	if start+len(substr) <= len(s) && s[start:start+len(substr)] == substr {
		return true
	}
	return containsAt(s, substr, start+1)
}