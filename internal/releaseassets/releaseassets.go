// Package releaseassets defines the shared naming scheme used to publish and fetch
// per-platform release binaries in S3, so the publish path (cmd/playground-release)
// and the serve path (cmd/playground) can never disagree on a key layout.
package releaseassets

import (
	"fmt"
	"slices"
)

// CLIFileName is the S3 "file name" prefix used for the affiro CLI binary, shared
// between cmd/playground-release's mainPaths entry and the download-proxy logic.
const CLIFileName = "affiro"

// The GOOS and GOARCH values this scheme covers.
const (
	OSLinux   = "linux"
	OSDarwin  = "darwin"
	OSWindows = "windows"

	ArchAMD64 = "amd64"
	ArchARM64 = "arm64"
)

// KnownOSes are the only GOOS values ever accepted from a caller or used to build an S3 key.
var KnownOSes = []string{OSLinux, OSDarwin, OSWindows}

// KnownArches are the only GOARCH values ever accepted from a caller or used to build an S3 key.
var KnownArches = []string{ArchAMD64, ArchARM64}

// IsKnownOS reports whether goos is one of KnownOSes.
func IsKnownOS(goos string) bool {
	return slices.Contains(KnownOSes, goos)
}

// IsKnownArch reports whether goarch is one of KnownArches.
func IsKnownArch(goarch string) bool {
	return slices.Contains(KnownArches, goarch)
}

// S3OSName maps a Go GOOS value to the name used in S3 object keys.
func S3OSName(goos string) string {
	switch goos {
	case OSWindows:
		return "win"
	case OSDarwin:
		return "osx"
	default:
		return goos
	}
}

// BuildName returns the S3 object basename for the given file name, GOOS, and GOARCH,
// e.g. BuildName("affiro", "darwin", "arm64") -> "affiro_osx_arm64".
func BuildName(fileName, goos, goarch string) string {
	name := fmt.Sprintf("%s_%s_%s", fileName, S3OSName(goos), goarch)
	if goos == OSWindows {
		name += ".exe"
	}
	return name
}

// Key returns the full S3 object key for the given file name, version ("latest" or a
// published version string), GOOS, and GOARCH.
func Key(fileName, version, goos, goarch string) string {
	return fmt.Sprintf("%s/%s/%s", fileName, version, BuildName(fileName, goos, goarch))
}
