package private

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// PluginContentComparator handles comparison of plugin content to detect changes
type PluginContentComparator struct{}

// NewPluginContentComparator creates a new content comparator
func NewPluginContentComparator() *PluginContentComparator {
	return &PluginContentComparator{}
}

// ContentComparisonResult contains the result of comparing plugin content
type ContentComparisonResult struct {
	HasChanged bool   // Whether the content has changed since last render
	NewHash    string // Hash of the new content
	OldHash    string // Hash of the previous content (if any)
}

// CompareHTML compares new HTML content against the stored hash
// Returns true if content has changed or if no previous hash exists
func (c *PluginContentComparator) CompareHTML(newHTML string, previousHash *string) ContentComparisonResult {
	newHash := c.HashHTML(newHTML)
	
	result := ContentComparisonResult{
		NewHash: newHash,
		OldHash: "",
	}
	
	if previousHash != nil {
		result.OldHash = *previousHash
		result.HasChanged = newHash != *previousHash
	} else {
		// No previous hash - treat as changed (first render)
		result.HasChanged = true
	}
	
	return result
}

// HashHTML creates a SHA256 hash of the HTML content
// The hash includes basic normalization to avoid false positives from whitespace changes
func (c *PluginContentComparator) HashHTML(html string) string {
	// Normalize HTML for more consistent hashing
	normalizedHTML := c.normalizeHTML(html)
	
	hash := sha256.Sum256([]byte(normalizedHTML))
	return fmt.Sprintf("%x", hash)
}

// normalizeHTML performs basic HTML normalization to avoid false positives
func (c *PluginContentComparator) normalizeHTML(html string) string {
	// For now, just trim whitespace
	// Could be extended to normalize more HTML formatting differences
	return strings.TrimSpace(html)
}

// CompareImage compares new image bytes against the stored hash
// Returns true if content has changed or if no previous hash exists
func (c *PluginContentComparator) CompareImage(imageBytes []byte, previousHash *string) ContentComparisonResult {
	newHash := c.HashImage(imageBytes)
	
	result := ContentComparisonResult{
		NewHash: newHash,
		OldHash: "",
	}
	
	if previousHash != nil {
		result.OldHash = *previousHash
		result.HasChanged = newHash != *previousHash
	} else {
		// No previous hash - treat as changed (first render)
		result.HasChanged = true
	}
	
	return result
}

// HashImage creates a SHA256 hash of the image bytes
func (c *PluginContentComparator) HashImage(imageBytes []byte) string {
	hash := sha256.Sum256(imageBytes)
	return fmt.Sprintf("%x", hash)
}

// ShouldSkipRender determines if rendering should be skipped based on content comparison
func (c *PluginContentComparator) ShouldSkipRender(comparison ContentComparisonResult) bool {
	return !comparison.HasChanged
}