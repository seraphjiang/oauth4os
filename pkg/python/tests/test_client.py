"""Tests for oauth4os Python SDK."""

import json
import threading
from http.server import HTTPServer, BaseHTTPRequestHandler
from unittest import TestCase

from oauth4os import Client


class MockHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path == "/oauth/token":
            self._json_response(200, {
                "access_token": "tok_py_test",
                "expires_in": 3600,
                "scope": "admin",
            })
        elif "/_search" in self.path:
            self._json_response(200, {
                "hits": {"hits": [{"_source": {"level": "error", "msg": "test"}}]}
            })
        elif self.path == "/oauth/register":
            self._json_response(201, {"client_id": "c_new", "client_secret": "s_new"})
        elif self.path == "/oauth/introspect":
            self._json_response(200, {"active": True, "scope": "admin"})
        else:
            self._json_response(200, {"result": "ok"})

    def do_GET(self):
        if self.path == "/health":
            self._json_response(200, {"status": "ok", "version": "test"})
        else:
            self._json_response(200, {})

    def do_DELETE(self):
        self.send_response(204)
        self.end_headers()

    def _json_response(self, code, data):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())

    def log_message(self, *args):
        pass  # suppress logs


class TestClient(TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = HTTPServer(("127.0.0.1", 0), MockHandler)
        cls.port = cls.server.server_address[1]
        cls.thread = threading.Thread(target=cls.server.serve_forever)
        cls.thread.daemon = True
        cls.thread.start()
        cls.url = f"http://127.0.0.1:{cls.port}"

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()

    def test_token_auto_fetch(self):
        c = Client(self.url, "test", "secret")
        tok = c.token()
        self.assertEqual(tok, "tok_py_test")
        # Cached
        self.assertEqual(c.token(), tok)

    def test_health(self):
        c = Client(self.url, "test", "secret")
        h = c.health()
        self.assertEqual(h["status"], "ok")

    def test_search(self):
        c = Client(self.url, "test", "secret")
        docs = c.search("logs-*", {"query": {"match_all": {}}})
        self.assertEqual(len(docs), 1)
        self.assertEqual(docs[0]["level"], "error")

    def test_register(self):
        c = Client(self.url, "test", "secret")
        cid, csec = c.register("agent", "read:*")
        self.assertEqual(cid, "c_new")
        self.assertEqual(csec, "s_new")

    def test_revoke(self):
        c = Client(self.url, "test", "secret")
        c.revoke_token("tok_123")  # should not raise

    def test_introspect(self):
        c = Client(self.url, "test", "secret")
        result = c.introspect("tok_123")
        self.assertTrue(result["active"])
