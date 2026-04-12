//! oauth4os Rust SDK — OAuth 2.0 proxy client for OpenSearch.
//!
//! ```rust
//! use oauth4os::Client;
//!
//! let client = Client::new("http://localhost:8443", "my-client", "my-secret")
//!     .scopes(vec!["read:logs-*"]);
//! let docs = client.search("logs-*", serde_json::json!({"query": {"match_all": {}}})).unwrap();
//! ```

use reqwest::blocking;
use serde::{Deserialize, Serialize};
use std::sync::Mutex;
use std::time::{Duration, Instant};

/// oauth4os client with automatic token management.
pub struct Client {
    base_url: String,
    client_id: String,
    client_secret: String,
    scopes: String,
    http: blocking::Client,
    token: Mutex<Option<CachedToken>>,
}

struct CachedToken {
    access_token: String,
    expires_at: Instant,
}

#[derive(Deserialize)]
struct TokenResponse {
    access_token: String,
    expires_in: Option<u64>,
}

#[derive(Deserialize)]
struct SearchResponse {
    hits: SearchHits,
}

#[derive(Deserialize)]
struct SearchHits {
    hits: Vec<SearchHit>,
}

#[derive(Deserialize)]
struct SearchHit {
    #[serde(rename = "_source")]
    source: serde_json::Value,
}

#[derive(Deserialize)]
struct RegisterResponse {
    client_id: String,
    client_secret: String,
}

impl Client {
    /// Create a new client.
    pub fn new(base_url: &str, client_id: &str, client_secret: &str) -> Self {
        Self {
            base_url: base_url.trim_end_matches('/').to_string(),
            client_id: client_id.to_string(),
            client_secret: client_secret.to_string(),
            scopes: "admin".to_string(),
            http: blocking::Client::builder()
                .timeout(Duration::from_secs(30))
                .build()
                .unwrap(),
            token: Mutex::new(None),
        }
    }

    /// Set requested scopes.
    pub fn scopes(mut self, scopes: Vec<&str>) -> Self {
        self.scopes = scopes.join(" ");
        self
    }

    /// Get a valid access token, fetching or refreshing as needed.
    pub fn token(&self) -> Result<String, reqwest::Error> {
        let mut cached = self.token.lock().unwrap();
        if let Some(ref t) = *cached {
            if Instant::now() < t.expires_at - Duration::from_secs(30) {
                return Ok(t.access_token.clone());
            }
        }
        let resp: TokenResponse = self
            .http
            .post(format!("{}/oauth/token", self.base_url))
            .form(&[
                ("grant_type", "client_credentials"),
                ("client_id", &self.client_id),
                ("client_secret", &self.client_secret),
                ("scope", &self.scopes),
            ])
            .send()?
            .json()?;
        let token = resp.access_token.clone();
        *cached = Some(CachedToken {
            access_token: resp.access_token,
            expires_at: Instant::now() + Duration::from_secs(resp.expires_in.unwrap_or(3600)),
        });
        Ok(token)
    }

    /// Search an OpenSearch index. Returns Vec of _source documents.
    pub fn search(
        &self,
        index: &str,
        query: serde_json::Value,
    ) -> Result<Vec<serde_json::Value>, reqwest::Error> {
        let resp: SearchResponse = self
            .http
            .post(format!("{}/{}/_search", self.base_url, index))
            .bearer_auth(self.token()?)
            .json(&query)
            .send()?
            .json()?;
        Ok(resp.hits.hits.into_iter().map(|h| h.source).collect())
    }

    /// Index a document.
    pub fn index(
        &self,
        index: &str,
        doc: &serde_json::Value,
    ) -> Result<serde_json::Value, reqwest::Error> {
        self.http
            .post(format!("{}/{}/_doc", self.base_url, index))
            .bearer_auth(self.token()?)
            .json(doc)
            .send()?
            .json()
    }

    /// Check proxy health.
    pub fn health(&self) -> Result<serde_json::Value, reqwest::Error> {
        self.http
            .get(format!("{}/health", self.base_url))
            .send()?
            .json()
    }

    /// Revoke a token by ID.
    pub fn revoke_token(&self, token_id: &str) -> Result<(), reqwest::Error> {
        self.http
            .delete(format!("{}/oauth/token/{}", self.base_url, token_id))
            .bearer_auth(self.token()?)
            .send()?;
        Ok(())
    }

    /// Dynamic client registration (RFC 7591).
    pub fn register(
        &self,
        client_name: &str,
        scope: &str,
    ) -> Result<(String, String), reqwest::Error> {
        let resp: RegisterResponse = self
            .http
            .post(format!("{}/oauth/register", self.base_url))
            .json(&serde_json::json!({
                "client_name": client_name,
                "scope": scope,
                "grant_types": ["client_credentials"]
            }))
            .send()?
            .json()?;
        Ok((resp.client_id, resp.client_secret))
    }
}
