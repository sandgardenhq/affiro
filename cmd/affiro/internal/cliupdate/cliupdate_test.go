package cliupdate_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sandgardenhq/affiro/cmd/affiro/internal/cliupdate"
)

func TestCheck_UpdateAvailable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/cli/version-check" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("version") != "v1.0.0" {
			t.Fatalf("unexpected version query param: %s", q.Get("version"))
		}
		w.Header().Set("Content-Type", "application/json")
		//nolint:errcheck
		w.Write([]byte(`{"currentVersion":"v1.0.0","latestVersion":"v1.1.0","updateAvailable":true,"downloadPath":"/api/v1/cli/download?os=linux&arch=amd64&version=v1.1.0"}`))
	}))
	defer srv.Close()

	result, err := cliupdate.Check(context.Background(), srv.Client(), srv.URL, "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatal("expected UpdateAvailable to be true")
	}
	if result.LatestVersion != "v1.1.0" {
		t.Fatalf("unexpected LatestVersion: %s", result.LatestVersion)
	}
	if result.DownloadPath != "/api/v1/cli/download?os=linux&arch=amd64&version=v1.1.0" {
		t.Fatalf("unexpected DownloadPath: %s", result.DownloadPath)
	}
}

func TestCheck_NoUpdateAvailable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		//nolint:errcheck
		w.Write([]byte(`{"currentVersion":"v1.1.0","latestVersion":"v1.1.0","updateAvailable":false}`))
	}))
	defer srv.Close()

	result, err := cliupdate.Check(context.Background(), srv.Client(), srv.URL, "v1.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UpdateAvailable {
		t.Fatal("expected UpdateAvailable to be false")
	}
}

func TestCheck_NonOKStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := cliupdate.Check(context.Background(), srv.Client(), srv.URL, "v1.0.0")
	if err == nil {
		t.Fatal("expected an error for non-200 response")
	}
}

func TestCheck_Unreachable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // closed immediately so the address is unreachable

	_, err := cliupdate.Check(context.Background(), srv.Client(), srv.URL, "v1.0.0")
	if err == nil {
		t.Fatal("expected an error when the server is unreachable")
	}
}

func TestDownload_SavesResponseBodyToFile(t *testing.T) {
	t.Parallel()

	const body = "fake binary contents"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/cli/download" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		//nolint:errcheck
		w.Write([]byte(body))
	}))
	defer srv.Close()

	destDir := t.TempDir()
	savedPath, err := cliupdate.Download(context.Background(), srv.Client(), srv.URL, "/api/v1/cli/download?os=linux&arch=amd64&version=v1.1.0", destDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Dir(savedPath) != destDir {
		t.Fatalf("expected saved path to be inside %s, got %s", destDir, savedPath)
	}
	got, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(got) != body {
		t.Fatalf("unexpected file contents: %s", got)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(savedPath)
		if err != nil {
			t.Fatalf("failed to stat saved file: %v", err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Fatalf("expected saved file to be executable, got mode %v", info.Mode())
		}
	}
}

func TestDownload_NonOKStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := cliupdate.Download(context.Background(), srv.Client(), srv.URL, "/api/v1/cli/download", t.TempDir())
	if err == nil {
		t.Fatal("expected an error for non-200 response")
	}
}

func TestApply_ReplacesTargetInPlace(t *testing.T) {
	t.Parallel()

	const oldBody = "old binary contents"
	const newBody = "new binary contents, a bit longer than the old one"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/cli/download" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		//nolint:errcheck
		w.Write([]byte(newBody))
	}))
	defer srv.Close()

	targetPath := filepath.Join(t.TempDir(), "affiro")
	// A non-default mode: proves Apply preserves the existing file's permissions rather than
	// always resetting to selfupdate's built-in 0755 default.
	const seedMode = 0o700
	if err := os.WriteFile(targetPath, []byte(oldBody), seedMode); err != nil {
		t.Fatalf("failed to seed target file: %v", err)
	}

	err := cliupdate.Apply(context.Background(), srv.Client(), srv.URL, "/api/v1/cli/download?os=linux&arch=amd64&version=v1.1.0", targetPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read target file: %v", err)
	}
	if string(got) != newBody {
		t.Fatalf("unexpected target file contents: %s", got)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(targetPath)
		if err != nil {
			t.Fatalf("failed to stat target file: %v", err)
		}
		if info.Mode().Perm() != seedMode {
			t.Fatalf("expected target file to keep mode %v, got %v", os.FileMode(seedMode), info.Mode().Perm())
		}
	}
}

func TestApply_NonOKStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	targetPath := filepath.Join(t.TempDir(), "affiro")
	if err := os.WriteFile(targetPath, []byte("old binary contents"), 0o755); err != nil {
		t.Fatalf("failed to seed target file: %v", err)
	}

	err := cliupdate.Apply(context.Background(), srv.Client(), srv.URL, "/api/v1/cli/download", targetPath)
	if err == nil {
		t.Fatal("expected an error for non-200 response")
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read target file: %v", err)
	}
	if string(got) != "old binary contents" {
		t.Fatalf("expected target file to be left untouched, got: %s", got)
	}
}

func TestApply_Unreachable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // closed immediately so the address is unreachable

	targetPath := filepath.Join(t.TempDir(), "affiro")
	if err := os.WriteFile(targetPath, []byte("old binary contents"), 0o755); err != nil {
		t.Fatalf("failed to seed target file: %v", err)
	}

	err := cliupdate.Apply(context.Background(), srv.Client(), srv.URL, "/api/v1/cli/download", targetPath)
	if err == nil {
		t.Fatal("expected an error when the server is unreachable")
	}
}

func TestApply_MidApplyFailureLeavesTargetIntact(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("directory write-permission semantics differ on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses the permission check this test relies on")
	}

	const newBody = "new binary contents"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck
		w.Write([]byte(newBody))
	}))
	defer srv.Close()

	dir := t.TempDir()
	targetPath := filepath.Join(dir, "affiro")
	if err := os.WriteFile(targetPath, []byte("old binary contents"), 0o755); err != nil {
		t.Fatalf("failed to seed target file: %v", err)
	}
	// selfupdate writes its replacement into a temp file in this directory before renaming it
	// over targetPath; without write permission on the directory, that step fails, forcing the
	// mid-apply (post-download, pre-swap) failure this test is checking survives cleanly.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("failed to make target directory read-only: %v", err)
	}
	t.Cleanup(func() {
		//nolint:errcheck
		os.Chmod(dir, 0o755)
	})

	err := cliupdate.Apply(context.Background(), srv.Client(), srv.URL, "/api/v1/cli/download", targetPath)
	if err == nil {
		t.Fatal("expected an error when the target directory isn't writable")
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read target file: %v", err)
	}
	if string(got) != "old binary contents" {
		t.Fatalf("expected target file to be left untouched after a failed apply, got: %s", got)
	}
}
