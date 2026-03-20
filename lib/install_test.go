package lib

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGoarchToTerramate(t *testing.T) {
	tests := []struct {
		goarch   string
		expected string
	}{
		{"amd64", "x86_64"},
		{"386", "i386"},
		{"arm64", "arm64"},
		{"arm", "arm"},
	}
	for _, tt := range tests {
		got := goarchToTerramate(tt.goarch)
		if got != tt.expected {
			t.Errorf("goarchToTerramate(%q) = %q, want %q", tt.goarch, got, tt.expected)
		}
	}
}

func TestBuildDownloadURL(t *testing.T) {
	url := buildDownloadURL("0.16.0")
	if url == "" {
		t.Fatal("expected non-empty URL")
	}

	goos := runtime.GOOS
	// URL should contain the version and the OS.
	if !containsAll(url, []string{"0.16.0", goos}) {
		t.Errorf("unexpected URL: %s", url)
	}
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestGetInstallLocation(t *testing.T) {
	loc := GetInstallLocation()
	if loc == "" {
		t.Fatal("expected non-empty install location")
	}
	if _, err := os.Stat(loc); os.IsNotExist(err) {
		t.Errorf("install location %s was not created", loc)
	}
}

func TestAddRecentAndGetRecentVersions(t *testing.T) {
	// Use a temporary home-like directory so we don't pollute the real one.
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	AddRecent("0.16.0")
	AddRecent("0.15.5")
	AddRecent("0.15.4")
	AddRecent("0.15.3") // Should push 0.15.4 out (cap = 3).

	recent, err := GetRecentVersions()
	if err != nil {
		t.Fatalf("GetRecentVersions: %v", err)
	}
	if len(recent) != maxRecentVersions {
		t.Errorf("expected %d recent versions, got %d: %v", maxRecentVersions, len(recent), recent)
	}
	// Most recent first.
	if recent[0] != "0.15.3 *recent" {
		t.Errorf("expected most recent to be 0.15.3, got %q", recent[0])
	}
}

func TestRecentDeduplicate(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	AddRecent("0.16.0")
	AddRecent("0.15.5")
	AddRecent("0.16.0") // duplicate – should move to top without increasing count.

	recent, err := GetRecentVersions()
	if err != nil {
		t.Fatalf("GetRecentVersions: %v", err)
	}
	if len(recent) != 2 {
		t.Errorf("expected 2 recent versions (no duplicates), got %d: %v", len(recent), recent)
	}
	if recent[0] != "0.16.0 *recent" {
		t.Errorf("expected 0.16.0 to be most recent, got %q", recent[0])
	}
}

func TestInstallableBinLocation_WritableDir(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "terramate")
	got := InstallableBinLocation(binPath)
	if got != binPath {
		t.Errorf("expected %q, got %q", binPath, got)
	}
}

func TestInstallableBinLocation_FallbackToHomeBin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	// /nonexistent-dir should not be writable.
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create a dir, then make it non-writable.
	noWriteDir := filepath.Join(tmpDir, "nowrite")
	os.MkdirAll(noWriteDir, 0555)
	defer os.Chmod(noWriteDir, 0755)

	binPath := filepath.Join(noWriteDir, "terramate")
	got := InstallableBinLocation(binPath)

	expectedBin := filepath.Join(tmpDir, "bin", "terramate")
	if got != expectedBin {
		t.Errorf("expected fallback path %q, got %q", expectedBin, got)
	}
}

func TestPrepareBinPath_RemovesExistingSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "versioned")
	link := filepath.Join(tmpDir, "terramate")

	if err := os.WriteFile(target, []byte("bin"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	if err := prepareBinPath(link); err != nil {
		t.Fatalf("prepareBinPath: %v", err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("expected symlink to be removed, got err=%v", err)
	}
}

func TestPrepareBinPath_RefusesToOverwriteRegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	regularFile := filepath.Join(tmpDir, "terramate")

	if err := os.WriteFile(regularFile, []byte("real binary"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := prepareBinPath(regularFile)
	if err == nil {
		t.Fatal("expected error for existing regular file")
	}
}
