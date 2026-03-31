package lib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidVersionFormat(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"0.16.0", true},
		{"1.0.0", true},
		{"0.16.0-rc1", true},
		{"0.16.0-beta1", true},
		{"v0.16.0", false},  // has leading 'v'
		{"0.16", false},     // only two parts
		{"abc", false},      // not numeric
		{"0.16.0.1", false}, // four parts
		{"", false},         // empty
	}

	for _, tt := range tests {
		got := ValidVersionFormat(tt.version)
		if got != tt.valid {
			t.Errorf("ValidVersionFormat(%q) = %v, want %v", tt.version, got, tt.valid)
		}
	}
}

func TestVersionExist(t *testing.T) {
	list := []string{"0.16.0", "0.15.5", "0.15.4"}

	if !VersionExist("0.16.0", list) {
		t.Error("expected 0.16.0 to exist")
	}
	if VersionExist("0.14.0", list) {
		t.Error("expected 0.14.0 to not exist")
	}
}

func TestGetVersionListFromURL(t *testing.T) {
	releases := []githubRelease{
		{TagName: "v0.16.0", Prerelease: false, Draft: false},
		{TagName: "v0.16.0-rc1", Prerelease: true, Draft: false},
		{TagName: "v0.15.5", Prerelease: false, Draft: false},
		{TagName: "v0.15.4", Prerelease: false, Draft: true}, // should be excluded
	}
	body, err := json.Marshal(releases)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, writeErr := w.Write(body); writeErr != nil {
			return
		}
	}))
	defer srv.Close()

	// Without pre-releases.
	versions, err := getVersionListFromURL(srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("expected 2 stable versions, got %d: %v", len(versions), versions)
	}
	if versions[0] != "0.16.0" || versions[1] != "0.15.5" {
		t.Errorf("unexpected versions: %v", versions)
	}

	// With pre-releases.
	versions, err = getVersionListFromURL(srv.URL, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 3 {
		t.Errorf("expected 3 versions (including pre-release), got %d: %v", len(versions), versions)
	}
}

func TestGetVersionListFromURL_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := getVersionListFromURL(srv.URL, false)
	if err == nil {
		t.Error("expected an error on non-200 status, got nil")
	}
}

func TestLatestVersion(t *testing.T) {
	got, err := LatestVersion([]string{"0.16.0", "0.15.5"})
	if err != nil {
		t.Fatalf("LatestVersion: %v", err)
	}
	if got != "0.16.0" {
		t.Fatalf("expected 0.16.0, got %q", got)
	}
}

func TestLatestMatchingVersion(t *testing.T) {
	got, err := LatestMatchingVersion([]string{"0.16.1", "0.16.0", "0.15.9"}, "0.16")
	if err != nil {
		t.Fatalf("LatestMatchingVersion: %v", err)
	}
	if got != "0.16.1" {
		t.Fatalf("expected 0.16.1, got %q", got)
	}
}

func TestLatestMatchingVersion_Prerelease(t *testing.T) {
	got, err := LatestMatchingVersion([]string{"0.17.0-rc2", "0.17.0-rc1", "0.16.0"}, "0.17")
	if err != nil {
		t.Fatalf("LatestMatchingVersion: %v", err)
	}
	if got != "0.17.0-rc2" {
		t.Fatalf("expected 0.17.0-rc2, got %q", got)
	}
}
