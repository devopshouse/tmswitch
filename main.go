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
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pborman/getopt"
	"github.com/spf13/viper"

	lib "github.com/devopshouse/tmswitch/lib"
)

const (
	defaultBin     = "/usr/local/bin/terramate"
	rcFilename     = ".tmswitchrc"
	tmvFilename    = ".terramate-version"
	tomlFilename   = ".tmswitch.toml"
	envVersion     = "TM_VERSION"
	envDefault     = "TM_DEFAULT_VERSION"
	envBinPath     = "TM_BINARY_PATH"
	defaultProduct = "terramate"
)

var appVersion = "dev"
var currentInstallOptions lib.InstallOptions
var currentReleaseAPIURL string

func main() {
	dir := lib.GetCurrentDirectory()

	archFlag := getopt.StringLong("arch", 'A', runtime.GOARCH, "Override CPU architecture type for downloaded binary")
	custBinPath := getopt.StringLong("bin", 'b',
		lib.ConvertExecutableExt(defaultBin),
		"Custom binary path. Ex: tmswitch -b /home/user/bin/terramate")
	defaultVersionFlag := getopt.StringLong("default", 'd', "", "Default to this version in case no other versions could be detected")
	logLevelFlag := getopt.StringLong("log-level", 'g', "INFO", "Set tmswitch logging level")
	installPathFlag := getopt.StringLong("install", 'i', "", "Custom install path")
	forceColorFlag := getopt.BoolLong("force-color", 'K', "Force color output if terminal supports it")
	noColorFlag := getopt.BoolLong("no-color", 'k', "Disable color output")
	latestFlag := getopt.BoolLong("latest", 'u', "Get latest stable version")
	listAllFlag := getopt.BoolLong("list-all", 'l', "List all versions, including beta and RC versions")
	showLatestFlag := getopt.BoolLong("show-latest", 'U', "Show latest stable version")
	latestPreFlag := getopt.StringLong("latest-pre", 'p', "", "Latest pre-release implicit version")
	showLatestPreFlag := getopt.StringLong("show-latest-pre", 'P', "", "Show latest pre-release implicit version")
	latestStableFlag := getopt.StringLong("latest-stable", 's', "", "Latest implicit stable version based on a prefix")
	showLatestStableFlag := getopt.StringLong("show-latest-stable", 'S', "", "Show latest implicit stable version based on a prefix")
	dryRunFlag := getopt.BoolLong("dry-run", 'r', "Only show what tmswitch would do. Don't download anything")
	mirrorFlag := getopt.StringLong("mirror", 'm', "", "Install from a remote API other than the default")
	productFlag := getopt.StringLong("product", 't', defaultProduct, "Specify which product to use")
	versionFlag := getopt.BoolLong("version", 'v', "Display tmswitch version")
	helpFlag := getopt.BoolLong("help", 'h', "Display help message")
	chDirPath := getopt.StringLong("chdir", 'c', dir,
		"Switch to a different working directory before executing")
	preFlag := getopt.BoolLong("pre", 0, "Include pre-release versions")

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

	configureCLI(*logLevelFlag, *forceColorFlag, *noColorFlag, *productFlag)

	releaseAPIURL, downloadBaseURL, err := deriveMirrorURLs(*mirrorFlag)
	if err != nil {
		log.Fatalf("Invalid mirror value: %v", err)
	}
	currentReleaseAPIURL = releaseAPIURL
	currentInstallOptions = lib.InstallOptions{
		Arch:            *archFlag,
		BinPath:         *custBinPath,
		DownloadBaseURL: downloadBaseURL,
		DryRun:          *dryRunFlag,
		InstallPath:     *installPathFlag,
	}

	homedir := lib.GetHomeDirectory()
	if handleImmediateActions(immediateActions{
		binPath:          *custBinPath,
		listAll:          *listAllFlag,
		showLatest:       *showLatestFlag,
		latestPre:        *latestPreFlag,
		showLatestPre:    *showLatestPreFlag,
		latestStable:     *latestStableFlag,
		showLatestStable: *showLatestStableFlag,
	}) {
		return
	}

	version, binPath, interactive := resolveInstallRequestWithDefault(args, *chDirPath, homedir, *custBinPath, *defaultVersionFlag)
	if *latestFlag {
		latest, err := latestVersion(false)
		if err != nil {
			log.Fatalf("Failed to resolve latest terramate version: %v", err)
		}
		installVersion(latest, &binPath)
		return
	}
	if interactive {
		selectVersionInteractive(&binPath, *preFlag)
		return
	}
	installVersion(version, &binPath)
}

type immediateActions struct {
	binPath          string
	listAll          bool
	showLatest       bool
	latestPre        string
	showLatestPre    string
	latestStable     string
	showLatestStable string
}

func handleImmediateActions(actions immediateActions) bool {
	switch {
	case actions.listAll:
		printVersionList(true)
		return true
	case actions.showLatest:
		version, err := latestVersion(false)
		printResolvedVersion(version, err, "latest terramate")
		return true
	case actions.latestPre != "":
		version, err := latestMatchingVersion(actions.latestPre, true)
		installResolvedVersion(actions.binPath, version, err, "latest pre-release terramate")
		return true
	case actions.showLatestPre != "":
		version, err := latestMatchingVersion(actions.showLatestPre, true)
		printResolvedVersion(version, err, "latest pre-release terramate")
		return true
	case actions.latestStable != "":
		version, err := latestMatchingVersion(actions.latestStable, false)
		installResolvedVersion(actions.binPath, version, err, "latest stable terramate")
		return true
	case actions.showLatestStable != "":
		version, err := latestMatchingVersion(actions.showLatestStable, false)
		printResolvedVersion(version, err, "latest stable terramate")
		return true
	default:
		return false
	}
}

func printResolvedVersion(version string, err error, label string) {
	if err != nil {
		log.Fatalf("Failed to resolve %s version: %v", label, err)
	}
	fmt.Println(version)
}

func installResolvedVersion(binPath string, version string, err error, label string) {
	if err != nil {
		log.Fatalf("Failed to resolve %s version: %v", label, err)
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
	return resolveInstallRequestWithDefault(args, chDirPath, homedir, cliBinPath, "")
}

func resolveInstallRequestWithDefault(args []string, chDirPath, homedir, cliBinPath, cliDefaultVersion string) (string, string, bool) {
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
	if cfg.BinPath != "" {
		binPath = cfg.BinPath
	}

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
	case strings.TrimSpace(cliDefaultVersion) != "":
		return strings.TrimSpace(cliDefaultVersion), binPath, false
	case envDefaultVersion != "":
		return envDefaultVersion, binPath, false
	case cfg.DefaultVersion != "":
		return cfg.DefaultVersion, binPath, false
	default:
		return "", binPath, true
	}
}

func installVersion(version string, binPath *string) {
	version, warnErr, fatalErr := validateRequestedVersion(version, versionListFetcher())
	if fatalErr != nil {
		fmt.Println(fatalErr.Error())
		usageMessage()
		os.Exit(1)
	}
	if warnErr != nil {
		fmt.Printf("Warning: could not fetch releases list (%v). Attempting install anyway.\n", warnErr)
	}
	options := currentInstallOptions
	options.BinPath = *binPath
	lib.InstallWithOptions(version, options)
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

func mustGetVersionList(includePrerelease bool) []string {
	versions, err := versionListFetcher()(includePrerelease)
	if err != nil {
		log.Fatalf("Failed to fetch version list: %v", err)
	}
	if len(versions) == 0 {
		log.Fatal("No terramate versions found.")
	}
	return versions
}

func printVersionList(includePrerelease bool) {
	for _, version := range mustGetVersionList(includePrerelease) {
		fmt.Println(version)
	}
}

func latestVersion(includePrerelease bool) (string, error) {
	return lib.LatestVersion(mustGetVersionList(includePrerelease))
}

func latestMatchingVersion(requested string, includePrerelease bool) (string, error) {
	return lib.LatestMatchingVersion(mustGetVersionList(includePrerelease), requested)
}

func versionListFetcher() func(bool) ([]string, error) {
	return func(includePrerelease bool) ([]string, error) {
		if currentReleaseAPIURL != "" {
			return lib.GetVersionListFromURL(currentReleaseAPIURL, includePrerelease)
		}
		return lib.GetVersionList(includePrerelease)
	}
}

// selectVersionInteractive shows an interactive prompt to pick a version.
func selectVersionInteractive(binPath *string, includePrerelease bool) {
	versions, err := versionListFetcher()(includePrerelease)
	if err != nil {
		log.Fatalf("Failed to fetch version list: %v", err)
	}
	if len(versions) == 0 {
		fmt.Println("No terramate versions found.")
		os.Exit(1)
	}

	// Prepend recently used versions for convenience.
	recent, err := lib.GetRecentVersions()
	if err != nil {
		log.Printf("Warning: failed to load recent terramate versions: %v", err)
	}
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
	options := currentInstallOptions
	options.BinPath = *binPath
	lib.InstallWithOptions(version, options)
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

func configureCLI(logLevel string, forceColor, noColor bool, product string) {
	if forceColor && noColor {
		log.Fatal("Cannot force color and disable color at the same time")
	}
	if forceColor {
		os.Setenv("FORCE_COLOR", "true")
	}
	if noColor {
		os.Setenv("NO_COLOR", "true")
	}

	if strings.EqualFold(logLevel, "OFF") {
		log.SetOutput(io.Discard)
	}

	if product == "" {
		return
	}
	if product != defaultProduct {
		log.Fatalf("Unsupported product %q. Only %q is currently supported.", product, defaultProduct)
	}
}

func deriveMirrorURLs(mirror string) (string, string, error) {
	mirror = strings.TrimSpace(mirror)
	if mirror == "" {
		return "", "", nil
	}

	switch {
	case strings.Contains(mirror, "api.github.com/repos/") && strings.Contains(mirror, "/releases"):
		base := strings.TrimPrefix(mirror, "https://api.github.com/repos/")
		base = strings.SplitN(base, "/releases", 2)[0]
		return mirror, "https://github.com/" + base + "/releases/download/", nil
	case strings.Contains(mirror, "github.com/") && strings.Contains(mirror, "/releases/download/"):
		base := strings.TrimPrefix(mirror, "https://github.com/")
		base = strings.SplitN(base, "/releases/download/", 2)[0]
		return "https://api.github.com/repos/" + base + "/releases?per_page=100", mirror, nil
	case strings.Contains(mirror, "github.com/"):
		base := strings.TrimPrefix(mirror, "https://github.com/")
		base = strings.TrimSuffix(base, "/")
		return "https://api.github.com/repos/" + base + "/releases?per_page=100", "https://github.com/" + base + "/releases/download/", nil
	default:
		return "", "", fmt.Errorf("unsupported mirror %q: expected a GitHub repository, API releases URL, or releases/download base URL", mirror)
	}
}

func usageMessage() {
	fmt.Print(`
tmswitch - Switch between different versions of terramate

Usage:
  tmswitch                        Interactive version selector
  tmswitch <version>              Install a specific version (e.g. tmswitch 0.16.0)
  tmswitch --latest               Install the latest stable version
  tmswitch --show-latest          Print the latest stable version
  tmswitch --latest-pre <prefix>  Install the latest matching pre-release version
  tmswitch --show-latest-pre <p>  Print the latest matching pre-release version
  tmswitch --latest-stable <p>    Install the latest matching stable version
  tmswitch --show-latest-stable <p>
                                   Print the latest matching stable version
  tmswitch --list-all             Print all versions, including pre-releases
  tmswitch --pre                  Include pre-release versions in the selector
  tmswitch --dry-run              Only show what tmswitch would do
  tmswitch --default <version>    Fallback version when no other source exists
  tmswitch --arch <arch>          Override CPU architecture for downloads
  tmswitch --bin <path>           Custom install path for the terramate binary
  tmswitch --chdir <dir>          Look for version files in a different directory
  tmswitch --install <dir>        Root path for .terramate.versions
  tmswitch --mirror <url>         GitHub repo/API/download base override
  tmswitch --product terramate    Product selector (currently terramate only)
  tmswitch --force-color          Force color output if terminal supports it
  tmswitch --no-color             Disable color output
  tmswitch --log-level <level>    Set log level (OFF suppresses stdlib logs)
  tmswitch --version              Display tmswitch version
  tmswitch --help                 Display this help message

Version files (in order of precedence):
  TM_VERSION                      Environment override for the Terramate version
  .tmswitchrc                     Plain text file with a version string
  .terramate-version              Plain text file with a version string (tfenv-style)
  TM_DEFAULT_VERSION              Environment fallback version
  .tmswitch.toml                  TOML file with optional 'version', 'default-version', and 'bin' keys
  TM_BINARY_PATH                  Environment override for the Terramate binary path

Flags:
  -A, --arch <arch>               Override CPU architecture for downloads
  -d, --default <version>         Fallback version when no other source exists
  -g, --log-level <level>         Set log level (OFF disables stdlib logs)
  -l, --list-all                  List all versions, including beta and RC versions
  -m, --mirror <url>              Override GitHub repo/API/download base
  -p, --latest-pre <prefix>       Install latest matching pre-release version
  -P, --show-latest-pre <prefix>  Print latest matching pre-release version
  -r, --dry-run                   Only show what tmswitch would do
  -s, --latest-stable <prefix>    Install latest matching stable version
  -S, --show-latest-stable <p>    Print latest matching stable version
  -t, --product terramate         Product selector (currently terramate only)
  -u, --latest                    Get latest stable version
  -U, --show-latest               Show latest stable version
  -i, --install <dir>             Root path for .terramate.versions
  -K, --force-color               Force color output if terminal supports it
  -k, --no-color                  Disable color output

`)
}
