package lib

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	terramateGitHubURL = "https://github.com/terramate-io/terramate/releases/download/"
	installBinaryName  = "terramate"
	installPathSuffix  = "/.terramate.versions/"
	recentFile         = "RECENT"
	maxRecentVersions  = 3
)

// InstallOptions customizes version download and activation.
type InstallOptions struct {
	Arch            string
	BinPath         string
	DownloadBaseURL string
	DryRun          bool
	InstallPath     string
}

// GetInstallLocation returns (and creates if necessary) the directory where
// versioned terramate binaries are stored (~/.terramate.versions/).
func GetInstallLocation() string {
	return GetInstallLocationWithBase("")
}

// GetInstallLocationWithBase returns the directory where versioned terramate
// binaries are stored, rooted at installPath or the user's home when empty.
func GetInstallLocationWithBase(installPath string) string {
	base := installPath
	if base == "" {
		base = GetHomeDirectory()
	}
	location := filepath.Join(base, ".terramate.versions")
	CreateDirIfNotExist(location)
	return location
}

// Install downloads (if necessary) and activates the requested version.
// It returns the path to the active symlink.
func Install(version, binPath string) string {
	return InstallWithOptions(version, InstallOptions{BinPath: binPath})
}

// InstallWithOptions downloads (if necessary) and activates the requested version.
func InstallWithOptions(version string, options InstallOptions) string {
	release := AcquireInstallLock()
	binPath := InstallableBinLocation(options.BinPath)
	installLocation := GetInstallLocationWithBase(options.InstallPath)
	versionedBin := ConvertExecutableExt(filepath.Join(installLocation, installBinaryName+"_"+version))

	if FileExists(versionedBin) {
		if options.DryRun {
			fmt.Printf("[DRY-RUN] Would switch terramate to version %q using %s\n", version, binPath)
			release()
			return binPath
		}
		activateVersion(versionedBin, binPath, version, options.InstallPath)
		release()
		return binPath
	}

	url := buildDownloadURL(version, options.DownloadBaseURL, options.Arch)
	if options.DryRun {
		fmt.Printf("[DRY-RUN] Would download terramate v%s from %s and activate %s\n", version, url, binPath)
		release()
		return binPath
	}

	archivePath, err := downloadFile(installLocation, url)
	if err != nil {
		release()
		fmt.Printf("Error downloading terramate v%s: %v\n", version, err)
		os.Exit(1)
	}

	if err := extractBinary(archivePath, installBinaryName, versionedBin); err != nil {
		os.Remove(archivePath)
		release()
		fmt.Printf("Error extracting terramate binary: %v\n", err)
		os.Exit(1)
	}
	os.Remove(archivePath)

	if err := os.Chmod(versionedBin, 0755); err != nil {
		log.Printf("Warning: could not set executable bit on %s: %v", versionedBin, err)
	}

	activateVersion(versionedBin, binPath, version, options.InstallPath)
	release()
	return binPath
}

// activateVersion removes the existing symlink/binary at binPath and creates a
// new symlink pointing to versionedBin.
func activateVersion(versionedBin, binPath, version, installPath string) {
	if err := prepareBinPath(binPath); err != nil {
		log.Fatalf("Failed to activate terramate version %q: %v", version, err)
	}
	CreateSymlink(versionedBin, binPath)
	fmt.Printf("Switched terramate to version %q\n", version)
	AddRecent(version, installPath)
}

func prepareBinPath(binPath string) error {
	info, err := os.Lstat(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect existing binary path %s: %w", binPath, err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing to overwrite existing non-symlink at %s", binPath)
	}

	if err := os.Remove(binPath); err != nil {
		return fmt.Errorf("remove existing symlink %s: %w", binPath, err)
	}
	return nil
}

// buildDownloadURL constructs the GitHub release download URL for the current
// OS and architecture.
//
// Terramate release asset naming (as of 2024):
//
//	terramate_{version}_{os}_{arch}.tar.gz   (Linux/Darwin)
//	terramate_{version}_{os}_{arch}.zip      (Windows)
//
// where {arch} uses "x86_64" for amd64, "arm64" for arm64, "i386" for 386.
func buildDownloadURL(version, downloadBaseURL, archOverride string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if archOverride != "" {
		goarch = archOverride
	}
	baseURL := terramateGitHubURL
	if downloadBaseURL != "" {
		baseURL = downloadBaseURL
	}

	arch := goarchToTerramate(goarch)
	ext := "tar.gz"
	if goos == osWindows {
		ext = "zip"
	}

	filename := fmt.Sprintf("terramate_%s_%s_%s.%s", version, goos, arch, ext)
	return fmt.Sprintf("%sv%s/%s", baseURL, version, filename)
}

// goarchToTerramate maps a GOARCH value to the arch string used in terramate
// release asset names.
func goarchToTerramate(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "386":
		return "i386"
	default:
		return goarch // arm64, etc. match directly
	}
}

// downloadFile downloads the file at the given URL into dir and returns the
// local path of the downloaded file.
func downloadFile(dir, url string) (string, error) {
	resp, err := http.Get(url) // #nosec G107
	if err != nil {
		return "", fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status %d for %s", resp.StatusCode, url)
	}

	// Derive the local filename from the URL.
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]
	destPath := filepath.Join(dir, filename)

	out, err := os.Create(destPath) // #nosec G304
	if err != nil {
		return "", fmt.Errorf("create %s: %w", destPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("writing %s: %w", destPath, err)
	}
	return destPath, nil
}

// extractBinary extracts the binary named binaryName from a .tar.gz or .zip
// archive and writes it to destPath.
func extractBinary(archivePath, binaryName, destPath string) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath, binaryName, destPath)
	}
	return extractFromTarGz(archivePath, binaryName, destPath)
}

func extractFromTarGz(archivePath, binaryName, destPath string) error {
	f, err := os.Open(archivePath) // #nosec G304
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// Match the binary by base name (handles paths like "./terramate").
		if filepath.Base(hdr.Name) == binaryName && hdr.Typeflag == tar.TypeReg {
			out, err := os.Create(destPath) // #nosec G304
			if err != nil {
				return err
			}
			defer out.Close()
			if _, err := io.Copy(out, tr); err != nil { //nolint:gosec // G110: archive is a known trusted terramate release
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("binary %q not found in archive %s", binaryName, archivePath)
}

func extractFromZip(archivePath, binaryName, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == binaryName || filepath.Base(f.Name) == binaryName+".exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.Create(destPath) // #nosec G304
			if err != nil {
				return err
			}
			defer out.Close()

			if _, err := io.Copy(out, rc); err != nil { //nolint:gosec // G110: archive is a known trusted terramate release
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("binary %q not found in zip %s", binaryName, archivePath)
}

// installLocation is a candidate path for placing the terramate binary symlink.
type installLocation struct {
	path   string
	create bool // whether to create the directory if it doesn't exist
}

// InstallableBinLocation returns the effective install path for the terramate
// binary symlink. It tries each candidate in order, falling back to ~/bin
// (creating it if necessary) when the primary location is not writable.
func InstallableBinLocation(userBinPath string) string {
	home := GetHomeDirectory()
	homeBin := ConvertExecutableExt(filepath.Join(home, "bin", installBinaryName))

	candidates := []installLocation{
		{path: userBinPath, create: false},
		{path: homeBin, create: true},
	}

	for _, loc := range candidates {
		binDir := Path(loc.path)
		if !DirExists(binDir) {
			if !loc.create {
				continue
			}
			CreateDirIfNotExist(binDir)
		}
		if !IsDirWritable(binDir) {
			continue
		}
		if loc.path != userBinPath {
			fmt.Printf("No write permission to default bin location.\n")
			fmt.Printf("Installing terramate at %s\n", loc.path)
			if !IsInPath(binDir) {
				fmt.Printf("RUN `export PATH=$PATH:%s` to add it to your PATH.\n", binDir)
			}
		}
		return loc.path
	}

	fmt.Printf("[Error] Could not find a writable location for the terramate binary.\n")
	os.Exit(1)
	return ""
}

// AddRecent adds version to the RECENT file (capped at maxRecentVersions).
func AddRecent(version, installPath string) {
	installLocation := GetInstallLocationWithBase(installPath)
	recentPath := filepath.Join(installLocation, recentFile)

	existing, err := GetRecentVersions()
	if err != nil {
		log.Printf("Warning: failed to load recent versions: %v", err)
	}
	// Strip the " *recent" suffix that GetRecentVersions appends.
	var clean []string
	for _, v := range existing {
		clean = append(clean, strings.TrimSuffix(v, " *recent"))
	}

	// Remove duplicates.
	var updated []string
	updated = append(updated, version)
	for _, v := range clean {
		if v != version {
			updated = append(updated, v)
		}
	}
	if len(updated) > maxRecentVersions {
		updated = updated[:maxRecentVersions]
	}
	WriteLines(updated, recentPath)
}

// GetRecentVersions returns the recently used versions with a " *recent" suffix.
func GetRecentVersions() ([]string, error) {
	return GetRecentVersionsFrom("")
}

// GetRecentVersionsFrom returns recently used versions from a custom install base.
func GetRecentVersionsFrom(installPath string) ([]string, error) {
	installLocation := GetInstallLocationWithBase(installPath)
	recentPath := filepath.Join(installLocation, recentFile)

	if !FileExists(recentPath) {
		return nil, nil
	}

	lines, err := ReadLines(recentPath)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, line := range lines {
		if ValidVersionFormat(line) {
			result = append(result, fmt.Sprintf("%s *recent", line))
		}
	}
	return result, nil
}
