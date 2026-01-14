package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AstraBert/arxiv-cli/internal/download"
	"github.com/spf13/cobra"
)

var (
	query      string
	limit      int
	pdf        bool
	summary    bool
	noMetadata bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "arxiv-cli",
		Short: "Download papers from arXiv by category or search query",
		Long:  "Intuitive command-line tool to download the most recent number of papers belonging a specific category from arXiv.",
		Version: "1.0.0",
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return fmt.Errorf("query is required (use --query or -q)")
			}

			ctx := context.Background()
			return download.DownloadArxivPapers(
				ctx,
				query,
				limit,
				!noMetadata,
				pdf,
				summary,
			)
		},
	}

	rootCmd.Flags().StringVarP(&query, "query", "q", "", "Search query (e.g., \"graphrag\", \"machine learning\") (required)")
	rootCmd.Flags().IntVarP(&limit, "limit", "l", 5, "The maximum number of papers to fetch")
	rootCmd.Flags().BoolVarP(&pdf, "pdf", "p", false, "Whether or not to fetch and save the PDF paper")
	rootCmd.Flags().BoolVarP(&summary, "summary", "s", false, "Whether or not to save the summary of the papers txt files")
	rootCmd.Flags().BoolVar(&noMetadata, "no-metadata", false, "Whether or not to disable fetching and saving the metadata of the paper to a JSONL file")

	if err := rootCmd.MarkFlagRequired("query"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
