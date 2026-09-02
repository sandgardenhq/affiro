// Package cliupdate checks the playground API for newer affiro CLI builds, downloads them, and
// can apply a downloaded build to the currently running executable.
package cliupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fynelabs/selfupdate"
	"github.com/sandgardenhq/affiro/internal/releaseassets"
)

// CheckResult mirrors the playground API's CLIVersionCheckResponse shape.
type CheckResult struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	DownloadPath    string `json:"downloadPath"`
}

// Check asks the playground API at baseURL whether currentVersion is out of date for this
// binary's GOOS/GOARCH.
func Check(ctx context.Context, client *http.Client, baseURL, currentVersion string) (CheckResult, error) {
	u, err := url.Parse(baseURL + "/api/v1/cli/version-check")
	if err != nil {
		return CheckResult{}, fmt.Errorf("invalid base URL: %w", err)
	}
	q := u.Query()
	q.Set("version", currentVersion)
	q.Set("os", runtime.GOOS)
	q.Set("arch", runtime.GOARCH)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return CheckResult{}, fmt.Errorf("failed to build version-check request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{}, fmt.Errorf("failed to reach playground API: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to close version-check response body: %v\n", closeErr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return CheckResult{}, fmt.Errorf("version-check request failed with status %d", resp.StatusCode)
	}
	var result CheckResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return CheckResult{}, fmt.Errorf("failed to decode version-check response: %w", err)
	}
	return result, nil
}

// Download fetches the build at baseURL+downloadPath (as returned by Check in
// CheckResult.DownloadPath) and saves it under destDir, named for this binary's own GOOS/GOARCH.
// It returns the path the build was saved to; it never touches the running binary.
func Download(ctx context.Context, client *http.Client, baseURL, downloadPath, destDir string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+downloadPath, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build download request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach playground API: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to close download response body: %v\n", closeErr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download request failed with status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}
	savedPath := filepath.Join(destDir, releaseassets.BuildName(releaseassets.CLIFileName, runtime.GOOS, runtime.GOARCH))
	out, err := os.Create(savedPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("failed to save downloaded build: %w", err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("failed to close downloaded build file: %w", err)
	}
	// S3 objects carry no permission bits, so the download comes back as a plain file;
	// mark it executable ourselves or it's unusable in place of the running binary.
	if err := os.Chmod(savedPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to mark downloaded build executable: %w", err)
	}
	return savedPath, nil
}

// Apply fetches the build at baseURL+downloadPath and atomically replaces targetPath with
// it (temp file + rename, via selfupdate). An empty targetPath updates the running executable.
// Unlike Download, it never writes to storageDir/updates.
func Apply(ctx context.Context, client *http.Client, baseURL, downloadPath, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+downloadPath, nil)
	if err != nil {
		return fmt.Errorf("failed to build download request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach playground API: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to close download response body: %v\n", closeErr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download request failed with status %d", resp.StatusCode)
	}

	opts := selfupdate.Options{TargetPath: targetPath}
	// selfupdate defaults TargetMode to 0755 rather than preserving the existing file's mode,
	// which would silently reset a more restrictive mode on every update; carry it forward.
	if resolvedPath, statErr := resolveTargetPath(targetPath); statErr == nil {
		if info, statErr := os.Stat(resolvedPath); statErr == nil {
			opts.TargetMode = info.Mode().Perm()
		}
	}

	if err := selfupdate.Apply(resp.Body, opts); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return fmt.Errorf("failed to roll back after a failed update (system left in an inconsistent state): %w (update error: %v)", rerr, err)
		}
		return fmt.Errorf("failed to apply update: %w", err)
	}
	return nil
}

// resolveTargetPath mirrors selfupdate's own empty-TargetPath handling: an empty path means
// the currently running executable.
func resolveTargetPath(targetPath string) (string, error) {
	if targetPath != "" {
		return targetPath, nil
	}
	return selfupdate.ExecutableRealPath()
}
