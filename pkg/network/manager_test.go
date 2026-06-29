package network

import (
	"strings"
	"testing"
)

// TestDrainPullStreamSuccess verifies a normal pull progress stream (status
// lines, no error field) is consumed without reporting an error.
func TestDrainPullStreamSuccess(t *testing.T) {
	body := `{"status":"Pulling from lib/img"}
{"status":"Pulling fs layer","id":"abc"}
{"status":"Download complete","id":"abc"}
{"status":"Status: Downloaded newer image"}
`
	if err := drainPullStream(strings.NewReader(body)); err != nil {
		t.Fatalf("expected nil error for clean pull stream, got %v", err)
	}
}

// TestDrainPullStreamReportsInStreamError proves the platform-fallback fix:
// the daemon reports a platform mismatch as an in-stream JSON error rather than
// as a failure of ImagePull itself, so drainPullStream must surface it for
// pullImage to fall back to the next platform candidate (e.g. amd64).
func TestDrainPullStreamReportsInStreamError(t *testing.T) {
	body := `{"status":"Pulling from lib/img"}
{"errorDetail":{"message":"no matching manifest for linux/arm64 in the manifest list entries"},"error":"no matching manifest for linux/arm64 in the manifest list entries"}
`
	err := drainPullStream(strings.NewReader(body))
	if err == nil {
		t.Fatal("expected an error for an in-stream pull failure, got nil")
	}
	if !strings.Contains(err.Error(), "no matching manifest") {
		t.Errorf("error should surface the daemon message, got: %v", err)
	}
}

// TestDrainPullStreamEmpty ensures an empty stream is treated as success.
func TestDrainPullStreamEmpty(t *testing.T) {
	if err := drainPullStream(strings.NewReader("")); err != nil {
		t.Fatalf("empty stream should not error, got %v", err)
	}
}

// TestDockerPlatform sanity-checks the host-platform mapping the pull/create
// path relies on: arm64 hosts map to linux/arm64, everything else to amd64.
func TestDockerPlatform(t *testing.T) {
	osName, arch := dockerPlatform()
	if osName != "linux" {
		t.Errorf("dockerPlatform OS: got %q want %q", osName, "linux")
	}
	if arch != "arm64" && arch != "amd64" {
		t.Errorf("dockerPlatform arch: got %q want arm64 or amd64", arch)
	}
}
