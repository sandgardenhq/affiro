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

usage: 
	affiro [-gui]
	affiro version
	affiro update`

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

func handleEarlyExitCommands(args []string, fgs flags) {
	if fgs.showHelp {
		fmt.Println(helpText)
		os.Exit(255)
	}
	printVersion := false
	update := false
	commands := 0
	for _, a := range args {
		switch a {
		case "version":
			printVersion = true
			commands++
			continue
		case "update":
			update = true
			commands++
			continue
		}
		if len(a) > 0 && a[0] != '-' {
			fmt.Println(helpText)
			os.Exit(255)
		}
	}
	if commands > 1 {
		fmt.Println(helpText)
		os.Exit(255)
	}
	if update {
		checkForUpdate(fgs.baseURL)
		os.Exit(253)
	}
	if printVersion {
		fmt.Println(fullVersion)
		os.Exit(254)
	}
}

type flags struct {
	storageDir       string
	baseURL          string
	showHelp         bool
	backgroundWorker bool
	guiMode          bool
}

func handleCommand(args []string, fgs flags) (CLIConfig, error) {
	handleEarlyExitCommands(args, fgs)

	if fgs.backgroundWorker && !keylogx.BackgroundWorkerAllowed {
		return CLIConfig{}, errBadBackgroundWorkerOS
	}
	if fgs.storageDir == "" {
		dir, err := defaultStorageDir()
		if err != nil {
			return CLIConfig{}, err
		}
		fgs.storageDir = dir
	}
	return CLIConfig{
		StorageDir:          fgs.storageDir,
		GUIMode:             fgs.guiMode,
		BackgroundWorker:    fgs.backgroundWorker,
		APIURL:              fgs.baseURL,
		ShowUnfinishedPages: os.Getenv("SHOW_UNFINISHED_PAGES") == "true",
	}, nil
}

func ParseCLI() (CLIConfig, error) {
	baseURL := apiBaseURL()
	flagSet := flag.NewFlagSet("monitor", flag.ContinueOnError)
	guiMode := flagSet.Bool("gui", false, "run in gui mode")
	backgroundWorker := flagSet.Bool("background-worker", false, "run a background worker (windows only)")
	storageDir := flagSet.String("dir", "", "store calculated signatures in this directory (default ~/.affiro)")
	showHelp := flagSet.Bool("help", false, "print help text and exit")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		fmt.Println(helpText)
		return CLIConfig{}, fmt.Errorf("parsing the command line: %w", err)
	}
	return handleCommand(os.Args[1:], flags{
		showHelp:         *showHelp,
		storageDir:       *storageDir,
		backgroundWorker: *backgroundWorker,
		guiMode:          *guiMode,
		baseURL:          baseURL,
	})
}
