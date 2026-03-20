# tmswitch

[![Go Report Card](https://goreportcard.com/badge/github.com/devopshouse/tmswitch)](https://goreportcard.com/report/github.com/devopshouse/tmswitch)

## Terramate Switcher

`tmswitch` is a command-line tool that lets you switch between different versions of
[Terramate](https://terramate.io/). If the requested version is not already installed,
`tmswitch` downloads it automatically from the official
[GitHub Releases](https://github.com/terramate-io/terramate/releases) page.

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

### Include pre-release versions

Pass the `--pre` flag to also show release candidates and other pre-release versions:

```bash
tmswitch --pre
```

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

---

## Options

| Flag | Short | Description |
|------|-------|-------------|
| `--bin <path>` | `-b` | Custom install path for the terramate symlink |
| `--chdir <dir>` | `-c` | Look for version/config files in a different directory |
| `--pre` | `-p` | Include pre-release versions in the selector |
| `--version` | `-v` | Display tmswitch version |
| `--help` | `-h` | Display help message |

Environment variables:

| Variable | Description |
|----------|-------------|
| `TM_VERSION` | Explicit Terramate version override |
| `TM_DEFAULT_VERSION` | Fallback Terramate version |
| `TM_BINARY_PATH` | Override binary install path unless `--bin` is provided |

---

## How it works

1. Fetches the list of available Terramate versions from the GitHub Releases API.
2. Prompts you to select a version (or uses the version specified via flag/file).
3. Downloads the versioned binary from GitHub and stores it in `~/.terramate.versions/`.
4. Creates (or updates) a symlink at `/usr/local/bin/terramate` (or your custom path).

Previously downloaded versions are cached locally and re-used without re-downloading.

---

## License

MIT

## Release automation

This repository is wired for GitHub Releases plus a Homebrew tap through GoReleaser.

Required setup:

1. Create the tap repository `devopshouse/homebrew-tap`.
2. Create a GitHub token with write access to that repository.
3. Save it in this repository as the `TAP_GITHUB_TOKEN` Actions secret.
4. Push a tag such as `v0.1.0`.

The release workflow in [.github/workflows/release.yml](/Users/xjulio/git/tmswitch/.github/workflows/release.yml) will:

1. Run the test suite.
2. Build release archives for macOS and Linux.
3. Publish a GitHub Release.
4. Update the Homebrew cask in `devopshouse/homebrew-tap`.

Notes:

- The Homebrew publishing uses `homebrew_casks` in [.goreleaser.yaml](/Users/xjulio/git/tmswitch/.goreleaser.yaml), which is the current GoReleaser-supported replacement for deprecated `brews`.
- The Homebrew install command is `brew install --cask devopshouse/tap/tmswitch`.
