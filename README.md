# arxiv-cli

Intuitive command-line tool to download the most recent number of papers belonging a specific category from arXiv.

## Installation

```bash
# with npm
npm install @cle-does-things/arxiv-cli

# with cargo
cargo install arxiv-cli
```

Check installation:

```bash
arxiv-cli --help
```

## Usage

```bash
arxiv-cli [OPTIONS] --category <CATEGORY>
```

**Options:**

- `-c`, `--category <CATEGORY>`: The category of the papers (required)
- `-l`, `--limit <LIMIT>`: The maximum number of papers to fetch (default: 5)
- `-p`, `--pdf`: Fetch and save the PDF of each paper
- `-s`, `--summary`: Save the summary of each paper as a `.txt` file
- `--no-metadata`: Disable fetching and saving metadata to a `.jsonl` file
- `-h`, `--help`: Print help information
- `-V`, `--version`: Print version information