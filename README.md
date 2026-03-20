# tmswitch

[![Build Status](https://github.com/devopshouse/tmswitch/actions/workflows/build.yml/badge.svg)](https://github.com/devopshouse/tmswitch/actions/workflows/build.yml)
[![Super-Linter](https://github.com/devopshouse/tmswitch/actions/workflows/super-linter.yml/badge.svg)](https://github.com/marketplace/actions/super-linter)
[![GitHub Release](https://img.shields.io/github/v/release/devopshouse/tmswitch)](https://github.com/devopshouse/tmswitch/releases/latest)
[![Release Asset Downloads](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/devopshouse/tmswitch/main/.github/badges/release-asset-downloads.json)](https://github.com/devopshouse/tmswitch/releases)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/devopshouse/tmswitch)
[![Go Report Card](https://img.shields.io/badge/dynamic/regex?url=https%3A%2F%2Fgoreportcard.com%2Freport%2Fgithub.com%2Fdevopshouse%2Ftmswitch&search=%22grade%22%3A%22(%5B%5E%22%5D%2B)%22&replace=%241&label=go%20report&color=brightgreen&cacheSeconds=3600)](https://goreportcard.com/report/github.com/devopshouse/tmswitch)

## Terramate Switcher

`tmswitch` is a command-line tool that lets you switch between different versions of
[Terramate](https://terramate.io/). If the requested version is not already installed,
`tmswitch` downloads it automatically from the official
[GitHub Releases](https://github.com/terramate-io/terramate/releases) page.

This project is based on ideas and behavior from:

- [warrensbox/terraform-switcher](https://github.com/warrensbox/terraform-switcher)
- [warrensbox/tgswitch](https://github.com/warrensbox/tgswitch)

License note:

- both upstream projects use the MIT License
- `tmswitch` also uses the MIT License
- see [THIRD_PARTY_NOTICES.md](/Users/xjulio/git/tmswitch/THIRD_PARTY_NOTICES.md) for attribution context

---

## Installation

### Linux / macOS (from source)

```bash
go install github.com/devopshouse/tmswitch@latest
```

### Homebrew

On macOS, install with:

```bash
brew tap devopshouse/tap
brew install --cask tmswitch
```

Or in a single command:

```bash
brew install --cask devopshouse/tap/tmswitch
```

Homebrew publishing uses a cask, which is the current GoReleaser-supported path for distributing prebuilt binaries.

Or build from source:

```bash
git clone https://github.com/devopshouse/tmswitch.git
cd tmswitch
go build -o tmswitch .
sudo mv tmswitch /usr/local/bin/
```

---

## How to use

### Interactive version selector

Run `tmswitch` with no arguments to get an interactive dropdown menu of available
Terramate versions. Use the arrow keys to navigate and **Enter** to confirm.

```bash
tmswitch
```

Recently used versions are shown at the top of the list for convenience.

### Supply version on the command line

```bash
tmswitch 0.16.0
```

### Include prerelease versions

Pass the `--pre` flag to also show release candidates and other prerelease versions:

```bash
tmswitch --pre
```

### Latest and implicit version helpers

```bash
tmswitch --latest
tmswitch --show-latest
tmswitch --latest-pre 0.17
tmswitch --show-latest-pre 0.17
tmswitch --latest-stable 0.16
tmswitch --show-latest-stable 0.16
tmswitch --list-all
```

These flags are modeled after the `tfswitch` UX:

- `--latest` installs the newest stable release
- `--show-latest` prints the newest stable release
- `--latest-pre <prefix>` installs the newest matching prerelease
- `--show-latest-pre <prefix>` prints the newest matching prerelease
- `--latest-stable <prefix>` installs the newest matching stable release
- `--show-latest-stable <prefix>` prints the newest matching stable release
- `--list-all` prints all available releases, including prereleases

### Use a version file

Create a `.tmswitchrc` or `.terramate-version` file in your project directory:

```bash
echo "0.16.0" > .tmswitchrc
tmswitch          # automatically installs and activates 0.16.0
```

### Use a TOML configuration file

Create a `.tmswitch.toml` file to set a preferred version, fallback version, and/or a custom binary path:

```toml
bin             = "/home/user/bin/terramate"
version         = "0.16.0"
default-version = "0.15.0"
```

`version` is used when no CLI arg or version file is present. `default-version` is only a fallback when no other version source is found.

### Environment overrides

`tmswitch` also supports `tfswitch`-style environment overrides:

```bash
export TM_VERSION=0.16.0
export TM_DEFAULT_VERSION=0.15.0
export TM_BINARY_PATH="$HOME/bin/terramate"
```

Precedence is:

1. CLI argument
2. `TM_VERSION`
3. `.tmswitchrc`
4. `.terramate-version`
5. `.tmswitch.toml` `version`
6. `TM_DEFAULT_VERSION`
7. `.tmswitch.toml` `default-version`
8. Interactive selector

### Custom binary path

```bash
tmswitch --bin /home/user/bin/terramate 0.16.0
```

### Operational flags

```bash
tmswitch --dry-run --latest
tmswitch --install "$HOME/custom-tools" --latest
tmswitch --arch amd64 --latest
tmswitch --mirror https://github.com/terramate-io/terramate --latest
```

- `--dry-run` shows what would happen without downloading or switching
- `--install <dir>` changes the root used for `.terramate.versions`
- `--arch <arch>` overrides the CPU architecture used for downloads
- `--mirror <url>` overrides the GitHub repo/API/download base used for version lookup and downloads
- `--default <version>` sets the CLI fallback version if no other source resolved one

---

## Options

| Flag | Short | Description |
| ---- | ----- | ----------- |
| `--bin <path>` | `-b` | Custom install path for the terramate symlink |
| `--chdir <dir>` | `-c` | Look for version/config files in a different directory |
| `--pre` | `-p` | Include prerelease versions in the selector |
| `--version` | `-v` | Display tmswitch version |
| `--help` | `-h` | Display help message |

Environment variables:

| Variable | Description |
| -------- | ----------- |
| `TM_VERSION` | Explicit Terramate version override |
| `TM_DEFAULT_VERSION` | Fallback Terramate version |
| `TM_BINARY_PATH` | Override binary install path unless `--bin` is provided |

Additional tfswitch-style flags:

| Flag | Description |
| ---- | ----------- |
| `--latest` | Install latest stable version |
| `--show-latest` | Print latest stable version |
| `--latest-pre <prefix>` | Install latest matching prerelease |
| `--show-latest-pre <prefix>` | Print latest matching prerelease |
| `--latest-stable <prefix>` | Install latest matching stable version |
| `--show-latest-stable <prefix>` | Print latest matching stable version |
| `--list-all` | Print all versions, including prereleases |
| `--default <version>` | Fallback version when no other source exists |
| `--dry-run` | Show actions without downloading or switching |
| `--install <dir>` | Root path used for `.terramate.versions` |
| `--arch <arch>` | Override download architecture |
| `--mirror <url>` | Override GitHub repo/API/download base |
| `--product terramate` | Product selector placeholder for tfswitch parity |
| `--force-color` | Force color output if terminal supports it |
| `--no-color` | Disable color output |
| `--log-level <level>` | Set log level (`OFF` suppresses stdlib logs) |

---

## How it works

1. Fetches the list of available Terramate versions from the GitHub Releases API.
2. Prompts you to select a version (or uses the version specified via flag/file).
3. Downloads the versioned binary from GitHub and stores it in `~/.terramate.versions/`.
4. Creates (or updates) a symlink at `/usr/local/bin/terramate` (or your custom path).

Previously downloaded versions are cached locally and reused without redownloading.

---

## License

MIT

## Release automation

This repository is wired for automated release PRs, GitHub Releases, and a Homebrew tap.

Required setup:

1. Create the tap repository `devopshouse/homebrew-tap`.
2. Create a GitHub token with write access to that repository.
3. Save it in this repository as the `TAP_GITHUB_TOKEN` Actions secret.
4. Create a GitHub token with write access to `devopshouse/tmswitch`.
5. Save it in this repository as the `RELEASE_PLEASE_TOKEN` Actions secret.

The automation now works like this:

1. [.github/workflows/release-please.yml](/Users/xjulio/git/tmswitch/.github/workflows/release-please.yml) runs on every push to `main`.
2. It opens or updates a release PR with the next version and changelog.
3. When that release PR is merged, `release-please` creates the next `v*` tag automatically.
4. That tag triggers [.github/workflows/release.yml](/Users/xjulio/git/tmswitch/.github/workflows/release.yml).
5. The release workflow runs the test suite, builds release archives, publishes binaries, and updates the Homebrew cask in `devopshouse/homebrew-tap`.

Notes:

- `release-please` works best with Conventional Commits in merged commits or squash-merge PR titles, such as `fix: handle brew cask resolution` or `feat: add latest-stable selector`.
- If `RELEASE_PLEASE_TOKEN` is not set, the workflow falls back to `GITHUB_TOKEN`. That is enough to manage the release PR, but it may not trigger downstream workflows from the created tag.
- The Homebrew publishing uses `homebrew_casks` in [.goreleaser.yaml](/Users/xjulio/git/tmswitch/.goreleaser.yaml), which is the current GoReleaser-supported replacement for deprecated `brews`.
- The Homebrew install command is `brew install --cask devopshouse/tap/tmswitch`.
