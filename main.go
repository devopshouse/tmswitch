package main

/*
 * tmswitch - Switch between different versions of terramate
 *
 * Usage:
 *   tmswitch               - interactive version selector
 *   tmswitch <version>     - install a specific version
 *   tmswitch --version     - display tmswitch version
 *   tmswitch --help        - display help message
 *   tmswitch --pre         - include pre-release versions in the list
 *
 * Version files (checked in order):
 *   .tmswitchrc            - plain text file containing a version string
 *   .terramate-version     - plain text file containing a version string
 *
 * TOML config file:
 *   .tmswitch.toml         - supports `bin` and `version` keys
 */

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pborman/getopt"
	"github.com/spf13/viper"

	lib "github.com/devopshouse/tmswitch/lib"
)

const (
	defaultBin   = "/usr/local/bin/terramate"
	rcFilename   = ".tmswitchrc"
	tmvFilename  = ".terramate-version"
	tomlFilename = ".tmswitch.toml"
	envVersion   = "TM_VERSION"
	envDefault   = "TM_DEFAULT_VERSION"
	envBinPath   = "TM_BINARY_PATH"
)

var appVersion = "dev"

func main() {
	dir := lib.GetCurrentDirectory()

	custBinPath := getopt.StringLong("bin", 'b',
		lib.ConvertExecutableExt(defaultBin),
		"Custom binary path. Ex: tmswitch -b /home/user/bin/terramate")
	versionFlag := getopt.BoolLong("version", 'v', "Display tmswitch version")
	helpFlag := getopt.BoolLong("help", 'h', "Display help message")
	chDirPath := getopt.StringLong("chdir", 'c', dir,
		"Switch to a different working directory before executing")
	preFlag := getopt.BoolLong("pre", 'p', "Include pre-release versions")

	getopt.Parse()
	args := getopt.Args()

	switch {
	case *versionFlag:
		fmt.Printf("\ntmswitch version: %s\n", appVersion)
		return

	case *helpFlag:
		usageMessage()
		return
	}

	homedir := lib.GetHomeDirectory()
	version, binPath, interactive := resolveInstallRequest(args, *chDirPath, homedir, *custBinPath)
	if interactive {
		selectVersionInteractive(&binPath, *preFlag)
		return
	}
	installVersion(version, &binPath)
}

type tomlConfig struct {
	Version        string
	DefaultVersion string
	BinPath        string
}

// resolveInstallRequest resolves the effective version/bin request, following
// CLI args first, then plain-text version files, then TOML explicit/default
// versions. If no version is resolved, interactive indicates that the selector
// should be shown.
func resolveInstallRequest(args []string, chDirPath, homedir, cliBinPath string) (string, string, bool) {
	tomlConfigFile := filepath.Join(chDirPath, tomlFilename)
	homeTOMLConfigFile := filepath.Join(homedir, tomlFilename)
	rcFile := filepath.Join(chDirPath, rcFilename)
	tmVersionFile := filepath.Join(chDirPath, tmvFilename)

	binPath := cliBinPath
	cfg := tomlConfig{}

	switch {
	case lib.FileExists(tomlConfigFile):
		cfg = readTOMLConfig(binPath, chDirPath)
	case lib.FileExists(homeTOMLConfigFile):
		cfg = readTOMLConfig(binPath, homedir)
	}
	binPath = cfg.BinPath

	if envBin := strings.TrimSpace(os.Getenv(envBinPath)); envBin != "" && cliBinPath == lib.ConvertExecutableExt(defaultBin) {
		binPath = envBin
	}

	envRequestedVersion := strings.TrimSpace(os.Getenv(envVersion))
	envDefaultVersion := strings.TrimSpace(os.Getenv(envDefault))

	switch {
	case len(args) == 1:
		return args[0], binPath, false
	case envRequestedVersion != "":
		return envRequestedVersion, binPath, false
	case lib.FileExists(rcFile):
		return lib.RetrieveFileContents(rcFile), binPath, false
	case lib.FileExists(tmVersionFile):
		return lib.RetrieveFileContents(tmVersionFile), binPath, false
	case cfg.Version != "":
		return cfg.Version, binPath, false
	case envDefaultVersion != "":
		return envDefaultVersion, binPath, false
	case cfg.DefaultVersion != "":
		return cfg.DefaultVersion, binPath, false
	default:
		return "", binPath, true
	}
}

func installVersion(version string, binPath *string) {
	version, warnErr, fatalErr := validateRequestedVersion(version, lib.GetVersionList)
	if fatalErr != nil {
		fmt.Println(fatalErr.Error())
		usageMessage()
		os.Exit(1)
	}
	if warnErr != nil {
		fmt.Printf("Warning: could not fetch releases list (%v). Attempting install anyway.\n", warnErr)
	}
	lib.Install(version, *binPath)
}

func validateRequestedVersion(requested string, fetchVersions func(bool) ([]string, error)) (string, error, error) {
	version := strings.TrimSpace(requested)
	if !lib.ValidVersionFormat(version) {
		return "", nil, fmt.Errorf("invalid terramate version format %q. Expected format: #.#.# or #.#.#-@", version)
	}

	versions, err := fetchVersions(true)
	if err != nil {
		return version, err, nil
	}
	if !lib.VersionExist(version, versions) {
		return "", nil, fmt.Errorf("version %q not found in the available releases", version)
	}
	return version, nil, nil
}

// selectVersionInteractive shows an interactive prompt to pick a version.
func selectVersionInteractive(binPath *string, includePrerelease bool) {
	versions, err := lib.GetVersionList(includePrerelease)
	if err != nil {
		log.Fatalf("Failed to fetch version list: %v", err)
	}
	if len(versions) == 0 {
		fmt.Println("No terramate versions found.")
		os.Exit(1)
	}

	// Prepend recently used versions for convenience.
	recent, _ := lib.GetRecentVersions()
	var displayList []string
	displayList = append(displayList, recent...)
	for _, v := range versions {
		if !lib.VersionExist(v+" *recent", recent) {
			displayList = append(displayList, v)
		}
	}

	prompt := promptui.Select{
		Label: "Select terramate version",
		Items: displayList,
		Size:  20,
	}

	_, selected, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt cancelled: %v\n", err)
		os.Exit(1)
	}

	// Strip the " *recent" marker if present.
	version := strings.TrimSuffix(selected, " *recent")
	lib.Install(version, *binPath)
}

// readTOMLConfig reads the .tmswitch.toml configuration file from the given
// directory, overriding binPath only if a custom bin is not already set.
func readTOMLConfig(binPath, dir string) tomlConfig {
	v := viper.New()
	v.SetConfigName(".tmswitch")
	v.SetConfigType("toml")
	v.AddConfigPath(dir)

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read TOML config: %v", err)
	}

	// If the user hasn't supplied a custom bin via -b, use the value from TOML.
	if binPath == lib.ConvertExecutableExt(defaultBin) {
		if cfgBin := v.GetString("bin"); cfgBin != "" {
			binPath = cfgBin
		}
	}

	return tomlConfig{
		Version:        v.GetString("version"),
		DefaultVersion: v.GetString("default-version"),
		BinPath:        binPath,
	}
}

func usageMessage() {
	fmt.Print(`
tmswitch - Switch between different versions of terramate

Usage:
  tmswitch                        Interactive version selector
  tmswitch <version>              Install a specific version (e.g. tmswitch 0.16.0)
  tmswitch --pre                  Include pre-release versions in the selector
  tmswitch --bin <path>           Custom install path for the terramate binary
  tmswitch --chdir <dir>          Look for version files in a different directory
  tmswitch --version              Display tmswitch version
  tmswitch --help                 Display this help message

Version files (in order of precedence):
  TM_VERSION                      Environment override for the Terramate version
  .tmswitchrc                     Plain text file with a version string
  .terramate-version              Plain text file with a version string (tfenv-style)
  TM_DEFAULT_VERSION              Environment fallback version
  .tmswitch.toml                  TOML file with optional 'version', 'default-version', and 'bin' keys
  TM_BINARY_PATH                  Environment override for the Terramate binary path

`)
}
