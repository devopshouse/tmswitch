package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const (
	// githubReleasesURL is the GitHub Releases API endpoint for terramate.
	githubReleasesURL = "https://api.github.com/repos/terramate-io/terramate/releases?per_page=100"
)

// githubRelease represents a single release from the GitHub Releases API.
type githubRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

// semverRegex matches valid semantic version strings (e.g. "0.16.0", "0.16.0-rc1").
var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?$`)

// ValidVersionFormat returns true if the given string looks like a valid semver.
func ValidVersionFormat(version string) bool {
	return semverRegex.MatchString(version)
}

// GetVersionList fetches the list of available terramate versions from the
// GitHub Releases API and returns them sorted newest-first.
// Set includePrerelease to true to also include pre-release / RC versions.
func GetVersionList(includePrerelease bool) ([]string, error) {
	return getVersionListFromURL(githubReleasesURL, includePrerelease)
}

// getVersionListFromURL is the testable implementation of GetVersionList.
func getVersionListFromURL(url string, includePrerelease bool) ([]string, error) {
	resp, err := http.Get(url) // #nosec G107
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching releases", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases JSON: %w", err)
	}

	var versions []string
	for _, r := range releases {
		if r.Draft {
			continue
		}
		if r.Prerelease && !includePrerelease {
			continue
		}
		// Strip the leading "v" from the tag name.
		v := strings.TrimPrefix(r.TagName, "v")
		if ValidVersionFormat(v) {
			versions = append(versions, v)
		}
	}
	return versions, nil
}

// VersionExist returns true if the given version is present in the list.
func VersionExist(version string, list []string) bool {
	for _, v := range list {
		if v == version {
			return true
		}
	}
	return false
}
