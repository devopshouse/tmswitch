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

Create a `.tmswitch.toml` file to set a default version and/or a custom binary path:

```toml
bin     = "/home/user/bin/terramate"
version = "0.16.0"
```

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