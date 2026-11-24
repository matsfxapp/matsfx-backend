package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matsfxapp/matsfx-backend/internal/config"
)

type seenRequest struct {
	method string
	path   string
	auth   string
}

func fakeS3(t *testing.T, status int) (*httptest.Server, *seenRequest) {
	t.Helper()
	var got seenRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = seenRequest{method: r.Method, path: r.URL.Path, auth: r.Header.Get("Authorization")}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func testConfig(endpoint string) config.S3 {
	return config.S3{
		Endpoint:  endpoint,
		Region:    "garage",
		Bucket:    "matsfx",
		AccessKey: "GKtest",
		SecretKey: "secret",
	}
}

func TestOpenCheckBucketPathStyle(t *testing.T) {
	srv, got := fakeS3(t, http.StatusOK)

	if _, err := Open(context.Background(), testConfig(srv.URL)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodHead || got.path != "/matsfx" {
		t.Fatalf("got %s %s, want HEAD /matsfx", got.method, got.path)
	}
	if !strings.Contains(got.auth, "Credential=GKtest/") {
		t.Fatalf("request not signed with the access key: %q", got.auth)
	}
}

func TestOpenMissingBucket(t *testing.T) {
	srv, _ := fakeS3(t, http.StatusNotFound)

	_, err := Open(context.Background(), testConfig(srv.URL))
	if err == nil || !strings.HasPrefix(err.Error(), "check bucket matsfx:") {
		t.Fatalf("expected bucket error, got %v", err)
	}
}
