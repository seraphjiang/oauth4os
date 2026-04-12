// Package sigv4 provides an http.RoundTripper that signs requests with AWS SigV4.
// Used for AOSS (OpenSearch Serverless) and managed OpenSearch with IAM auth.
package sigv4

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// Transport wraps an http.RoundTripper and adds SigV4 signatures.
type Transport struct {
	Base      http.RoundTripper
	Region    string
	Service   string // "aoss" or "es"
	AccessKey string
	SecretKey string
	Token     string // optional session token
}

// New creates a SigV4 transport using env vars, ECS/AppRunner container creds, or provided credentials.
func New(base http.RoundTripper, region, service string) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	t := &Transport{
		Base:      base,
		Region:    region,
		Service:   service,
		AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Token:     os.Getenv("AWS_SESSION_TOKEN"),
	}
	// If no env vars, try ECS/AppRunner container credentials
	if t.AccessKey == "" {
		t.refreshContainerCreds()
	}
	return t
}

func (t *Transport) refreshContainerCreds() {
	uri := os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
	if uri == "" {
		return
	}
	resp, err := http.Get("http://169.254.170.2" + uri)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// Parse JSON manually (no deps)
	t.AccessKey = jsonVal(string(body), "AccessKeyId")
	t.SecretKey = jsonVal(string(body), "SecretAccessKey")
	t.Token = jsonVal(string(body), "Token")
}

func jsonVal(body, key string) string {
	k := `"` + key + `"`
	i := strings.Index(body, k)
	if i < 0 {
		return ""
	}
	rest := body[i+len(k):]
	// skip :" or " : "
	i = strings.Index(rest, `"`)
	if i < 0 {
		return ""
	}
	rest = rest[i+1:]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid mutating the original
	r := req.Clone(req.Context())

	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	r.Header.Set("x-amz-date", amzdate)
	if t.Token != "" {
		r.Header.Set("x-amz-security-token", t.Token)
	}

	// Read and hash body
	var bodyHash string
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		bodyHash = hashSHA256(body)
	} else {
		bodyHash = hashSHA256(nil)
	}
	r.Header.Set("x-amz-content-sha256", bodyHash)

	// Canonical request
	signedHeaders, canonicalHeaders := canonicalHeaderStr(r)
	canonicalReq := strings.Join([]string{
		r.Method,
		canonicalURI(r),
		canonicalQueryString(r),
		canonicalHeaders,
		signedHeaders,
		bodyHash,
	}, "\n")

	// String to sign
	scope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, t.Region, t.Service)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzdate, scope, hashSHA256([]byte(canonicalReq)))

	// Signing key
	signingKey := deriveKey(t.SecretKey, datestamp, t.Region, t.Service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Authorization header
	r.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		t.AccessKey, scope, signedHeaders, signature))

	return t.Base.RoundTrip(r)
}

func canonicalURI(r *http.Request) string {
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	// URI-encode each path segment per RFC 3986, then rejoin
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = uriEncode(s, false)
	}
	return strings.Join(segments, "/")
}

func canonicalQueryString(r *http.Request) string {
	params := r.URL.Query()
	if len(params) == 0 {
		return ""
	}
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vs := params[k]
		sort.Strings(vs)
		for _, v := range vs {
			parts = append(parts, uriEncode(k, true)+"="+uriEncode(v, true))
		}
	}
	return strings.Join(parts, "&")
}

func canonicalHeaderStr(r *http.Request) (signed, canonical string) {
	headers := make(map[string]string)
	var keys []string

	// Host is special in Go — it's in r.Host, not r.Header
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	headers["host"] = host
	keys = append(keys, "host")

	for k := range r.Header {
		lk := strings.ToLower(k)
		if lk == "host" {
			continue // already handled above
		}
		if strings.HasPrefix(lk, "x-amz-") || lk == "content-type" {
			headers[lk] = strings.TrimSpace(r.Header.Get(k))
			keys = append(keys, lk)
		}
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+":"+headers[k]+"\n")
	}
	return strings.Join(keys, ";"), strings.Join(parts, "")
}

// uriEncode encodes a string per RFC 3986. If encodeSlash is false, '/' is preserved.
func uriEncode(s string, encodeSlash bool) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '~' || c == '.' {
			b.WriteByte(c)
		} else if c == '/' && !encodeSlash {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func hashSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func deriveKey(secret, datestamp, region, service string) []byte {
	k := hmacSHA256([]byte("AWS4"+secret), []byte(datestamp))
	k = hmacSHA256(k, []byte(region))
	k = hmacSHA256(k, []byte(service))
	k = hmacSHA256(k, []byte("aws4_request"))
	return k
}
