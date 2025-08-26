package logging

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// findProjectRoot finds the project root directory by looking for go.mod
func findProjectRoot(t *testing.T) string {
	// Get the directory of this test file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Unable to get current file path")
	}
	
	// Start from the test file directory and walk up
	dir := filepath.Dir(filename)
	for {
		// Check if go.mod exists in current directory
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		
		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory, go.mod not found
			t.Fatal("Could not find go.mod to determine project root")
		}
		dir = parent
	}
}

// TestNoDirectLogging ensures all Go files use structured logging through the logging package
// instead of direct fmt.Printf, log.Printf, print, or println calls
func TestNoDirectLogging(t *testing.T) {
	// Find the project root directory
	projectRoot := findProjectRoot(t)
	// Patterns to detect direct logging calls
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\bfmt\.Printf\s*\(`),
		regexp.MustCompile(`\bfmt\.Print\s*\(`),
		regexp.MustCompile(`\bfmt\.Println\s*\(`),
		regexp.MustCompile(`\blog\.Printf\s*\(`),
		regexp.MustCompile(`\blog\.Print\s*\(`),
		regexp.MustCompile(`\blog\.Println\s*\(`),
		regexp.MustCompile(`\bprintln\s*\(`),
		regexp.MustCompile(`\bprint\s*\(`),
	}

	// Files/directories to exclude from linting
	excludePatterns := []*regexp.Regexp{
		regexp.MustCompile(`_test\.go$`),                    // Test files
		regexp.MustCompile(`main\.go$`),                     // Allow version output in main.go
		regexp.MustCompile(`/vendor/`),                      // Vendor directory
		regexp.MustCompile(`/node_modules/`),                // Node modules
		regexp.MustCompile(`/ui/`),                          // UI directory
		regexp.MustCompile(`/docs/`),                        // Documentation
		regexp.MustCompile(`\.md$`),                         // Markdown files
		regexp.MustCompile(`internal/logging/lint_test\.go`), // This test file
	}

	var violations []string

	// Walk through all Go files in the project
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only check Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Check if file should be excluded
		relativePath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			t.Logf("Warning: could not get relative path for %s: %v", path, err)
			return nil
		}
		for _, excludePattern := range excludePatterns {
			if excludePattern.MatchString(relativePath) {
				return nil
			}
		}

		// Read and scan the file
		file, err := os.Open(path)
		if err != nil {
			t.Logf("Warning: could not open file %s: %v", path, err)
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Skip comment lines
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}

			// Check for violations
			for _, pattern := range patterns {
				if pattern.MatchString(line) {
					violations = append(violations, 
						fmt.Sprintf("%s:%d: %s",
							relativePath,
							lineNum,
							strings.TrimSpace(line)))
				}
			}
		}

		return scanner.Err()
	})

	if err != nil {
		t.Fatalf("Error walking directory tree: %v", err)
	}

	// Report violations
	if len(violations) > 0 {
		t.Errorf("Found %d direct logging violations. All logging should use the logging package:\n", len(violations))
		for _, violation := range violations {
			t.Errorf("  %s", violation)
		}
		t.Errorf("\nUse logging.InfoWithComponent(), logging.WarnWithComponent(), logging.ErrorWithComponent(), etc. instead")
	}
}