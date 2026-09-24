package internal

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sandgardenhq/affiro/client"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/cliupdate"
	"github.com/sandgardenhq/affiro/cmd/affiro/internal/keylogx"
	"github.com/sandgardenhq/affiro/internal/buildinfo"
)

var fullVersion = "affiro " + buildinfo.Version

const helpText = `affiro CLI

usage: affiro [-gui]`

// apiBaseURL returns the affiro API host to talk to: AFFIRO_API_BASE_URL when set (e.g. to
// point at a local API server), otherwise the app.affiro.com host the client package also
// defaults to. It is what a caller passes to client.WithHost.
func apiBaseURL() string {
	if baseURL := os.Getenv("AFFIRO_API_BASE_URL"); baseURL != "" {
		return baseURL
	}
	return client.DefaultHost
}

// checkForUpdate reports whether a newer affiro build is published and, if so, applies it to
// the currently running executable in place. Any failure (network, API, or apply) prints a
// message and returns rather than crashing, so -version stays usable when the affiro API is
// unreachable.
func checkForUpdate(baseURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := cliupdate.Check(ctx, http.DefaultClient, baseURL, buildinfo.Version)
	if err != nil {
		fmt.Println("could not check for updates:", err)
		return
	}
	if !result.UpdateAvailable {
		fmt.Println("up to date")
		return
	}
	fmt.Printf("a newer build is available: %s, updating...\n", result.LatestVersion)
	if err := cliupdate.Apply(ctx, http.DefaultClient, baseURL, result.DownloadPath, ""); err != nil {
		fmt.Println("could not apply the update:", err)
		return
	}
	fmt.Printf("updated to %s\n", result.LatestVersion)
}

type CLIConfig struct {
	StorageDir          string
	APIURL              string
	BackgroundWorker    bool
	GUIMode             bool
	ShowUnfinishedPages bool
}

// defaultStorageDir is ~/.affiro, created if it is not there yet.
func defaultStorageDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating the home directory: %w", err)
	}
	dir := filepath.Join(homeDir, ".affiro")
	if err := os.MkdirAll(dir, 0777); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return dir, nil
}

var errBadBackgroundWorkerOS = errors.New("background workers not supported on this operating system")

// ParseFlags reads the command line. -help and -version are handled here and exit the process.
func ParseFlags() (CLIConfig, error) {
	// This is a strange condition; all CLI programs generally have their 0th argument as the
	// program that they are. This is likely dead code.
	if len(os.Args) == 0 {
		fmt.Println(helpText)
		os.Exit(255)
	}
	flagSet := flag.NewFlagSet("monitor", flag.ContinueOnError)
	guiMode := flagSet.Bool("gui", false, "run in gui mode")
	backgroundWorker := flagSet.Bool("background-worker", false, "run a background worker (windows only)")
	storageDir := flagSet.String("dir", "", "store calculated signatures in this directory (default ~/.affiro)")
	showVersion := flagSet.Bool("version", false, "print version and exit")
	showHelp := flagSet.Bool("help", false, "print help text and exit")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return CLIConfig{}, fmt.Errorf("parsing the command line: %w", err)
	}
	if *backgroundWorker && !keylogx.BackgroundWorkerAllowed {
		return CLIConfig{}, errBadBackgroundWorkerOS
	}
	if *storageDir == "" {
		dir, err := defaultStorageDir()
		if err != nil {
			return CLIConfig{}, err
		}
		*storageDir = dir
	}
	baseURL := apiBaseURL()
	switch {
	case *showHelp:
		fmt.Println(helpText)
		os.Exit(255)
	case *showVersion:
		fmt.Println(fullVersion)
		checkForUpdate(baseURL)
		os.Exit(254)
	}
	return CLIConfig{
		StorageDir:          *storageDir,
		GUIMode:             *guiMode,
		BackgroundWorker:    *backgroundWorker,
		APIURL:              baseURL,
		ShowUnfinishedPages: os.Getenv("SHOW_UNFINISHED_PAGES") == "true",
	}, nil
}
