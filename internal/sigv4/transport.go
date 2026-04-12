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

// New creates a SigV4 transport using env vars or provided credentials.
func New(base http.RoundTripper, region, service string) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &Transport{
		Base:      base,
		Region:    region,
		Service:   service,
		AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Token:     os.Getenv("AWS_SESSION_TOKEN"),
	}
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
	r.Header.Set("host", r.URL.Host)

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
	return path
}

func canonicalQueryString(r *http.Request) string {
	return r.URL.RawQuery
}

func canonicalHeaderStr(r *http.Request) (signed, canonical string) {
	headers := make(map[string]string)
	var keys []string
	for k := range r.Header {
		lk := strings.ToLower(k)
		if lk == "host" || strings.HasPrefix(lk, "x-amz-") || lk == "content-type" {
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
