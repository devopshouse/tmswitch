package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	lib "github.com/devopshouse/tmswitch/lib"
)

func TestResolveInstallRequest_PrioritizesArgOverConfig(t *testing.T) {
	t.Setenv(envVersion, "0.18.0")
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "version = \"0.16.0\"\n")

	version, binPath, interactive := resolveInstallRequest(
		[]string{"0.17.0"},
		projectDir,
		homeDir,
		"/custom/bin/terramate",
	)

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.17.0" {
		t.Fatalf("expected CLI arg to win, got %q", version)
	}
	if binPath != "/custom/bin/terramate" {
		t.Fatalf("expected CLI bin path to remain unchanged, got %q", binPath)
	}
}

func TestResolveInstallRequest_PrioritizesEnvVersionOverFilesAndTOML(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "version = \"0.16.0\"\n")
	mustWriteFile(t, filepath.Join(projectDir, rcFilename), "0.15.0\n")
	t.Setenv(envVersion, "0.18.0")

	version, _, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/usr/local/bin/terramate")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.18.0" {
		t.Fatalf("expected environment version to win, got %q", version)
	}
}

func TestResolveInstallRequest_PrioritizesTOMLVersionOverVersionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "version = \"0.16.0\"\n")
	mustWriteFile(t, filepath.Join(projectDir, rcFilename), "0.15.0\n")
	mustWriteFile(t, filepath.Join(projectDir, tmvFilename), "0.14.0\n")

	version, _, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/usr/local/bin/terramate")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.15.0" {
		t.Fatalf("expected version file to win over TOML, got %q", version)
	}
}

func TestResolveInstallRequest_UsesHomeTOMLWhenProjectHasNoTOML(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(homeDir, tomlFilename), "version = \"0.16.0\"\n")
	mustWriteFile(t, filepath.Join(projectDir, rcFilename), "0.15.0\n")

	version, _, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/usr/local/bin/terramate")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.15.0" {
		t.Fatalf("expected version file to win over home TOML, got %q", version)
	}
}

func TestResolveInstallRequest_UsesVersionFilesWhenNoTOML(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, rcFilename), "0.15.0\n")
	mustWriteFile(t, filepath.Join(projectDir, tmvFilename), "0.14.0\n")

	version, _, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/usr/local/bin/terramate")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.15.0" {
		t.Fatalf("expected rc file to win, got %q", version)
	}
}

func TestResolveInstallRequest_UsesTOMLVersionWhenNoVersionFilesExist(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "version = \"0.16.0\"\n")

	version, _, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/usr/local/bin/terramate")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.16.0" {
		t.Fatalf("expected TOML version to be used, got %q", version)
	}
}

func TestResolveInstallRequest_UsesTOMLDefaultVersionAsFallback(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "default-version = \"0.16.0\"\n")

	version, _, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/usr/local/bin/terramate")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.16.0" {
		t.Fatalf("expected TOML default-version to be used, got %q", version)
	}
}

func TestResolveInstallRequest_UsesEnvDefaultVersionBeforeTOMLDefaultVersion(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "default-version = \"0.16.0\"\n")
	t.Setenv(envDefault, "0.17.0")

	version, _, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/usr/local/bin/terramate")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.17.0" {
		t.Fatalf("expected environment default version to win, got %q", version)
	}
}

func TestResolveInstallRequest_TOMLBinStillAppliesWhenVersionComesFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "bin = \"/custom/bin/terramate\"\nversion = \"0.16.0\"\n")
	mustWriteFile(t, filepath.Join(projectDir, rcFilename), "0.15.0\n")

	version, binPath, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/usr/local/bin/terramate")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.15.0" {
		t.Fatalf("expected version file to win, got %q", version)
	}
	if binPath != "/custom/bin/terramate" {
		t.Fatalf("expected TOML bin path to be applied, got %q", binPath)
	}
}

func TestResolveInstallRequest_UsesEnvBinPathOverTOMLWhenCLIDefault(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "bin = \"/toml/bin/terramate\"\ndefault-version = \"0.16.0\"\n")
	t.Setenv(envBinPath, "/env/bin/terramate")

	version, binPath, interactive := resolveInstallRequest(nil, projectDir, homeDir, defaultBin)

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.16.0" {
		t.Fatalf("expected version to resolve from TOML default, got %q", version)
	}
	if binPath != "/env/bin/terramate" {
		t.Fatalf("expected environment bin path to win, got %q", binPath)
	}
}

func TestResolveInstallRequest_CLIBinPathOverridesEnvBinPath(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "default-version = \"0.16.0\"\n")
	t.Setenv(envBinPath, "/env/bin/terramate")

	_, binPath, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/cli/bin/terramate")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if binPath != "/cli/bin/terramate" {
		t.Fatalf("expected CLI bin path to win, got %q", binPath)
	}
}

func TestResolveInstallRequest_UsesCLIDefaultBeforeEnvAndTOMLDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)
	mustWriteFile(t, filepath.Join(projectDir, tomlFilename), "default-version = \"0.16.0\"\n")
	t.Setenv(envDefault, "0.17.0")

	version, _, interactive := resolveInstallRequestWithDefault(nil, projectDir, homeDir, "/usr/local/bin/terramate", "0.18.0")

	if interactive {
		t.Fatal("expected non-interactive install request")
	}
	if version != "0.18.0" {
		t.Fatalf("expected CLI default version to win, got %q", version)
	}
}

func TestResolveInstallRequest_FallsBackToInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	mustMkdirAll(t, homeDir)
	mustMkdirAll(t, projectDir)

	version, _, interactive := resolveInstallRequest(nil, projectDir, homeDir, "/usr/local/bin/terramate")

	if !interactive {
		t.Fatal("expected interactive mode when no version source exists")
	}
	if version != "" {
		t.Fatalf("expected no resolved version, got %q", version)
	}
}

func TestValidateRequestedVersion_TrimsAndValidatesReleaseExists(t *testing.T) {
	fetch := func(bool) ([]string, error) {
		return []string{"0.16.0", "0.15.0-rc1"}, nil
	}

	version, warnErr, fatalErr := validateRequestedVersion(" 0.16.0 \n", fetch)

	if warnErr != nil {
		t.Fatalf("unexpected warning: %v", warnErr)
	}
	if fatalErr != nil {
		t.Fatalf("unexpected fatal error: %v", fatalErr)
	}
	if version != "0.16.0" {
		t.Fatalf("expected trimmed version, got %q", version)
	}
}

func TestValidateRequestedVersion_ReturnsFatalForUnknownRelease(t *testing.T) {
	fetch := func(bool) ([]string, error) {
		return []string{"0.16.0"}, nil
	}

	_, warnErr, fatalErr := validateRequestedVersion("0.99.0", fetch)

	if warnErr != nil {
		t.Fatalf("unexpected warning: %v", warnErr)
	}
	if fatalErr == nil {
		t.Fatal("expected fatal error for unknown release")
	}
}

func TestValidateRequestedVersion_ReturnsWarningOnLookupFailure(t *testing.T) {
	fetch := func(bool) ([]string, error) {
		return nil, errors.New("lookup failed")
	}

	version, warnErr, fatalErr := validateRequestedVersion("0.16.0", fetch)

	if fatalErr != nil {
		t.Fatalf("unexpected fatal error: %v", fatalErr)
	}
	if warnErr == nil {
		t.Fatal("expected warning when release lookup fails")
	}
	if version != "0.16.0" {
		t.Fatalf("expected original version to be preserved, got %q", version)
	}
}

func TestLatestVersion_ReturnsNewestStable(t *testing.T) {
	version, err := lib.LatestVersion([]string{"0.16.2", "0.16.1", "0.16.0"})
	if err != nil {
		t.Fatalf("LatestVersion: %v", err)
	}
	if version != "0.16.2" {
		t.Fatalf("expected 0.16.2, got %q", version)
	}
}

func TestLatestMatchingVersion_StablePrefix(t *testing.T) {
	version, err := lib.LatestMatchingVersion([]string{"0.16.2", "0.16.1", "0.15.9"}, "0.16")
	if err != nil {
		t.Fatalf("LatestMatchingVersion: %v", err)
	}
	if version != "0.16.2" {
		t.Fatalf("expected 0.16.2, got %q", version)
	}
}

func TestLatestMatchingVersion_PrereleasePrefix(t *testing.T) {
	version, err := lib.LatestMatchingVersion([]string{"0.17.0-rc2", "0.17.0-rc1", "0.16.2"}, "0.17")
	if err != nil {
		t.Fatalf("LatestMatchingVersion: %v", err)
	}
	if version != "0.17.0-rc2" {
		t.Fatalf("expected 0.17.0-rc2, got %q", version)
	}
}

func TestLatestMatchingVersion_ExactVersion(t *testing.T) {
	version, err := lib.LatestMatchingVersion([]string{"0.17.0-rc2", "0.16.2"}, "0.16.2")
	if err != nil {
		t.Fatalf("LatestMatchingVersion: %v", err)
	}
	if version != "0.16.2" {
		t.Fatalf("expected 0.16.2, got %q", version)
	}
}

func TestDeriveMirrorURLs_FromRepoURL(t *testing.T) {
	apiURL, downloadURL, err := deriveMirrorURLs("https://github.com/example/terramate")
	if err != nil {
		t.Fatalf("deriveMirrorURLs: %v", err)
	}
	if apiURL != "https://api.github.com/repos/example/terramate/releases?per_page=100" {
		t.Fatalf("unexpected api URL: %q", apiURL)
	}
	if downloadURL != "https://github.com/example/terramate/releases/download/" {
		t.Fatalf("unexpected download URL: %q", downloadURL)
	}
}

func TestDeriveMirrorURLs_Invalid(t *testing.T) {
	if _, _, err := deriveMirrorURLs("https://example.com/releases"); err == nil {
		t.Fatal("expected invalid mirror error")
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
