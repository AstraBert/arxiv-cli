use std::{fs, io::Write};

use arxiv::{Arxiv, ArxivQueryBuilder};
use serde::{Deserialize, Serialize};

/// Sanitize a filename to be Windows-compatible
fn sanitize_filename(name: &str) -> String {
    // Replace invalid Windows filename characters with underscores
    let invalid_chars = ['<', '>', ':', '"', '/', '\\', '|', '?', '*'];
    let mut sanitized = name.to_string();
    for ch in invalid_chars {
        sanitized = sanitized.replace(ch, "_");
    }
    // Trim leading/trailing whitespace and dots
    sanitized = sanitized.trim().trim_end_matches('.').to_string();
    // Limit filename length to 200 characters to be safe
    if sanitized.len() > 200 {
        sanitized.truncate(200);
    }
    sanitized
}

const JSON_FILE: &str = "metadata.jsonl";
const PDF_DIRECTORY: &str = "pdfs/";
const TEXT_DIRECTORY: &str = "texts/";

#[derive(Serialize, Deserialize, Clone)]
pub struct SerDesArxiv {
    pub id: String,
    pub updated: String,
    pub published: String,
    pub title: String,
    #[serde(skip_serializing)]
    pub summary: String,
    pub authors: Vec<String>,
    pub primary_category: String,
    pub categories: Vec<String>,
    pub pdf_url: String,
    pub html_url: String,
    pub comment: Option<String>,
}

impl SerDesArxiv {
    fn from_arxiv(arxiv_paper: Arxiv) -> Self {
        Self {
            id: arxiv_paper.id,
            updated: arxiv_paper.updated,
            published: arxiv_paper.published,
            title: arxiv_paper.title,
            summary: arxiv_paper.summary,
            authors: arxiv_paper.authors,
            primary_category: arxiv_paper.primary_category,
            categories: arxiv_paper.categories,
            pdf_url: arxiv_paper.pdf_url.replace("httpss", "https"),
            html_url: arxiv_paper.html_url.replace("httpss", "https"),
            comment: arxiv_paper.comment,
        }
    }

    pub async fn fetch_pdf(&self, out_path: &str) -> anyhow::Result<()> {
        let body = reqwest::get(&self.pdf_url).await?.bytes().await?;
        let out_path = if out_path.ends_with(".pdf") {
            out_path.to_string()
        } else {
            format!("{}.pdf", out_path)
        };
        let mut file = fs::File::create(out_path)?;
        file.write_all(&body)?;
        Ok(())
    }

    // TODO: make this function actually usable
    // pub async fn fetch_text(&self, out_path: &str) -> anyhow::Result<()> {
    //     let body = reqwest::get(&self.html_url).await?.bytes().await?;
    //     let html_text = from_read(&body[..], 20)?;
    //     let out_path = if out_path.ends_with(".txt") {
    //         out_path.to_string()
    //     } else {
    //         format!("{}.txt", out_path)
    //     };
    //     fs::write(out_path, &html_text)?;
    //     Ok(())
    // }

    pub fn write_summary(&self, out_path: &str) -> anyhow::Result<()> {
        let out_path = if out_path.ends_with(".txt") {
            out_path.to_string()
        } else {
            format!("{}.txt", out_path)
        };
        let summary = self.summary.clone();
        fs::write(out_path, summary)?;
        Ok(())
    }
}

pub async fn download_arxiv_papers(
    search_query: String,
    num_results: i32,
    save_metadata: bool,
    save_pdfs: bool,
    save_summaries: bool,
) -> anyhow::Result<()> {
    let query = ArxivQueryBuilder::new()
        .search_query(&search_query)
        .start(0)
        .max_results(num_results)
        .sort_by("submittedDate")
        .sort_order("descending")
        .build();
    let arxivs = arxiv::fetch_arxivs(query).await?;
    let mut jsonl_text: String = "".to_string();
    for a in arxivs {
        let paper = SerDesArxiv::from_arxiv(a);
        if save_metadata {
            let paper_copy = paper.clone();
            let paper_metadata = serde_json::to_string(&paper_copy)?;
            jsonl_text += &format!("{}\n", paper_metadata);
        }
        if save_pdfs {
            let pdf_dir_exists = fs::exists(PDF_DIRECTORY)?;
            if !pdf_dir_exists {
                fs::create_dir(PDF_DIRECTORY)?;
            }
            let sanitized_title = sanitize_filename(&paper.title);
            let path = format!("{}/{}", PDF_DIRECTORY, sanitized_title);
            paper.fetch_pdf(&path).await?;
        }
        if save_summaries {
            let txt_dir_exists = fs::exists(TEXT_DIRECTORY)?;
            if !txt_dir_exists {
                fs::create_dir(TEXT_DIRECTORY)?;
            }
            let sanitized_title = sanitize_filename(&paper.title);
            let path = format!("{}/{}.txt", TEXT_DIRECTORY, sanitized_title);
            paper.write_summary(&path)?;
        }
    }
    if !jsonl_text.is_empty() {
        fs::write(JSON_FILE, &jsonl_text)?;
    }
    Ok(())
}

#[cfg(test)]
mod test {
    use super::*;
    use serial_test::serial;
    use std::path::Path;

    #[tokio::test]
    #[serial]
    async fn integration_test_defaults() {
        if Path::new(PDF_DIRECTORY).exists() {
            fs::remove_dir_all(PDF_DIRECTORY).expect("Should be able to remove PDF directory");
        }
        if Path::new(TEXT_DIRECTORY).exists() {
            fs::remove_dir_all(TEXT_DIRECTORY).expect("Should be able to remove text directory");
        }
        if Path::new(JSON_FILE).exists() {
            fs::remove_file(JSON_FILE).expect("Should be able to remove metadata.jsonl file");
        }
        let result = download_arxiv_papers("cs.CL".to_string(), 5, true, false, false).await;
        match result {
            Ok(_) => {}
            Err(e) => {
                eprintln!("An error occurred: {}", e.to_string());
                assert!(false)
            }
        }
        let file_exists = fs::exists(JSON_FILE)
            .expect("Should be able to check the existance of the metadata.jsonl file");
        assert!(file_exists);
        let content =
            fs::read_to_string(JSON_FILE).expect("Should be able to read metadata.jsonl file");
        assert!(content.len() > 0);
    }

    #[tokio::test]
    #[serial]
    async fn integration_test_pdfs() {
        if Path::new(PDF_DIRECTORY).exists() {
            fs::remove_dir_all(PDF_DIRECTORY).expect("Should be able to remove PDF directory");
        }
        if Path::new(TEXT_DIRECTORY).exists() {
            fs::remove_dir_all(TEXT_DIRECTORY).expect("Should be able to remove text directory");
        }
        if Path::new(JSON_FILE).exists() {
            fs::remove_file(JSON_FILE).expect("Should be able to remove metadata.jsonl file");
        }
        let result = download_arxiv_papers("cs.CL".to_string(), 2, false, true, false).await;
        match result {
            Ok(_) => {}
            Err(e) => {
                eprintln!("An error occurred: {}", e.to_string());
                assert!(false)
            }
        }
        let file_exists = fs::exists(JSON_FILE)
            .expect("Should be able to check the existance of the metadata.jsonl file");
        assert!(!file_exists);
        let dir_exists: bool = fs::exists(PDF_DIRECTORY)
            .expect("Should be able to check the existance of the PDF directory");
        assert!(dir_exists);
        let dir_content =
            fs::read_dir(PDF_DIRECTORY).expect("Should be able to read the PDF directory");
        let mut count = 0;
        for entry in dir_content {
            let _dir_entry = entry.expect("Should be able to read entry");
            count += 1;
        }
        assert_eq!(count, 2);
    }

    #[tokio::test]
    #[serial]
    async fn integration_test_texts() {
        if Path::new(PDF_DIRECTORY).exists() {
            fs::remove_dir_all(PDF_DIRECTORY).expect("Should be able to remove PDF directory");
        }
        if Path::new(TEXT_DIRECTORY).exists() {
            fs::remove_dir_all(TEXT_DIRECTORY).expect("Should be able to remove text directory");
        }
        if Path::new(JSON_FILE).exists() {
            fs::remove_file(JSON_FILE).expect("Should be able to remove metadata.jsonl file");
        }
        let result = download_arxiv_papers("cs.CL".to_string(), 2, false, false, true).await;
        match result {
            Ok(_) => {}
            Err(e) => {
                eprintln!("An error occurred: {}", e.to_string());
                assert!(false)
            }
        }
        let file_exists = fs::exists(JSON_FILE)
            .expect("Should be able to check the existance of the metadata.jsonl file");
        assert!(!file_exists);
        let pdf_dir_exists: bool = fs::exists(PDF_DIRECTORY)
            .expect("Should be able to check the existance of the PDF directory");
        assert!(!pdf_dir_exists);
        let text_dir_exists: bool = fs::exists(TEXT_DIRECTORY)
            .expect("Should be able to check the existance of the text directory");
        assert!(text_dir_exists);
        let dir_content =
            fs::read_dir(TEXT_DIRECTORY).expect("Should be able to read the PDF directory");
        let mut count = 0;
        for entry in dir_content {
            let _dir_entry = entry.expect("Should be able to read entry");
            count += 1;
        }
        assert_eq!(count, 2);
    }

    #[tokio::test]
    #[serial]
    async fn integration_test_all() {
        if Path::new(PDF_DIRECTORY).exists() {
            fs::remove_dir_all(PDF_DIRECTORY).expect("Should be able to remove PDF directory");
        }
        if Path::new(TEXT_DIRECTORY).exists() {
            fs::remove_dir_all(TEXT_DIRECTORY).expect("Should be able to remove text directory");
        }
        if Path::new(JSON_FILE).exists() {
            fs::remove_file(JSON_FILE).expect("Should be able to remove metadata.jsonl file");
        }
        let result = download_arxiv_papers("cs.CL".to_string(), 2, true, true, true).await;
        match result {
            Ok(_) => {}
            Err(e) => {
                eprintln!("An error occurred: {}", e.to_string());
                assert!(false)
            }
        }
        let file_exists = fs::exists(JSON_FILE)
            .expect("Should be able to check the existance of the metadata.jsonl file");
        assert!(file_exists);
        let content =
            fs::read_to_string(JSON_FILE).expect("Should be able to read metadata.jsonl file");
        assert!(content.len() > 0);
        let pdf_dir_exists: bool = fs::exists(PDF_DIRECTORY)
            .expect("Should be able to check the existance of the PDF directory");
        assert!(pdf_dir_exists);
        let pdf_dir_content =
            fs::read_dir(PDF_DIRECTORY).expect("Should be able to read the PDF directory");
        let mut pdf_count = 0;
        for entry in pdf_dir_content {
            let _dir_entry = entry.expect("Should be able to read entry");
            pdf_count += 1;
        }
        assert_eq!(pdf_count, 2);
        let text_dir_exists: bool = fs::exists(TEXT_DIRECTORY)
            .expect("Should be able to check the existance of the text directory");
        assert!(text_dir_exists);
        let dir_content =
            fs::read_dir(TEXT_DIRECTORY).expect("Should be able to read the PDF directory");
        let mut count = 0;
        for entry in dir_content {
            let _dir_entry = entry.expect("Should be able to read entry");
            count += 1;
        }
        assert_eq!(count, 2);
    }

    #[test]
    fn test_serdes_arxiv_write_summary() {
        let paper = SerDesArxiv {
            id: "".to_string(),
            updated: "".to_string(),
            published: "".to_string(),
            title: "test_title".to_string(),
            summary: "This is a test summary.".to_string(),
            authors: vec![],
            primary_category: "".to_string(),
            categories: vec![],
            pdf_url: "".to_string(),
            html_url: "".to_string(),
            comment: None,
        };
        let out_path = "test_summary.txt";
        paper
            .write_summary(out_path)
            .expect("Should write summary to file");
        let written = fs::read_to_string(out_path).expect("Should read summary file");
        assert_eq!(written, "This is a test summary.");
        fs::remove_file(out_path).expect("Should clean up summary file");
    }

    #[test]
    fn test_serdes_arxiv_to_string() {
        let paper = SerDesArxiv {
            id: "".to_string(),
            updated: "".to_string(),
            published: "".to_string(),
            title: "test_title".to_string(),
            summary: "This is a test summary.".to_string(),
            authors: vec![],
            primary_category: "".to_string(),
            categories: vec![],
            pdf_url: "".to_string(),
            html_url: "".to_string(),
            comment: None,
        };
        let json_content = serde_json::to_string(&paper).expect("Should be able to serialize");
        assert!(!json_content.contains("summary"));
    }
}
