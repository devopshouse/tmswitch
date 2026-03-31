package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	f, err := os.CreateTemp("", "tmswitch-test-*")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if !FileExists(f.Name()) {
		t.Errorf("expected %s to exist", f.Name())
	}
	if FileExists(f.Name() + "-nonexistent") {
		t.Error("expected non-existent file to not exist")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()
	if !DirExists(tmpDir) {
		t.Errorf("expected %s to exist as a dir", tmpDir)
	}
	if DirExists(filepath.Join(tmpDir, "nope")) {
		t.Error("expected non-existent dir to not exist")
	}
}

func TestCreateDirIfNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "a", "b", "c")
	CreateDirIfNotExist(newDir)
	if !DirExists(newDir) {
		t.Errorf("expected directory %s to have been created", newDir)
	}
}

func TestReadLinesWriteLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "lines.txt")

	lines := []string{"0.16.0", "0.15.5"}
	WriteLines(lines, path)

	got, err := ReadLines(path)
	if err != nil {
		t.Fatalf("ReadLines: %v", err)
	}
	if len(got) != len(lines) {
		t.Fatalf("expected %d lines, got %d", len(lines), len(got))
	}
	for i, l := range lines {
		if got[i] != l {
			t.Errorf("line %d: expected %q, got %q", i, l, got[i])
		}
	}
}

func TestRetrieveFileContents(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "version")
	WriteLines([]string{"0.16.0"}, path)

	v := RetrieveFileContents(path)
	if v != "0.16.0" {
		t.Errorf("expected 0.16.0, got %q", v)
	}
}

func TestCheckSymlinkCreateRemove(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	link := filepath.Join(tmpDir, "link")

	if err := os.WriteFile(target, []byte("hi"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if CheckSymlink(link) {
		t.Error("link should not exist yet")
	}
	if CheckSymlink(target) {
		t.Error("regular file should not be treated as symlink")
	}

	CreateSymlink(target, link)
	if !CheckSymlink(link) {
		t.Error("link should exist after creation")
	}

	RemoveSymlink(link)
	if CheckSymlink(link) {
		t.Error("link should not exist after removal")
	}
}

func TestConvertExecutableExt(t *testing.T) {
	// On non-Windows, path should be returned unchanged.
	path := "/usr/local/bin/terramate"
	got := ConvertExecutableExt(path)
	// We only test that the function returns a non-empty string.
	if got == "" {
		t.Error("expected non-empty result")
	}
}

func TestPath(t *testing.T) {
	p := Path("/usr/local/bin/terramate")
	if p != "/usr/local/bin" {
		t.Errorf("expected /usr/local/bin, got %q", p)
	}
}

func TestIsInPath(t *testing.T) {
	orig := os.Getenv("PATH")
	defer os.Setenv("PATH", orig)

	os.Setenv("PATH", "/usr/local/bin:/usr/bin:/home/user/bin")
	if !IsInPath("/usr/local/bin") {
		t.Error("expected /usr/local/bin to be in PATH")
	}
	if IsInPath("/nonexistent/bin") {
		t.Error("expected /nonexistent/bin to not be in PATH")
	}
}

func TestAcquireInstallLock(t *testing.T) {
	release := AcquireInstallLock()

	// Second acquisition should fail — but we can only verify the lock file exists.
	lockPath := filepath.Join(os.TempDir(), installLockFile)
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("expected lock file to exist after AcquireInstallLock")
	}

	release()

	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("expected lock file to be removed after release")
	}
}
