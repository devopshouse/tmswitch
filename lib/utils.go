package lib

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// GetHomeDirectory returns the current user's home directory.
func GetHomeDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	return home
}

// GetCurrentDirectory returns the current working directory.
func GetCurrentDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}
	return dir
}

// FileExists returns true if a file exists at the given path.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists returns true if a directory exists at the given path.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// CreateDirIfNotExist creates a directory (and all parents) if it does not exist.
func CreateDirIfNotExist(path string) {
	if !DirExists(path) {
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", path, err)
		}
	}
}

// IsDirWritable returns true if the given directory is writable.
func IsDirWritable(path string) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	tmpFile := filepath.Join(path, ".tmswitch_write_test")
	f, err := os.Create(tmpFile) // #nosec G304
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(tmpFile)
	return true
}

// CheckSymlink returns true if the given path is a symlink.
func CheckSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// RemoveSymlink removes the symlink at the given path.
func RemoveSymlink(path string) {
	if err := os.Remove(path); err != nil {
		log.Fatalf("Failed to remove symlink %s: %v", path, err)
	}
}

// CreateSymlink creates a symlink from src to dest.
func CreateSymlink(src, dest string) {
	if err := os.Symlink(src, dest); err != nil {
		log.Fatalf("Failed to create symlink %s -> %s: %v", dest, src, err)
	}
}

// RenameFile renames src to dest.
func RenameFile(src, dest string) {
	if err := os.Rename(src, dest); err != nil {
		log.Fatalf("Failed to rename %s to %s: %v", src, dest, err)
	}
}

// RemoveFile removes a file at the given path.
func RemoveFile(path string) {
	if err := os.Remove(path); err != nil {
		log.Printf("Failed to remove file %s: %v", path, err)
	}
}

// ReadLines reads all lines from a file, trimming whitespace.
func ReadLines(path string) ([]string, error) {
	f, err := os.Open(path) // #nosec G304
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// WriteLines writes a slice of strings to a file, one per line.
func WriteLines(lines []string, path string) {
	f, err := os.Create(path) // #nosec G304
	if err != nil {
		log.Fatalf("Failed to write file %s: %v", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	w.Flush()
}

// RetrieveFileContents reads the first non-empty line from a file.
func RetrieveFileContents(path string) string {
	lines, err := ReadLines(path)
	if err != nil || len(lines) == 0 {
		log.Fatalf("Failed to read version from %s: %v", path, err)
	}
	return lines[0]
}

// ConvertExecutableExt appends ".exe" on Windows if not already present.
func ConvertExecutableExt(path string) string {
	if runtime.GOOS == "windows" {
		if filepath.Ext(path) != ".exe" {
			return path + ".exe"
		}
	}
	return path
}

// Path returns the directory component of the given file path.
func Path(p string) string {
	return filepath.Dir(p)
}
