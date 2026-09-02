// internal/releaseassets/releaseassets_test.go
package releaseassets

import "testing"

func TestIsKnownOS(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		goos   string
		expect bool
	}{
		{"linux", true},
		{"darwin", true},
		{"windows", true},
		{"plan9", false},
		{"", false},
	}
	for _, tc := range tcs {
		t.Run(tc.goos, func(t *testing.T) {
			t.Parallel()
			if got := IsKnownOS(tc.goos); got != tc.expect {
				t.Errorf("IsKnownOS(%q) = %v, want %v", tc.goos, got, tc.expect)
			}
		})
	}
}

func TestIsKnownArch(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		goarch string
		expect bool
	}{
		{"amd64", true},
		{"arm64", true},
		{"386", false},
		{"", false},
	}
	for _, tc := range tcs {
		t.Run(tc.goarch, func(t *testing.T) {
			t.Parallel()
			if got := IsKnownArch(tc.goarch); got != tc.expect {
				t.Errorf("IsKnownArch(%q) = %v, want %v", tc.goarch, got, tc.expect)
			}
		})
	}
}

func TestS3OSName(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		goos   string
		expect string
	}{
		{"windows", "win"},
		{"darwin", "osx"},
		{"linux", "linux"},
	}
	for _, tc := range tcs {
		t.Run(tc.goos, func(t *testing.T) {
			t.Parallel()
			if got := S3OSName(tc.goos); got != tc.expect {
				t.Errorf("S3OSName(%q) = %q, want %q", tc.goos, got, tc.expect)
			}
		})
	}
}

func TestBuildName(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name     string
		fileName string
		goos     string
		goarch   string
		expect   string
	}{
		{name: "linux amd64", fileName: "affiro", goos: "linux", goarch: "amd64", expect: "affiro_linux_amd64"},
		{name: "darwin arm64", fileName: "affiro", goos: "darwin", goarch: "arm64", expect: "affiro_osx_arm64"},
		{name: "windows amd64 gets exe suffix", fileName: "affiro", goos: "windows", goarch: "amd64", expect: "affiro_win_amd64.exe"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := BuildName(tc.fileName, tc.goos, tc.goarch); got != tc.expect {
				t.Errorf("BuildName(%q, %q, %q) = %q, want %q", tc.fileName, tc.goos, tc.goarch, got, tc.expect)
			}
		})
	}
}

func TestKey(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name     string
		fileName string
		version  string
		goos     string
		goarch   string
		expect   string
	}{
		{name: "versioned windows", fileName: "affiro", version: "v1.4.2", goos: "windows", goarch: "amd64", expect: "affiro/v1.4.2/affiro_win_amd64.exe"},
		{name: "latest darwin", fileName: "affiro", version: "latest", goos: "darwin", goarch: "arm64", expect: "affiro/latest/affiro_osx_arm64"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Key(tc.fileName, tc.version, tc.goos, tc.goarch); got != tc.expect {
				t.Errorf("Key(...) = %q, want %q", got, tc.expect)
			}
		})
	}
}

func TestCLIFileName(t *testing.T) {
	t.Parallel()
	if CLIFileName != "affiro" {
		t.Errorf("expected CLIFileName to be %q, got %q", "affiro", CLIFileName)
	}
}
