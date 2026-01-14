package download

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replace invalid characters",
			input:    "x < y | x > y? better: /, \"\\\" or *",
			expected: "x _ y _ x _ y_ better_ _, ___ or _",
		},
		{
			name:     "truncate long filename",
			input:    "Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dictas sunt",
			expected: "Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dictas",
		},
		{
			name:     "trim whitespace and dots",
			input:    "  test file.  ",
			expected: "test file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestArxivPaperWriteSummary(t *testing.T) {
	paper := ArxivPaper{
		Title:   "test_title",
		Summary: "This is a test summary.",
	}

	outPath := "test_summary.txt"
	t.Cleanup(func() {
		os.Remove(outPath)
	})

	if err := paper.WriteSummary(outPath); err != nil {
		t.Fatalf("WriteSummary() error = %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("Failed to read summary file: %v", err)
	}

	if string(content) != "This is a test summary." {
		t.Errorf("WriteSummary() wrote %q, want %q", string(content), "This is a test summary.")
	}
}

func TestArxivPaperJSONSerialization(t *testing.T) {
	paper := ArxivPaper{
		ID:              "test-id",
		Title:           "test_title",
		Summary:         "This is a test summary.",
		Authors:         []string{"Author 1", "Author 2"},
		PrimaryCategory: "cs.CL",
		Categories:      []string{"cs.CL"},
		PDFURL:          "https://arxiv.org/pdf/test.pdf",
		HTMLURL:         "https://arxiv.org/abs/test",
	}

	jsonData, err := json.Marshal(paper)
	if err != nil {
		t.Fatalf("Failed to marshal paper: %v", err)
	}

	jsonStr := string(jsonData)
	if strings.Contains(jsonStr, "summary") {
		t.Error("JSON serialization should not include summary field")
	}

	if !strings.Contains(jsonStr, "test_title") {
		t.Error("JSON serialization should include title")
	}
}

func TestDownloadArxivPapersIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Clean up any existing files/directories
	t.Cleanup(func() {
		os.Remove(JSONFile)
		os.RemoveAll(PDFDirectory)
		os.RemoveAll(TextDirectory)
	})

	// Remove existing files before test
	os.Remove(JSONFile)
	os.RemoveAll(PDFDirectory)
	os.RemoveAll(TextDirectory)

	ctx := testingContext(t)
	err := DownloadArxivPapers(ctx, "cat:cs.CL", 2, true, false, false)
	if err != nil {
		t.Fatalf("DownloadArxivPapers() error = %v", err)
	}

	// Check metadata file exists
	if _, err := os.Stat(JSONFile); os.IsNotExist(err) {
		t.Error("metadata.jsonl file was not created")
	}

	// Check metadata file has content
	content, err := os.ReadFile(JSONFile)
	if err != nil {
		t.Fatalf("Failed to read metadata file: %v", err)
	}
	if len(content) == 0 {
		t.Error("metadata.jsonl file is empty")
	}
}

func TestDownloadArxivPapersPDFs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Cleanup(func() {
		os.Remove(JSONFile)
		os.RemoveAll(PDFDirectory)
		os.RemoveAll(TextDirectory)
	})

	os.Remove(JSONFile)
	os.RemoveAll(PDFDirectory)
	os.RemoveAll(TextDirectory)

	ctx := testingContext(t)
	err := DownloadArxivPapers(ctx, "cat:cs.CL", 2, false, true, false)
	if err != nil {
		t.Fatalf("DownloadArxivPapers() error = %v", err)
	}

	// Check PDF directory exists
	if _, err := os.Stat(PDFDirectory); os.IsNotExist(err) {
		t.Error("PDF directory was not created")
	}

	// Count PDF files
	entries, err := os.ReadDir(PDFDirectory)
	if err != nil {
		t.Fatalf("Failed to read PDF directory: %v", err)
	}

	pdfCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pdf") {
			pdfCount++
		}
	}

	if pdfCount != 2 {
		t.Errorf("Expected 2 PDF files, got %d", pdfCount)
	}
}

func TestDownloadArxivPapersSummaries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Cleanup(func() {
		os.Remove(JSONFile)
		os.RemoveAll(PDFDirectory)
		os.RemoveAll(TextDirectory)
	})

	os.Remove(JSONFile)
	os.RemoveAll(PDFDirectory)
	os.RemoveAll(TextDirectory)

	ctx := testingContext(t)
	err := DownloadArxivPapers(ctx, "cat:cs.CL", 2, false, false, true)
	if err != nil {
		t.Fatalf("DownloadArxivPapers() error = %v", err)
	}

	// Check text directory exists
	if _, err := os.Stat(TextDirectory); os.IsNotExist(err) {
		t.Error("Text directory was not created")
	}

	// Count text files
	entries, err := os.ReadDir(TextDirectory)
	if err != nil {
		t.Fatalf("Failed to read text directory: %v", err)
	}

	textCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			textCount++
		}
	}

	if textCount != 2 {
		t.Errorf("Expected 2 text files, got %d", textCount)
	}
}

func TestDownloadArxivPapersAll(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Cleanup(func() {
		os.Remove(JSONFile)
		os.RemoveAll(PDFDirectory)
		os.RemoveAll(TextDirectory)
	})

	os.Remove(JSONFile)
	os.RemoveAll(PDFDirectory)
	os.RemoveAll(TextDirectory)

	ctx := testingContext(t)
	err := DownloadArxivPapers(ctx, "cat:cs.CL", 2, true, true, true)
	if err != nil {
		t.Fatalf("DownloadArxivPapers() error = %v", err)
	}

	// Check all outputs exist
	if _, err := os.Stat(JSONFile); os.IsNotExist(err) {
		t.Error("metadata.jsonl file was not created")
	}

	if _, err := os.Stat(PDFDirectory); os.IsNotExist(err) {
		t.Error("PDF directory was not created")
	}

	if _, err := os.Stat(TextDirectory); os.IsNotExist(err) {
		t.Error("Text directory was not created")
	}

	// Verify PDF count
	pdfEntries, _ := os.ReadDir(PDFDirectory)
	pdfCount := 0
	for _, entry := range pdfEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pdf") {
			pdfCount++
		}
	}
	if pdfCount != 2 {
		t.Errorf("Expected 2 PDF files, got %d", pdfCount)
	}

	// Verify text count
	textEntries, _ := os.ReadDir(TextDirectory)
	textCount := 0
	for _, entry := range textEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			textCount++
		}
	}
	if textCount != 2 {
		t.Errorf("Expected 2 text files, got %d", textCount)
	}
}

// Helper function to create a context for testing
func testingContext(t *testing.T) context.Context {
	ctx := context.Background()
	// Add timeout for tests if needed
	return ctx
}
