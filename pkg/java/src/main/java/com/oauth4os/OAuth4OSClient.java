package com.oauth4os;

import java.io.IOException;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.StringJoiner;

/**
 * oauth4os Java SDK — OAuth 2.0 proxy client for OpenSearch.
 *
 * <pre>{@code
 * var client = new OAuth4OSClient("http://localhost:8443", "my-client", "my-secret", "read:logs-*");
 * String result = client.search("logs-*", "{\"query\":{\"match_all\":{}}}");
 * }</pre>
 *
 * <p>Token is auto-managed — fetched on first call, refreshed when expired.
 * Zero external dependencies (uses java.net.http from JDK 11+).
 */
public class OAuth4OSClient {

    private final String baseUrl;
    private final String clientId;
    private final String clientSecret;
    private final String scopes;
    private final HttpClient http;

    private String cachedToken;
    private Instant tokenExpiry = Instant.EPOCH;

    public OAuth4OSClient(String baseUrl, String clientId, String clientSecret, String scopes) {
        this.baseUrl = baseUrl.replaceAll("/+$", "");
        this.clientId = clientId;
        this.clientSecret = clientSecret;
        this.scopes = scopes;
        this.http = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(10))
                .build();
    }

    /** Get a valid access token, fetching or refreshing as needed. */
    public synchronized String token() throws IOException, InterruptedException {
        if (cachedToken != null && Instant.now().isBefore(tokenExpiry.minusSeconds(30))) {
            return cachedToken;
        }
        return fetchToken();
    }

    private String fetchToken() throws IOException, InterruptedException {
        String body = formEncode(Map.of(
                "grant_type", "client_credentials",
                "client_id", clientId,
                "client_secret", clientSecret,
                "scope", scopes));
        HttpRequest req = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + "/oauth/token"))
                .header("Content-Type", "application/x-www-form-urlencoded")
                .POST(HttpRequest.BodyPublishers.ofString(body))
                .build();
        HttpResponse<String> resp = http.send(req, HttpResponse.BodyHandlers.ofString());
        if (resp.statusCode() != 200) {
            throw new IOException("Token request failed: " + resp.statusCode() + " " + resp.body());
        }
        // Minimal JSON parsing without external deps
        cachedToken = extractJsonString(resp.body(), "access_token");
        long expiresIn = extractJsonLong(resp.body(), "expires_in", 3600);
        tokenExpiry = Instant.now().plusSeconds(expiresIn);
        return cachedToken;
    }

    /** Execute an authenticated request. */
    public String doRequest(String method, String path, String jsonBody)
            throws IOException, InterruptedException {
        HttpRequest.Builder builder = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + path))
                .header("Authorization", "Bearer " + token())
                .header("Content-Type", "application/json");
        if (jsonBody != null) {
            builder.method(method, HttpRequest.BodyPublishers.ofString(jsonBody));
        } else {
            builder.method(method, HttpRequest.BodyPublishers.noBody());
        }
        HttpResponse<String> resp = http.send(builder.build(), HttpResponse.BodyHandlers.ofString());
        return resp.body();
    }

    /** Search an OpenSearch index. */
    public String search(String index, String queryJson) throws IOException, InterruptedException {
        return doRequest("POST", "/" + index + "/_search", queryJson);
    }

    /** Index a document. */
    public String index(String index, String docJson) throws IOException, InterruptedException {
        return doRequest("POST", "/" + index + "/_doc", docJson);
    }

    /** Check proxy health (unauthenticated). */
    public String health() throws IOException, InterruptedException {
        HttpRequest req = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + "/health"))
                .GET().build();
        return http.send(req, HttpResponse.BodyHandlers.ofString()).body();
    }

    /** Revoke a token by ID. */
    public void revokeToken(String tokenId) throws IOException, InterruptedException {
        doRequest("DELETE", "/oauth/token/" + tokenId, null);
    }

    /** Dynamic client registration (RFC 7591). Returns JSON with client_id and client_secret. */
    public String register(String clientName, String scope) throws IOException, InterruptedException {
        String body = String.format(
                "{\"client_name\":\"%s\",\"scope\":\"%s\",\"grant_types\":[\"client_credentials\"]}",
                clientName, scope);
        HttpRequest req = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + "/oauth/register"))
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(body))
                .build();
        HttpResponse<String> resp = http.send(req, HttpResponse.BodyHandlers.ofString());
        return resp.body();
    }

    // --- helpers (no external JSON lib) ---

    private static String formEncode(Map<String, String> params) {
        StringJoiner sj = new StringJoiner("&");
        params.forEach((k, v) -> sj.add(
                URLEncoder.encode(k, StandardCharsets.UTF_8) + "=" +
                URLEncoder.encode(v, StandardCharsets.UTF_8)));
        return sj.toString();
    }

    private static String extractJsonString(String json, String key) {
        String search = "\"" + key + "\":\"";
        int start = json.indexOf(search);
        if (start < 0) return "";
        start += search.length();
        int end = json.indexOf("\"", start);
        return json.substring(start, end);
    }

    private static long extractJsonLong(String json, String key, long defaultVal) {
        String search = "\"" + key + "\":";
        int start = json.indexOf(search);
        if (start < 0) return defaultVal;
        start += search.length();
        StringBuilder sb = new StringBuilder();
        for (int i = start; i < json.length() && Character.isDigit(json.charAt(i)); i++) {
            sb.append(json.charAt(i));
        }
        return sb.length() > 0 ? Long.parseLong(sb.toString()) : defaultVal;
    }
}
