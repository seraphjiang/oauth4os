"""oauth4os Python client with automatic token management."""

import time
import threading
from urllib.parse import urljoin

try:
    import requests
except ImportError:
    raise ImportError("Install requests: pip install requests")


class Client:
    """OAuth4OS client — auto-refreshing tokens, OpenSearch queries via proxy.

    Args:
        base_url: Proxy URL (e.g. "http://localhost:8443")
        client_id: OAuth client ID
        client_secret: OAuth client secret
        scopes: List of scopes (default: ["admin"])
        timeout: HTTP timeout in seconds (default: 30)
    """

    def __init__(self, base_url, client_id, client_secret, scopes=None, timeout=30):
        self.base_url = base_url.rstrip("/")
        self.client_id = client_id
        self.client_secret = client_secret
        self.scopes = " ".join(scopes or ["admin"])
        self.timeout = timeout
        self._session = requests.Session()
        self._token = None
        self._expiry = 0
        self._lock = threading.Lock()

    def token(self):
        """Get a valid access token, fetching or refreshing as needed."""
        with self._lock:
            if self._token and time.time() < self._expiry - 30:
                return self._token
            return self._fetch_token()

    def _fetch_token(self):
        resp = self._session.post(
            f"{self.base_url}/oauth/token",
            data={
                "grant_type": "client_credentials",
                "client_id": self.client_id,
                "client_secret": self.client_secret,
                "scope": self.scopes,
            },
            timeout=self.timeout,
        )
        resp.raise_for_status()
        data = resp.json()
        self._token = data["access_token"]
        self._expiry = time.time() + data.get("expires_in", 3600)
        return self._token

    def _auth_headers(self):
        return {"Authorization": f"Bearer {self.token()}"}

    def do(self, method, path, json=None):
        """Execute an authenticated request against the proxy."""
        resp = self._session.request(
            method,
            f"{self.base_url}{path}",
            json=json,
            headers=self._auth_headers(),
            timeout=self.timeout,
        )
        resp.raise_for_status()
        return resp

    def search(self, index, query):
        """Query an OpenSearch index. Returns list of _source docs."""
        resp = self.do("POST", f"/{index}/_search", json=query)
        hits = resp.json().get("hits", {}).get("hits", [])
        return [h["_source"] for h in hits]

    def index(self, index, doc, doc_id=None):
        """Index a document. Returns the response body."""
        path = f"/{index}/_doc"
        if doc_id:
            path += f"/{doc_id}"
        return self.do("POST", path, json=doc).json()

    def health(self):
        """Check proxy health (unauthenticated)."""
        return self._session.get(
            f"{self.base_url}/health", timeout=self.timeout
        ).json()

    def create_token(self, scope):
        """Issue a new scoped token."""
        resp = self._session.post(
            f"{self.base_url}/oauth/token",
            data={"grant_type": "client_credentials", "client_id": self.client_id, "scope": scope},
            timeout=self.timeout,
        )
        resp.raise_for_status()
        return resp.json()["access_token"]

    def revoke_token(self, token_id):
        """Revoke a token by ID."""
        resp = self._session.delete(
            f"{self.base_url}/oauth/token/{token_id}",
            headers=self._auth_headers(),
            timeout=self.timeout,
        )
        resp.raise_for_status()

    def register(self, client_name, scope=""):
        """Dynamic client registration (RFC 7591). Returns (client_id, client_secret)."""
        resp = self._session.post(
            f"{self.base_url}/oauth/register",
            json={"client_name": client_name, "scope": scope, "grant_types": ["client_credentials"]},
            timeout=self.timeout,
        )
        resp.raise_for_status()
        data = resp.json()
        return data["client_id"], data["client_secret"]

    def introspect(self, token_value):
        """Introspect a token (RFC 7662)."""
        resp = self._session.post(
            f"{self.base_url}/oauth/introspect",
            data={"token": token_value},
            headers=self._auth_headers(),
            timeout=self.timeout,
        )
        resp.raise_for_status()
        return resp.json()
