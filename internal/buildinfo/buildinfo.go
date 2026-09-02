// package buildinfo stores compile-time set variables for binaries built in this repository.
package buildinfo

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// These variables are normally set by ldflags at build time.

var CommitSHA string
var Time string
var Version string
var Asset string
var StartedAt time.Time

const versionLocal = "local"

func init() {
	if Version == "" {
		Version = versionLocal
	}
	if CommitSHA == "" {
		sha, err := getCurrentGitHash()
		if err != nil {
			sha = "unknown"
		}
		CommitSHA = sha
	}
	if Time == "" {
		Time = time.Now().UTC().Format("20060102-150405")
	}
	StartedAt = time.Now().UTC()
}

func FullVersion() string {
	if Version != versionLocal {
		return Asset + " " + Version
	}
	return Asset + " " + Version + "-" + Time + "-" + CommitSHA
}

func getCurrentGitHash() (string, error) {
	revParse, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("failed to git rev-parse --short: %w", err)
	}
	return strings.TrimSpace(string(revParse)), nil
}
