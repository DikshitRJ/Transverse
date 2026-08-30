// Package objectstore provides a minimal S3-API-compatible client sufficient for the
// evidence upload pipeline: presigned PUT URLs for direct browser uploads, and
// authenticated GET/DELETE for server-side download and cleanup. It implements AWS
// Signature Version 4 by hand against the standard library only, so it works against
// MinIO (or any S3-compatible store) without pulling in the official MinIO/AWS SDKs.
package objectstore

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"transverse/internal/config"
)

// sigV4Region is a fixed placeholder region. MinIO does not enforce AWS regions the way
// S3 does, but the SigV4 algorithm requires some region string to be present on both
// sides of the signature, so client and server just need to agree on a value.
const sigV4Region = "us-east-1"

// MinIOClient is a small S3-API-compatible client backed by presigned, path-style URLs.
type MinIOClient struct {
	endpoint   string // host[:port], no scheme
	useSSL     bool
	accessKey  string
	secretKey  string
	bucket     string
	httpClient *http.Client
}

// NewMinIOClient constructs a client from config. It does not verify connectivity or
// bucket existence at construction time — call Ping and EnsureBucket for that.
func NewMinIOClient(cfg *config.Config) *MinIOClient {
	return &MinIOClient{
		endpoint:   cfg.MinIOEndpoint,
		useSSL:     cfg.MinIOUseSSL,
		accessKey:  cfg.MinIORootUser,
		secretKey:  cfg.MinIORootPassword,
		bucket:     cfg.MinIOBucket,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *MinIOClient) scheme() string {
	if c.useSSL {
		return "https"
	}
	return "http"
}

// Ping performs a lightweight, unauthenticated liveness check against MinIO's health
// endpoint. It returns an error if the server cannot be reached at all; callers should
// treat that as "degrade gracefully", not as fatal to process startup.
func (c *MinIOClient) Ping(ctx context.Context) error {
	u := fmt.Sprintf("%s://%s/minio/health/live", c.scheme(), c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// EnsureBucket creates the configured bucket if it does not already exist. MinIO returns
// a variety of non-5xx statuses for an already-existing/owned bucket, so any non-5xx
// response is treated as "fine, bucket is usable" rather than probed further.
func (c *MinIOClient) EnsureBucket(ctx context.Context) error {
	signedURL, err := c.presignedURL(http.MethodPut, "", 15*time.Minute)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, signedURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("minio ensure-bucket failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	return nil
}

// PresignPut returns a presigned URL the caller can PUT object bytes directly to,
// satisfying evidence.ObjectStore.
func (c *MinIOClient) PresignPut(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
	return c.presignedURL(http.MethodPut, objectKey, expires)
}

// Get downloads an object's full contents, satisfying evidence.ObjectStore.
func (c *MinIOClient) Get(ctx context.Context, objectKey string) ([]byte, error) {
	signedURL, err := c.presignedURL(http.MethodGet, objectKey, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, signedURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("minio get failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

// Delete removes an object, satisfying evidence.ObjectStore. A missing object is not
// treated as an error since the caller's intent ("this key should not exist") is met.
func (c *MinIOClient) Delete(ctx context.Context, objectKey string) error {
	signedURL, err := c.presignedURL(http.MethodDelete, objectKey, 5*time.Minute)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, signedURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("minio delete failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	return nil
}

// presignedURL builds an AWS Signature Version 4 presigned URL (query-string signing,
// SignedHeaders=host, UNSIGNED-PAYLOAD) for the given method and object key, valid for
// `expires`. objectKey == "" signs a request against the bucket root (used for bucket
// creation). See:
// https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-query-string-auth.html
func (c *MinIOClient) presignedURL(method, objectKey string, expires time.Duration) (string, error) {
	if c.bucket == "" {
		return "", fmt.Errorf("objectstore: bucket not configured")
	}

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, sigV4Region)

	canonicalURI := "/" + c.bucket
	if objectKey != "" {
		canonicalURI += "/" + uriEncodePath(objectKey)
	}

	query := url.Values{}
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", c.accessKey+"/"+credentialScope)
	query.Set("X-Amz-Date", amzDate)
	query.Set("X-Amz-Expires", strconv.Itoa(int(expires.Seconds())))
	query.Set("X-Amz-SignedHeaders", "host")
	canonicalQueryString := query.Encode()

	host := c.endpoint
	canonicalHeaders := "host:" + host + "\n"
	signedHeaders := "host"

	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		"UNSIGNED-PAYLOAD",
	}, "\n")

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(c.secretKey, dateStamp, sigV4Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	return fmt.Sprintf("%s://%s%s?%s&X-Amz-Signature=%s", c.scheme(), host, canonicalURI, canonicalQueryString, signature), nil
}

func deriveSigningKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// uriEncodePath percent-encodes a path per RFC 3986, preserving "/" separators, as
// required by AWS SigV4 canonical URI construction. net/url's QueryEscape is not
// suitable here: it escapes spaces as "+" and would also escape "/" within a segment
// boundary incorrectly if applied to the whole path at once.
func uriEncodePath(p string) string {
	segments := strings.Split(p, "/")
	for i, seg := range segments {
		segments[i] = uriEncodeSegment(seg)
	}
	return strings.Join(segments, "/")
}

func uriEncodeSegment(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
