"""LangChain agent with oauth4os-scoped OpenSearch access."""
import os
import json
import requests


class OAuth4OSSearchTool:
    """Search OpenSearch via oauth4os proxy with scoped token."""

    def __init__(self, proxy_url=None, token=None):
        self.proxy_url = (proxy_url or os.environ.get("OAUTH4OS_URL", "http://localhost:8443")).rstrip("/")
        self.token = token or os.environ.get("OAUTH4OS_TOKEN", "")
        self.session = requests.Session()
        self.session.headers["Authorization"] = f"Bearer {self.token}"
        self.session.headers["Content-Type"] = "application/json"

    def search(self, index, query, size=10):
        """Search an index with a query string."""
        resp = self.session.post(
            f"{self.proxy_url}/{index}/_search",
            json={"query": {"query_string": {"query": query}}, "size": size},
        )
        resp.raise_for_status()
        hits = resp.json().get("hits", {}).get("hits", [])
        return [h["_source"] for h in hits]

    def count(self, index, query="*"):
        """Count matching documents."""
        resp = self.session.post(
            f"{self.proxy_url}/{index}/_count",
            json={"query": {"query_string": {"query": query}}},
        )
        resp.raise_for_status()
        return resp.json().get("count", 0)

    def get_mappings(self, index):
        """Get index field mappings."""
        resp = self.session.get(f"{self.proxy_url}/{index}/_mapping")
        resp.raise_for_status()
        return resp.json()


def main():
    import sys
    query = " ".join(sys.argv[1:]) or "level:ERROR"
    tool = OAuth4OSSearchTool()

    print(f"Searching logs-* for: {query}")
    results = tool.search("logs-*", query, size=5)

    if not results:
        print("No results found.")
        return

    for i, doc in enumerate(results, 1):
        ts = doc.get("timestamp", doc.get("@timestamp", ""))
        level = doc.get("level", "")
        msg = doc.get("message", "")[:120]
        svc = doc.get("service", "")
        print(f"  [{i}] {ts} {level} {svc}: {msg}")

    total = tool.count("logs-*", query)
    print(f"\nTotal matches: {total}")


if __name__ == "__main__":
    main()
