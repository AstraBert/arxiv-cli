mod download;

use crate::download::download_arxiv_papers;
use clap::Parser;

/// Download papers from arXiv by category or search query.
#[derive(Parser, Debug)]
#[command(version = "0.1.0")]
#[command(name = "arxiv-cli")]
#[command(about, long_about = None)]
struct Args {
    /// The category of the papers (e.g., cs.AI, cs.CL)
    #[arg(short, long)]
    category: Option<String>,

    /// Search query (e.g., "graphrag", "machine learning")
    #[arg(short, long)]
    query: Option<String>,

    /// The maximum number of papers to fetch
    #[arg(short, long, default_value_t = 5)]
    limit: i32,

    /// Whether or not to fetch and save the PDF paper
    #[arg(short, long, default_value_t = false)]
    pdf: bool,

    /// Whether or not to save the summary of the papers txt files
    #[arg(short, long, default_value_t = false)]
    summary: bool,

    /// Whether or not to disable fetching and saving the metadata of the paper to a JSONL file
    #[arg(long, default_value_t = false)]
    no_metadata: bool,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();

    // Ensure at least one of category or query is provided
    if args.category.is_none() && args.query.is_none() {
        eprintln!("Error: Either --category or --query must be provided");
        std::process::exit(1);
    }

    // Build search query based on inputs
    let search_query = match (&args.category, &args.query) {
        (Some(cat), Some(q)) => format!("cat:{} AND {}", cat, q),
        (Some(cat), None) => format!("cat:{}", cat),
        (None, Some(q)) => q.clone(),
        (None, None) => unreachable!(),
    };

    download_arxiv_papers(
        search_query,
        args.limit,
        !args.no_metadata,
        args.pdf,
        args.summary,
    )
    .await?;
    Ok(())
}
