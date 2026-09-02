package buildinfo

import "testing"

func Test(t *testing.T) {
	v := FullVersion()
	expect := Asset + " " + Version + "-" + Time + "-" + CommitSHA
	if v != expect {
		t.Errorf("full version mismatched")
	}
	Version = "v0.0.1"
	expect = Asset + " " + Version
	v = FullVersion()
	if v != expect {
		t.Errorf("full version mismatched")
	}
}
