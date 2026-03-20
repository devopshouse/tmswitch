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
	defaultBin    = "/usr/local/bin/terramate"
	rcFilename    = ".tmswitchrc"
	tmvFilename   = ".terramate-version"
	tomlFilename  = ".tmswitch.toml"
	appVersion    = "0.1.0"
)

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

	TOMLConfigFile := filepath.Join(*chDirPath, tomlFilename)
	HomeTOMLConfigFile := filepath.Join(homedir, tomlFilename)
	RCFile := filepath.Join(*chDirPath, rcFilename)
	TMVersionFile := filepath.Join(*chDirPath, tmvFilename)

	// TOML config takes highest precedence (after CLI flags).
	switch {
	case lib.FileExists(TOMLConfigFile) || lib.FileExists(HomeTOMLConfigFile):
		binPath := *custBinPath
		cfgVersion := ""
		if lib.FileExists(TOMLConfigFile) {
			cfgVersion, binPath = readTOMLConfig(binPath, *chDirPath)
		} else {
			cfgVersion, binPath = readTOMLConfig(binPath, homedir)
		}

		switch {
		case len(args) == 1:
			installVersionArg(args[0], &binPath)
		case lib.FileExists(RCFile) && len(args) == 0:
			v := lib.RetrieveFileContents(RCFile)
			installVersion(v, &binPath)
		case lib.FileExists(TMVersionFile) && len(args) == 0:
			v := lib.RetrieveFileContents(TMVersionFile)
			installVersion(v, &binPath)
		case cfgVersion != "" && len(args) == 0:
			installVersion(cfgVersion, &binPath)
		default:
			selectVersionInteractive(&binPath, *preFlag)
		}

	case len(args) == 1:
		binPath := *custBinPath
		installVersionArg(args[0], &binPath)

	case lib.FileExists(RCFile) && len(args) == 0:
		binPath := *custBinPath
		v := lib.RetrieveFileContents(RCFile)
		installVersion(v, &binPath)

	case lib.FileExists(TMVersionFile) && len(args) == 0:
		binPath := *custBinPath
		v := lib.RetrieveFileContents(TMVersionFile)
		installVersion(v, &binPath)

	default:
		binPath := *custBinPath
		selectVersionInteractive(&binPath, *preFlag)
	}
}

// installVersionArg validates and installs the version supplied as a CLI arg.
func installVersionArg(requested string, binPath *string) {
	if !lib.ValidVersionFormat(requested) {
		fmt.Println("Invalid terramate version format. Expected format: #.#.# or #.#.#-@")
		usageMessage()
		os.Exit(1)
	}

	// Attempt to validate against the releases list; skip validation on API errors.
	versions, err := lib.GetVersionList(true)
	if err != nil {
		fmt.Printf("Warning: could not fetch releases list (%v). Attempting install anyway.\n", err)
	} else if !lib.VersionExist(requested, versions) {
		fmt.Printf("Version %q not found in the available releases.\n", requested)
		os.Exit(1)
	}

	lib.Install(requested, *binPath)
}

// installVersion installs the given version (read from a config/version file).
func installVersion(version string, binPath *string) {
	version = strings.TrimSpace(version)
	if !lib.ValidVersionFormat(version) {
		fmt.Printf("Invalid version %q in version file.\n", version)
		os.Exit(1)
	}
	lib.Install(version, *binPath)
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
// Returns the version and effective bin path.
func readTOMLConfig(binPath, dir string) (string, string) {
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

	version := v.GetString("version")
	return version, binPath
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
  .tmswitch.toml                  TOML file with optional 'version' and 'bin' keys
  .tmswitchrc                     Plain text file with a version string
  .terramate-version              Plain text file with a version string (tfenv-style)

`)
}
