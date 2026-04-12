"""oauth4os — Python SDK for OAuth 2.0 proxy for OpenSearch.

Usage:
    from oauth4os import Client

    c = Client("http://localhost:8443", "my-client", "my-secret", scopes=["read:logs-*"])
    docs = c.search("logs-*", {"query": {"match": {"level": "error"}}})
"""

from oauth4os.client import Client

__all__ = ["Client"]
__version__ = "0.1.0"
