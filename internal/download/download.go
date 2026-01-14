package download

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	JSONFile     = "metadata.jsonl"
	PDFDirectory = "pdfs/"
	TextDirectory = "texts/"
	arxivAPIBase = "http://export.arxiv.org/api/query"
)

type ArxivPaper struct {
	ID              string   `json:"id"`
	Updated         string   `json:"updated"`
	Published       string   `json:"published"`
	Title           string   `json:"title"`
	Summary         string   `json:"-"` // skip in JSON like Rust
	Authors         []string `json:"authors"`
	PrimaryCategory string   `json:"primary_category"`
	Categories      []string `json:"categories"`
	PDFURL          string   `json:"pdf_url"`
	HTMLURL         string   `json:"html_url"`
	Comment         *string  `json:"comment,omitempty"`
}

// Atom XML structures for parsing arXiv API response
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []Entry  `xml:"entry"`
}

type Entry struct {
	XMLName    xml.Name   `xml:"entry"`
	ID         string     `xml:"id"`
	Updated    string     `xml:"updated"`
	Published  string     `xml:"published"`
	Title      string     `xml:"title"`
	Summary    string     `xml:"summary"`
	Authors    []Author   `xml:"author"`
	Links      []Link     `xml:"link"`
	Categories []Category `xml:"category"`
	Comment    Comment    `xml:"http://arxiv.org/schemas/atom comment"`
}

type Comment struct {
	XMLName xml.Name `xml:"http://arxiv.org/schemas/atom comment"`
	Value   string   `xml:",chardata"`
}

type Author struct {
	Name string `xml:"name"`
}

type Link struct {
	Type  string `xml:"type,attr"`
	HRef  string `xml:"href,attr"`
	Rel   string `xml:"rel,attr"`
	Title string `xml:"title,attr"`
}

type Category struct {
	Term string `xml:"term,attr"`
}

func sanitizeFilename(name string) string {
	invalidChars := []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*'}
	sanitized := name
	for _, ch := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, string(ch), "_")
	}
	sanitized = strings.TrimSpace(sanitized)
	sanitized = strings.TrimRight(sanitized, ".")
	if len(sanitized) > 200 {
		sanitized = sanitized[:200]
	}
	return sanitized
}

func (p *ArxivPaper) FetchPDF(ctx context.Context, outPath string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.PDFURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch PDF: HTTP %d", resp.StatusCode)
	}

	if !strings.HasSuffix(outPath, ".pdf") {
		outPath += ".pdf"
	}

	file, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	return nil
}

func (p *ArxivPaper) WriteSummary(outPath string) error {
	if !strings.HasSuffix(outPath, ".txt") {
		outPath += ".txt"
	}
	return os.WriteFile(outPath, []byte(p.Summary), 0644)
}

func fetchArxivPapers(ctx context.Context, searchQuery string, numResults int) ([]ArxivPaper, error) {
	baseURL, err := url.Parse(arxivAPIBase)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	params := url.Values{}
	params.Set("search_query", searchQuery)
	params.Set("start", "0")
	params.Set("max_results", fmt.Sprintf("%d", numResults))
	params.Set("sortBy", "submittedDate")
	params.Set("sortOrder", "descending")
	baseURL.RawQuery = params.Encode()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from arXiv API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arXiv API returned HTTP %d", resp.StatusCode)
	}

	var feed Feed
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&feed); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	papers := make([]ArxivPaper, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		paper := ArxivPaper{
			ID:              entry.ID,
			Updated:         entry.Updated,
			Published:       entry.Published,
			Title:           strings.TrimSpace(entry.Title),
			Summary:         strings.TrimSpace(entry.Summary),
			Authors:         make([]string, 0, len(entry.Authors)),
			PrimaryCategory: "",
			Categories:      make([]string, 0, len(entry.Categories)),
			Comment:         nil,
		}

		for _, author := range entry.Authors {
			paper.Authors = append(paper.Authors, author.Name)
		}

		for _, category := range entry.Categories {
			paper.Categories = append(paper.Categories, category.Term)
			if paper.PrimaryCategory == "" {
				paper.PrimaryCategory = category.Term
			}
		}

		for _, link := range entry.Links {
			if link.Rel == "alternate" && link.Type == "text/html" {
				paper.HTMLURL = strings.ReplaceAll(link.HRef, "httpss", "https")
			} else if link.Title == "pdf" {
				paper.PDFURL = strings.ReplaceAll(link.HRef, "httpss", "https")
			} else if link.Type == "application/pdf" {
				paper.PDFURL = strings.ReplaceAll(link.HRef, "httpss", "https")
			}
		}

		if entry.Comment.Value != "" {
			comment := entry.Comment.Value
			paper.Comment = &comment
		}

		papers = append(papers, paper)
	}

	return papers, nil
}

func DownloadArxivPapers(ctx context.Context, searchQuery string, numResults int, saveMetadata, savePDFs, saveSummaries bool) error {
	papers, err := fetchArxivPapers(ctx, searchQuery, numResults)
	if err != nil {
		return fmt.Errorf("failed to fetch papers: %w", err)
	}

	var jsonlLines []string

	for _, paper := range papers {
		if saveMetadata {
			paperCopy := paper
			metadataJSON, err := json.Marshal(paperCopy)
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			jsonlLines = append(jsonlLines, string(metadataJSON))
		}

		if savePDFs {
			if err := os.MkdirAll(PDFDirectory, 0755); err != nil {
				return fmt.Errorf("failed to create PDF directory: %w", err)
			}
			sanitizedTitle := sanitizeFilename(paper.Title)
			path := filepath.Join(PDFDirectory, sanitizedTitle)
			if err := paper.FetchPDF(ctx, path); err != nil {
				return fmt.Errorf("failed to fetch PDF for %s: %w", paper.Title, err)
			}
		}

		if saveSummaries {
			if err := os.MkdirAll(TextDirectory, 0755); err != nil {
				return fmt.Errorf("failed to create text directory: %w", err)
			}
			sanitizedTitle := sanitizeFilename(paper.Title)
			path := filepath.Join(TextDirectory, sanitizedTitle+".txt")
			if err := paper.WriteSummary(path); err != nil {
				return fmt.Errorf("failed to write summary for %s: %w", paper.Title, err)
			}
		}
	}

	if len(jsonlLines) > 0 {
		content := strings.Join(jsonlLines, "\n") + "\n"
		if err := os.WriteFile(JSONFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write metadata file: %w", err)
		}
	}

	return nil
}
